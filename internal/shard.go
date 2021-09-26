package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/WelcomerTeam/RealRock/limiter"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/czlib"
	"github.com/rs/zerolog"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	"go.uber.org/atomic"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

const (
	WebsocketReadLimit          = 512 << 20
	WebsocketReconnectCloseCode = 4000

	// Time required before warning about an event taking too long.
	DispatchWarningTimeout = 30 * time.Second

	MessageChannelBuffer      = 64
	MinPayloadCompressionSize = 1000000 // Applies higher compression to payloads larger than this in bytes

	// Time necessary to abort chunking when no events have been received yet in this time frame.
	InitialMemberChunkTimeout = 10 * time.Second
	// Time necessary to mark chunking as completed when no more events are received in this time frame.
	MemberChunkTimeout = 1 * time.Second
	// Time between chunks no longer marked as chunked anymore.
	ChunkStatePersistTimeout = 10 * time.Second

	// Number of retries attempted before considering a shard not working.
	ShardConnectRetries = 3

	ShardWSRateLimit      = 118
	GatewayLargeThreshold = 100

	FirstEventTimeout = 5 * time.Second

	WaitForReadyTimeout = 15 * time.Second
	ReadyTimeout        = 2 * time.Second

	MaxReconnectWait = 60 * time.Second

	WriteJSONRetry = 1 * time.Second
)

// Shard represents the shard object.
type Shard struct {
	ctx    context.Context `json:"-"`
	cancel func()

	RoutineDeadSignal   DeadSignal `json:"-"`
	HeartbeatDeadSignal DeadSignal `json:"-"`

	Start            time.Time     `json:"start"`
	RetriesRemaining *atomic.Int32 `json:"-"`

	Logger zerolog.Logger `json:"-"`

	ShardID int `json:"shard_id"`

	Sandwich   *Sandwich   `json:"-"`
	Manager    *Manager    `json:"-"`
	ShardGroup *ShardGroup `json:"-"`

	HeartbeatActive   *atomic.Bool `json:"-"`
	LastHeartbeatAck  *atomic.Time `json:"-"`
	LastHeartbeatSent *atomic.Time `json:"-"`

	Heartbeater       *time.Ticker  `json:"-"`
	HeartbeatInterval time.Duration `json:"-"`

	// Duration since last heartbeat Ack beforereconnecting.
	HeartbeatFailureInterval time.Duration `json:"-"`

	unavailableMu sync.RWMutex               `json:"-"`
	Unavailable   map[discord.Snowflake]bool `json:"-"`

	// Stores a local list of all guilds in the shard.d
	guildsMu sync.RWMutex               `json:"-"`
	Guilds   map[discord.Snowflake]bool `json:"-"`

	channelMu sync.RWMutex
	MessageCh chan discord.GatewayPayload
	ErrorCh   chan error

	Sequence  *atomic.Int64
	SessionID *atomic.String

	wsConnMu sync.RWMutex
	wsConn   *websocket.Conn

	wsRatelimit *limiter.DurationLimiter

	ready chan void
}

// NewShard creates a new shard object.
func (sg *ShardGroup) NewShard(shardID int) (sh *Shard) {
	logger := sg.Logger.With().Int("shardId", shardID).Logger()
	sh = &Shard{
		RoutineDeadSignal:   DeadSignal{},
		HeartbeatDeadSignal: DeadSignal{},

		RetriesRemaining: atomic.NewInt32(ShardConnectRetries),

		Logger: logger,

		ShardID: shardID,

		Sandwich:   sg.Manager.Sandwich,
		Manager:    sg.Manager,
		ShardGroup: sg,

		HeartbeatActive:   atomic.NewBool(false),
		LastHeartbeatAck:  &atomic.Time{},
		LastHeartbeatSent: &atomic.Time{},

		unavailableMu: sync.RWMutex{},
		Unavailable:   make(map[discord.Snowflake]bool),

		guildsMu: sync.RWMutex{},
		Guilds:   make(map[discord.Snowflake]bool),

		channelMu: sync.RWMutex{},

		Sequence:  &atomic.Int64{},
		SessionID: &atomic.String{},

		wsConnMu: sync.RWMutex{},

		// We use 118 just to allow heartbeating to not be limited
		// by WS but not use it itself.
		wsRatelimit: limiter.NewDurationLimiter(ShardWSRateLimit, time.Minute),

		ready: make(chan void, 1),
	}

	sh.ctx, sh.cancel = context.WithCancel(sg.Manager.ctx)

	return sh
}

// Open starts listening to the gateway.
func (sh *Shard) Open() {
	sh.Logger.Debug().Msg("Started listening to shard")

	for {
		err := sh.Listen(sh.ctx)
		if xerrors.Is(err, context.Canceled) {
			sh.Logger.Debug().Msg("Shard context canceled")

			return
		}

		select {
		case <-sh.RoutineDeadSignal.Dead():
			return
		case <-sh.ctx.Done():
			return
		default:
		}
	}
}

// Connect connects to the gateway and handles identifying.
func (sh *Shard) Connect() (err error) {
	sh.Logger.Debug().Msg("Connecting shard")

	// Empty ready channel.
readyConsumer:
	for {
		select {
		case <-sh.ready:
		default:
			break readyConsumer
		}
	}

	select {
	case <-sh.ctx.Done():
	default:
		sh.cancel()
	}

	sh.RoutineDeadSignal.Close("CONNECT")
	sh.RoutineDeadSignal.Revive()

	sh.HeartbeatDeadSignal.Close("HB")
	sh.HeartbeatDeadSignal.Revive()

	sh.ctx, sh.cancel = context.WithCancel(sh.Manager.ctx)

	sh.Manager.gatewayMu.RLock()
	gatewayURL := sh.Manager.Gateway.URL
	sh.Manager.gatewayMu.RUnlock()

	defer func() {
		if err != nil && sh.hasWsConn() {
			sh.CloseWS(websocket.StatusNormalClosure)
		}
	}()

	if !sh.hasWsConn() {
		errorCh, messageCh, err := sh.FeedWebsocket(sh.ctx, gatewayURL, nil)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to dial gateway")

			go sh.Sandwich.PublishSimpleWebhook(
				fmt.Sprintf("Failed to dial `%s`", gatewayURL),
				"`"+err.Error()+"`",
				fmt.Sprintf(
					"Manager: %s ShardGroup: %d ShardID: %d/%d",
					sh.Manager.Configuration.Identifier,
					sh.ShardGroup.ID,
					sh.ShardID,
					sh.ShardGroup.ShardCount,
				),
				EmbedColourDanger,
			)

			return err
		}

		sh.channelMu.Lock()
		sh.ErrorCh = errorCh
		sh.MessageCh = messageCh
		sh.channelMu.Unlock()
	} else {
		sh.Logger.Info().Msg("Reusing websocket connection")
	}

	sh.Logger.Trace().Msg("Reading from gateway")

	// Read a message from Gateway, this should be Hello
	msg, err := sh.readMessage()
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to read message")

		return
	}

	var helloResponse discord.Hello

	err = sh.decodeContent(msg, &helloResponse)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to decode HELLO event")

		return
	}

	now := time.Now().UTC()

	sh.Start = now
	sh.LastHeartbeatAck.Store(now)
	sh.LastHeartbeatSent.Store(now)

	sh.HeartbeatInterval = helloResponse.HeartbeatInterval * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures
	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

	go sh.Heartbeat(sh.ctx)

	sequence := sh.Sequence.Load()
	sessionID := sh.SessionID.Load()

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Int64("sequence", sequence).
		Msg("Received HELLO event")

	if sessionID == "" || sequence == 0 {
		err = sh.Identify(sh.ctx)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to identify")

			go sh.Sandwich.PublishSimpleWebhook(
				"Failed to Identify",
				"`"+err.Error()+"`",
				fmt.Sprintf(
					"Manager: %s ShardGroup: %d ShardID: %d/%d",
					sh.Manager.Configuration.Identifier,
					sh.ShardGroup.ID,
					sh.ShardID,
					sh.ShardGroup.ShardCount,
				),
				EmbedColourDanger,
			)

			return
		}
	} else {
		err = sh.Resume(sh.ctx)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to resume")

			go sh.Sandwich.PublishSimpleWebhook(
				"Failed to Resume",
				"`"+err.Error()+"`",
				fmt.Sprintf(
					"Manager: %s ShardGroup: %d ShardID: %d/%d",
					sh.Manager.Configuration.Identifier,
					sh.ShardGroup.ID,
					sh.ShardID,
					sh.ShardGroup.ShardCount,
				),
				EmbedColourDanger,
			)

			return
		}

		// We can assume the bot is now connected to discord.
	}

	t := time.NewTicker(FirstEventTimeout)
	defer t.Stop()

	// We wait until we either receive a first event, error
	// or we hit our FirstEventTimeout. We do nothing when
	// hitting the FirstEventtimeout.

	sh.Logger.Trace().Msg("Waiting for first event")

	sh.channelMu.RLock()
	defer sh.channelMu.RUnlock()

	sh.channelMu.RLock()
	errorCh := sh.ErrorCh
	messageCh := sh.MessageCh
	sh.channelMu.RUnlock()

	select {
	case err = <-errorCh:
		sh.Logger.Error().Err(err).Msg("Encountered error whilst connecting")

		go sh.Sandwich.PublishSimpleWebhook(
			"Encountered error during connection",
			"`"+err.Error()+"`",
			fmt.Sprintf(
				"Manager: %s ShardGroup: %d ShardID: %d/%d",
				sh.Manager.Configuration.Identifier,
				sh.ShardGroup.ID,
				sh.ShardID,
				sh.ShardGroup.ShardCount,
			),
			EmbedColourDanger,
		)

		return err
	case msg = <-messageCh:
		sh.Logger.Debug().Msgf("Received first event %d %s", msg.Op, msg.Type)

		messageCh <- msg
	case <-t.C:
	}

	return err
}

// Heartbeat maintains a heartbeat with discord.
func (sh *Shard) Heartbeat(ctx context.Context) {
	sh.HeartbeatActive.Store(true)
	defer sh.HeartbeatActive.Store(false)

	sh.HeartbeatDeadSignal.Started("HB")
	defer sh.HeartbeatDeadSignal.Done("HB")

	for {
		select {
		case <-sh.HeartbeatDeadSignal.Dead():
			return
		case <-ctx.Done():
			return
		case <-sh.Heartbeater.C:
			seq := sh.Sequence.Load()

			err := sh.SendEvent(ctx, discord.GatewayOpHeartbeat, seq)

			now := time.Now().UTC()
			sh.LastHeartbeatSent.Store(now)

			if err != nil || now.Sub(sh.LastHeartbeatAck.Load()) > sh.HeartbeatFailureInterval {
				if err != nil {
					sh.Logger.Error().Err(err).Msg("Failed to heartbeat. Reconnecting")

					go sh.Sandwich.PublishSimpleWebhook(
						"Failed to heartbeat. Reconnecting",
						"`"+err.Error()+"`",
						fmt.Sprintf(
							"Manager: %s ShardGroup: %d ShardID: %d/%d",
							sh.Manager.Configuration.Identifier,
							sh.ShardGroup.ID,
							sh.ShardID,
							sh.ShardGroup.ShardCount,
						),
						EmbedColourDanger,
					)
				} else {
					sh.Logger.Warn().Msg("Failed to ack and passed heartbeat failure interval")

					go sh.Sandwich.PublishSimpleWebhook(
						"Failed to ack and passed heartbeat failure interval",
						"",
						fmt.Sprintf(
							"Manager: %s ShardGroup: %d ShardID: %d/%d",
							sh.Manager.Configuration.Identifier,
							sh.ShardGroup.ID,
							sh.ShardID,
							sh.ShardGroup.ShardCount,
						),
						EmbedColourWarning,
					)
				}

				err = sh.Reconnect(websocket.StatusNormalClosure)
				if err != nil {
					sh.Logger.Error().Err(err).Msg("Failed to reconnect")
				}

				return
			}
		}
	}
}

// Listen to gateway and process accordingly.
func (sh *Shard) Listen(ctx context.Context) (err error) {
	sh.wsConnMu.RLock()
	wsConn := sh.wsConn
	sh.wsConnMu.RUnlock()

	for {
		select {
		case <-sh.RoutineDeadSignal.Dead():
			return
		case <-ctx.Done():
			return
		default:
		}

		msg, err := sh.readMessage()
		if err != nil {
			if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
				break
			}

			sh.Logger.Error().Err(err).Msg("Error reading from gateway")

			var closeError *websocket.CloseError

			if errors.As(err, &closeError) {
				// If possible, we will check the close error to determine if we can continue
				switch closeError.Code {
				case discord.CloseNotAuthenticated, // Not authenticated
					discord.CloseInvalidShard,      // Invalid shard
					discord.CloseShardingRequired,  // Sharding required
					discord.CloseInvalidAPIVersion, // Invalid API version
					discord.CloseInvalidIntents,    // Invalid Intent(s)
					discord.CloseDisallowedIntents: // Disallowed intent(s)
					sh.Logger.Error().Int("code", int(closeError.Code)).Msg("Shard received closure code")

					go sh.Sandwich.PublishSimpleWebhook(
						"Shard received closure code",
						"`"+strconv.Itoa(int(closeError.Code))+"` - `"+err.Error()+"`",
						fmt.Sprintf(
							"Manager: %s ShardGroup: %d ShardID: %d/%d",
							sh.Manager.Configuration.Identifier,
							sh.ShardGroup.ID,
							sh.ShardID,
							sh.ShardGroup.ShardCount,
						),
						EmbedColourWarning,
					)

					sh.ShardGroup.Error.Store(err.Error())

					return err
				default:
					sh.Logger.Warn().Msgf("Websocket was closed with code %d", closeError.Code)
				}
			}

			sh.wsConnMu.RLock()
			connEqual := wsConn == sh.wsConn
			sh.wsConnMu.RUnlock()

			if connEqual {
				// We have likely closed so we should attempt to reconnect
				sh.Logger.Warn().Msg("We have encountered an error whilst in the same connection. Reconnecting")
				err = sh.Reconnect(websocket.StatusNormalClosure)

				if err != nil {
					return err
				}

				return nil
			}

			sh.wsConnMu.RLock()
			wsConn = sh.wsConn
			sh.wsConnMu.RUnlock()
		}

		sh.OnEvent(ctx, msg)

		sh.wsConnMu.RLock()
		connNotEqual := wsConn != sh.wsConn
		sh.wsConnMu.RUnlock()

		// In the event we have reconnected, the wsConn could have changed,
		// we will use the new wsConn if this is the case
		if connNotEqual {
			sh.Logger.Debug().Msg("New wsConn was assigned to shard")

			sh.wsConnMu.RLock()
			wsConn = sh.wsConn
			sh.wsConnMu.RUnlock()
		}
	}

	return err
}

// FeedWebsocket reads websocket events and feeds them through a channel.
func (sh *Shard) FeedWebsocket(ctx context.Context, u string,
	opts *websocket.DialOptions) (errorCh chan error, messageCh chan discord.GatewayPayload, err error) {
	messageCh = make(chan discord.GatewayPayload, MessageChannelBuffer)
	errorCh = make(chan error, 1)

	conn, _, err := websocket.Dial(ctx, u, opts)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to dial websocket")

		return errorCh, messageCh, xerrors.Errorf("failed to connect to websocket: %w", err)
	}

	conn.SetReadLimit(WebsocketReadLimit)

	sh.wsConnMu.Lock()
	sh.wsConn = conn
	sh.wsConnMu.Unlock()

	go func() {
		sh.RoutineDeadSignal.Started("FEED")
		defer sh.RoutineDeadSignal.Done("FEED")

		for {
			messageType, data, connectionErr := conn.Read(ctx)

			select {
			case <-sh.RoutineDeadSignal.Dead():
				return
			case <-ctx.Done():
				return
			default:
			}

			sandwichEventCount.WithLabelValues(sh.Manager.Configuration.Identifier).Add(1)

			if connectionErr != nil {
				sh.Logger.Error().Err(connectionErr).Msg("Failed to read from gateway")
				errorCh <- connectionErr

				return
			}

			if messageType == websocket.MessageBinary {
				data, connectionErr = czlib.Decompress(data)
				if connectionErr != nil {
					sh.Logger.Error().Err(connectionErr).Msg("Failed to decompress data")
					errorCh <- connectionErr

					return
				}
			}

			msg := sh.Sandwich.receivedPool.Get().(*discord.GatewayPayload)

			connectionErr = json.Unmarshal(data, &msg)
			if connectionErr != nil {
				sh.Logger.Error().Err(connectionErr).Msg("Failed to unmarshal message")

				continue
			}

			// TODO Add Analytics

			messageCh <- *msg
		}
	}()

	return errorCh, messageCh, nil
}

// Identify sends the identify packet to discord.
func (sh *Shard) Identify(ctx context.Context) (err error) {
	sh.Manager.gatewayMu.Lock()
	sh.Manager.Gateway.SessionStartLimit.Remaining--
	sh.Manager.gatewayMu.Unlock()

	sh.Manager.WaitForIdentify(sh.ShardID, sh.ShardGroup.ShardCount)
	sh.Logger.Debug().Msg("Wait for identify completed")

	sh.Manager.configurationMu.RLock()
	token := sh.Manager.Configuration.Token
	presence := sh.Manager.Configuration.Bot.DefaultPresence
	intents := sh.Manager.Configuration.Bot.Intents
	sh.Manager.configurationMu.RUnlock()

	sh.Logger.Debug().Msg("Sending identify")

	err = sh.SendEvent(ctx, discord.GatewayOpIdentify, discord.Identify{
		Token: token,
		Properties: &discord.IdentifyProperties{
			OS:      runtime.GOOS,
			Browser: "Sandwich " + VERSION,
			Device:  "Sandwich " + VERSION,
		},
		Compress:       true,
		LargeThreshold: GatewayLargeThreshold,
		Shard:          [2]int{sh.ShardID, sh.ShardGroup.ShardCount},
		Presence:       presence,
		Intents:        intents,
	})

	return err
}

// Resume sends the resume packet to discord.
func (sh *Shard) Resume(ctx context.Context) (err error) {
	sh.Manager.configurationMu.RLock()
	token := sh.Manager.Configuration.Token
	sh.Manager.configurationMu.RUnlock()

	sh.Logger.Debug().Msg("Sending resume")

	err = sh.SendEvent(ctx, discord.GatewayOpResume, discord.Resume{
		Token:     token,
		SessionID: sh.SessionID.Load(),
		Sequence:  sh.Sequence.Load(),
	})

	return err
}

// SendEvent sends an event to discord.
func (sh *Shard) SendEvent(ctx context.Context, op discord.GatewayOp, data interface{}) (err error) {
	packet := sh.Sandwich.sentPool.Get().(*discord.SentPayload)
	defer sh.Sandwich.sentPool.Put(packet)

	packet.Op = op
	packet.Data = data

	err = sh.WriteJSON(ctx, op, packet)
	if err != nil {
		return xerrors.Errorf("sendEvent writeJson: %w", err)
	}

	return
}

// WriteJSON writes json data to the websocket.
func (sh *Shard) WriteJSON(ctx context.Context, op discord.GatewayOp, i interface{}) (err error) {
	// In very rare circumstances, we can be writing to the websocket whilst
	// context is being remade. We will recover and dismiss any SIGSEGVs that
	// are raised.
	defer func() {
		if r := recover(); r != nil {
			sh.Logger.Warn().Err(err).Msg("Recovered panic in WriteJSON")

			time.Sleep(WriteJSONRetry)

			err = sh.WriteJSON(ctx, op, i)
		}
	}()

	res, err := json.Marshal(i)
	if err != nil {
		return err
	}

	if op != discord.GatewayOpHeartbeat {
		sh.wsRatelimit.Lock()
	}

	sh.wsConnMu.RLock()
	wsConn := sh.wsConn
	sh.wsConnMu.RUnlock()

	sh.Logger.Trace().Msg("<<< " + gotils_strconv.B2S(res))

	err = wsConn.Write(ctx, websocket.MessageText, res)

	return err
}

// decodeContent converts the stored msg into the passed interface.
func (sh *Shard) decodeContent(msg discord.GatewayPayload, out interface{}) (err error) {
	err = json.Unmarshal(msg.Data, &out)

	// TODO: Remove in production.
	outD, _ := json.Marshal(out)
	if bytes.Compare(outD, msg.Data) != 0 {
		// sh.Logger.Warn().Str("In", string(msg.Data)).Str("Out", string(outD)).Msg("Varied payloads detected")
	} else {
		sh.Logger.Info().Str("type", msg.Type).Int("op", int(msg.Op)).Msg("payload pass")
	}

	return
}

// readMessage fills the shard msg buffer from a websocket message.
func (sh *Shard) readMessage() (msg discord.GatewayPayload, err error) {
	sh.channelMu.RLock()
	errorCh := sh.ErrorCh
	messageCh := sh.MessageCh
	sh.channelMu.RUnlock()

	select {
	case err = <-errorCh:
		return msg, err
	case msg = <-messageCh:
		return msg, nil
	}
}

// Close closes the shard connection.
func (sh *Shard) Close(code websocket.StatusCode) {
	sh.Logger.Info().Int("code", int(code)).Msg("Closing shard")

	sh.RoutineDeadSignal.Close("CLOSE")
	sh.RoutineDeadSignal.Revive()

	if sh.ctx != nil {
		sh.cancel()
	}

	if sh.hasWsConn() {
		if err := sh.CloseWS(code); err != nil {
			sh.Logger.Debug().Err(err).Msg("Encountered error closing websocket")
		}
	}
}

// CloseWS closes the websocket. This will always return 0 as the error is suppressed.
func (sh *Shard) CloseWS(statusCode websocket.StatusCode) (err error) {
	if sh.hasWsConn() {
		sh.Logger.Debug().Int("code", int(statusCode)).Msg("Closing websocket connection")

		sh.wsConnMu.RLock()
		wsConn := sh.wsConn
		sh.wsConnMu.RUnlock()

		err = wsConn.Close(statusCode, "")
		if err != nil && !xerrors.Is(err, context.Canceled) {
			sh.Logger.Warn().Err(err).Msg("Failed to close websocket connection")
		}

		sh.wsConnMu.Lock()
		sh.wsConn = nil
		sh.wsConnMu.Unlock()
	}

	return nil
}

// WaitForReady blocks until the shard is ready.
func (sh *Shard) WaitForReady() {
	since := time.Now().UTC()
	t := time.NewTicker(WaitForReadyTimeout)

	defer t.Stop()

	sh.RoutineDeadSignal.Started("WFR")
	defer sh.RoutineDeadSignal.Done("WFR")

	for {
		select {
		case <-sh.ready:
			return
		case <-sh.RoutineDeadSignal.Dead():
			return
		case <-t.C:
			sh.Logger.Debug().
				Dur("since", time.Now().UTC().Sub(since).Round(time.Second)).
				Msg("Still waiting for shard to be ready")
		}
	}
}

// Reconnect attempts to reconnect to the gateway.
func (sh *Shard) Reconnect(code websocket.StatusCode) error {
	wait := time.Second

	sh.Close(code)

	for {
		sh.Logger.Info().Msg("Trying to reconnect to gateway")

		err := sh.Connect()
		if err == nil {
			sh.RetriesRemaining.Store(ShardConnectRetries)
			sh.Logger.Info().Msg("Successfully reconnected to gateway")

			return nil
		}

		retries := sh.RetriesRemaining.Sub(-1)
		if retries <= 0 {
			sh.Logger.Warn().Msg("Ran out of retries whilst connecting. Attempting to reconnect client")
			sh.Close(code)

			err = sh.Connect()
			if err != nil {
				go sh.Sandwich.PublishSimpleWebhook(
					"Failed to connect to gateway",
					"`"+err.Error()+"`",
					fmt.Sprintf(
						"Manager: %s ShardGroup: %d ShardID: %d/%d",
						sh.Manager.Configuration.Identifier,
						sh.ShardGroup.ID,
						sh.ShardID,
						sh.ShardGroup.ShardCount,
					),
					EmbedColourDanger,
				)
			}

			return err
		}

		sh.Logger.Warn().Err(err).Dur("retry", wait).Msg("Failed to reconnect to gateway")
		<-time.After(wait)

		wait *= 2
		if wait > MaxReconnectWait {
			wait = MaxReconnectWait
		}
	}
}

func (sh *Shard) hasWsConn() (hasWsConn bool) {
	sh.wsConnMu.RLock()
	hasWsConn = sh.wsConn != nil
	sh.wsConnMu.RUnlock()

	return
}
