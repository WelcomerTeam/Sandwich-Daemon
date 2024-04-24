package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WelcomerTeam/RealRock/deadlock"
	"github.com/WelcomerTeam/RealRock/limiter"
	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/WelcomerTeam/czlib"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/rs/zerolog"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	"go.uber.org/atomic"
	"nhooyr.io/websocket"
)

const (
	WebsocketReadLimit          = 512 << 20
	WebsocketReconnectCloseCode = 4000

	// Time required before warning about an event taking too long.
	DispatchWarningTimeout = 30 * time.Second

	MessageChannelBuffer = 64

	// Time necessary to mark chunking as completed when no more events are received in this time frame.
	MemberChunkTimeout = 2 * time.Second

	// Number of retries attempted before considering a shard not working.
	ShardConnectRetries = 10

	ShardWSRateLimit      = 110
	GatewayLargeThreshold = 250

	FirstEventTimeout = 5 * time.Second

	WaitForReadyTimeout = 15 * time.Second
	ReadyTimeout        = 5 * time.Second

	MaxReconnectWait = 60 * time.Second

	WriteJSONRetry = 1 * time.Second
)

// Shard represents the shard object.
type Shard struct {
	ctx    context.Context
	cancel func()

	RoutineDeadSignal   deadlock.DeadSignal `json:"-"`
	HeartbeatDeadSignal deadlock.DeadSignal `json:"-"`

	Start            *atomic.Time  `json:"start"`
	Init             *atomic.Time  `json:"init"`
	RetriesRemaining *atomic.Int32 `json:"-"`

	Logger zerolog.Logger `json:"-"`

	ShardID          int32          `json:"shard_id"`
	ResumeGatewayURL *atomic.String `json:"resume_gateway_url"`
	ConnectionURL    *atomic.String `json:"connection_url"`

	Sandwich   *Sandwich   `json:"-"`
	Manager    *Manager    `json:"-"`
	ShardGroup *ShardGroup `json:"-"`

	HeartbeatActive   *atomic.Bool `json:"-"`
	LastHeartbeatAck  *atomic.Time `json:"-"`
	LastHeartbeatSent *atomic.Time `json:"-"`

	Heartbeater       *time.Ticker  `json:"-"`
	HeartbeatInterval time.Duration `json:"-"`

	// Duration since last heartbeat Ack before reconnecting.
	HeartbeatFailureInterval time.Duration `json:"-"`

	// Map of guilds that are currently unavailable.
	Unavailable *csmap.CsMap[discord.Snowflake, bool] `json:"unavailable"`

	// Map of guilds that have were present in ready and not received yet.
	Lazy *csmap.CsMap[discord.Snowflake, bool] `json:"lazy"`

	// Stores a local list of all guilds in the shard.
	Guilds *csmap.CsMap[discord.Snowflake, bool] `json:"guilds"`

	statusMu sync.RWMutex
	Status   sandwich_structs.ShardStatus `json:"status"`

	channelMu sync.RWMutex
	MessageCh chan discord.GatewayPayload `json:"-"`
	ErrorCh   chan error                  `json:"-"`

	Sequence  *atomic.Int32  `json:"-"`
	SessionID *atomic.String `json:"-"`

	wsConnMu sync.RWMutex
	wsConn   *websocket.Conn

	wsRatelimit *limiter.DurationLimiter

	ready chan void

	IsReady bool
}

// NewShard creates a new shard object.
func (sg *ShardGroup) NewShard(shardID int32) (sh *Shard) {
	logger := sg.Logger.With().Int32("shardId", shardID).Logger()
	sh = &Shard{
		RoutineDeadSignal:   deadlock.DeadSignal{},
		HeartbeatDeadSignal: deadlock.DeadSignal{},

		RetriesRemaining: atomic.NewInt32(ShardConnectRetries),

		Logger: logger,

		ShardID: shardID,

		Sandwich:   sg.Manager.Sandwich,
		Manager:    sg.Manager,
		ShardGroup: sg,

		Start: &atomic.Time{},
		Init:  atomic.NewTime(time.Now().UTC()),

		HeartbeatActive:   atomic.NewBool(false),
		LastHeartbeatAck:  &atomic.Time{},
		LastHeartbeatSent: &atomic.Time{},

		Unavailable: csmap.Create(
			csmap.WithSize[discord.Snowflake, bool](1000),
		),

		Lazy: csmap.Create(
			csmap.WithSize[discord.Snowflake, bool](1000),
		),

		Guilds: csmap.Create(
			csmap.WithSize[discord.Snowflake, bool](1000),
		),

		statusMu: sync.RWMutex{},
		Status:   sandwich_structs.ShardStatusIdle,

		channelMu: sync.RWMutex{},

		Sequence:         &atomic.Int32{},
		SessionID:        &atomic.String{},
		ResumeGatewayURL: &atomic.String{},
		ConnectionURL:    &atomic.String{},

		wsConnMu: sync.RWMutex{},

		// We use 110 both to allow heartbeating to not be limited and to allow
		// bursts of 10 messages can be sent.
		wsRatelimit: limiter.NewDurationLimiter(ShardWSRateLimit, 2*time.Minute),

		ready: make(chan void, 1),
	}

	sh.ctx, sh.cancel = context.WithCancel(sg.Manager.ctx)

	return sh
}

// Open starts listening to the gateway.
func (sh *Shard) Open() {
	sh.Logger.Debug().Msg("Started listening to shard")

	// We put the connecting here instead of in the Connect() as to allow
	// the Reconnecting status to not be overidden.

	for {
		err := sh.Listen(sh.ctx)
		if errors.Is(err, context.Canceled) {
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
func (sh *Shard) Connect() error {
	sh.Logger.Debug().Msg("Connecting shard")

	// Do not override status if it is currently Reconnecting.
	if sh.GetStatus() != sandwich_structs.ShardStatusReconnecting {
		sh.SetStatus(sandwich_structs.ShardStatusConnecting)
	}

	var err error

	defer func() {
		if err != nil {
			sh.SetStatus(sandwich_structs.ShardStatusErroring)
		}
	}()

	// Empty ready channel.
readyConsumer:
	for {
		select {
		case <-sh.ready:
			sh.IsReady = true
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

	defer func() {
		if err != nil && sh.hasWsConn() {
			sh.CloseWS(websocket.StatusNormalClosure)
		}
	}()

	var origGwUrl = gatewayURL.String()
	var resumeGwUrl = sh.ResumeGatewayURL.Load()
	var gwUrl string

	if resumeGwUrl != "" {
		gwUrl = resumeGwUrl

		sh.Logger.Debug().Str("url", gwUrl).Msg("Resuming shard")
		sh.ResumeGatewayURL.Store("") // Reset the resume url
	} else {
		sh.SessionID.Store("")
		gwUrl = origGwUrl
	}

	if !sh.hasWsConn() || sh.ConnectionURL.Load() != gwUrl {
		if sh.hasWsConn() {
			sh.Logger.Debug().Msg("Closing existing websocket connection")
			err = sh.CloseWS(websocket.StatusInternalError)

			if err != nil {
				sh.Logger.Error().Err(err).Msg("Failed to close existing websocket connection")
			}

			sh.ConnectionURL.Store("")
		}

		sh.ConnectionURL.Store(gwUrl)

		errorCh, messageCh, err := sh.FeedWebsocket(sh.ctx, gwUrl, nil)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to dial gateway")

			go sh.Sandwich.PublishSimpleWebhook(
				fmt.Sprintf("Failed to dial `%s`", gwUrl),
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

		return err
	}

	var helloResponse discord.Hello

	err = sh.decodeContent(msg, &helloResponse)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	sh.Start.Store(now)
	sh.LastHeartbeatAck.Store(now)
	sh.LastHeartbeatSent.Store(now)

	var hbIntervalWithJitter = int32(float32(helloResponse.HeartbeatInterval) * 0.8)

	sh.HeartbeatInterval = time.Duration(hbIntervalWithJitter) * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures

	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

	go sh.Heartbeat(sh.ctx)

	sequence := sh.Sequence.Load()
	sessionID := sh.SessionID.Load()

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Int32("sequence", sequence).
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

			return err
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

			return err
		}

		// We can assume the bot is now connected to discord.
	}

	t := time.NewTicker(FirstEventTimeout)
	defer t.Stop()

	// We wait until we either receive a first event, error
	// or we hit our FirstEventTimeout. We do nothing when
	// hitting the FirstEventtimeout.

	sh.SetStatus(sandwich_structs.ShardStatusConnected)

	sh.Logger.Trace().Msg("Waiting for first event")

	sh.channelMu.RLock()
	defer sh.channelMu.RUnlock()

	sh.channelMu.RLock()
	errorCh := sh.ErrorCh
	messageCh := sh.MessageCh
	sh.channelMu.RUnlock()

	select {
	case err = <-errorCh:
		if err == nil {
			err = fmt.Errorf("error channel closed")
		}

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
	sh.HeartbeatDeadSignal.Started()

	defer func() {
		sh.HeartbeatActive.Store(false)
		sh.HeartbeatDeadSignal.Done()
	}()

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

			if err != nil || (now.Sub(sh.LastHeartbeatAck.Load()) > sh.HeartbeatFailureInterval) {
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
					err = fmt.Errorf("failed to ack and passed heartbeat failure interval")

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

				sh.ErrorCh <- err

				return
			}
		}
	}
}

// Listen to gateway and process accordingly.
func (sh *Shard) Listen(ctx context.Context) error {
	sh.wsConnMu.RLock()
	wsConn := sh.wsConn
	sh.wsConnMu.RUnlock()

	for {
		select {
		case <-sh.RoutineDeadSignal.Dead():
			return nil
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := sh.readMessage()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
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

		trace := csmap.Create(
			csmap.WithSize[string, discord.Int64](uint64(1)),
		)

		trace.Store("receive", discord.Int64(time.Now().Unix()))

		sh.OnEvent(ctx, msg, trace)

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

	return nil
}

// FeedWebsocket reads websocket events and feeds them through a channel.
func (sh *Shard) FeedWebsocket(ctx context.Context, u string,
	opts *websocket.DialOptions,
) (errorCh chan error, messageCh chan discord.GatewayPayload, err error) {
	messageCh = make(chan discord.GatewayPayload, MessageChannelBuffer)
	errorCh = make(chan error, 1)

	conn, _, err := websocket.Dial(ctx, u, opts)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to dial websocket")

		return errorCh, messageCh, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	conn.SetReadLimit(WebsocketReadLimit)

	sh.wsConnMu.Lock()
	sh.wsConn = conn
	sh.wsConnMu.Unlock()

	go func() {
		sh.RoutineDeadSignal.Started()
		defer sh.RoutineDeadSignal.Done()

		for {
			messageType, data, connectionErr := conn.Read(ctx)

			select {
			case <-sh.RoutineDeadSignal.Dead():
				return
			case <-ctx.Done():
				return
			default:
			}

			sandwichEventCount.WithLabelValues(sh.Manager.Identifier.Load()).Add(1)

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

			msg, _ := sh.Sandwich.receivedPool.Get().(*discord.GatewayPayload)

			connectionErr = sandwichjson.Unmarshal(data, &msg)
			if connectionErr != nil {
				sh.Logger.Error().Err(connectionErr).Msg("Failed to unmarshal message")

				continue
			}

			select {
			case messageCh <- *msg:
				continue
			case <-sh.RoutineDeadSignal.Dead():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return errorCh, messageCh, nil
}

// Identify sends the identify packet to discord.
func (sh *Shard) Identify(ctx context.Context) error {
	sh.Manager.gatewayMu.Lock()
	sh.Manager.Gateway.SessionStartLimit.Remaining--
	sh.Manager.gatewayMu.Unlock()

	err := sh.Manager.WaitForIdentify(sh.ShardID, sh.ShardGroup.ShardCount)
	if err != nil {
		return fmt.Errorf("failed to wait for identify: %w", err)
	}

	sh.Logger.Debug().Msg("Wait for identify completed")

	sh.Manager.configurationMu.RLock()
	token := sh.Manager.Configuration.Token
	presence := sh.fillInUpdateStatus(&sh.Manager.Configuration.Bot.DefaultPresence)
	intents := sh.Manager.Configuration.Bot.Intents
	sh.Manager.configurationMu.RUnlock()

	sh.Logger.Debug().Msg("Sending identify")

	return sh.SendEvent(ctx, discord.GatewayOpIdentify, discord.Identify{
		Token: token,
		Properties: &discord.IdentifyProperties{
			OS:      runtime.GOOS,
			Browser: "Sandwich " + VERSION,
			Device:  "Sandwich " + VERSION,
		},
		Compress:       true,
		LargeThreshold: GatewayLargeThreshold,
		Shard:          [2]int32{sh.ShardID, sh.ShardGroup.ShardCount},
		Presence:       presence,
		Intents:        intents,
	})
}

// Resume sends the resume packet to discord.
func (sh *Shard) Resume(ctx context.Context) error {
	sh.Manager.configurationMu.RLock()
	token := sh.Manager.Configuration.Token
	sh.Manager.configurationMu.RUnlock()

	sh.Logger.Debug().Msg("Sending resume")

	return sh.SendEvent(ctx, discord.GatewayOpResume, discord.Resume{
		Token:     token,
		SessionID: sh.SessionID.Load(),
		Sequence:  sh.Sequence.Load(),
	})
}

func (sh *Shard) fillInUpdateStatus(us *discord.UpdateStatus) *discord.UpdateStatus {
	for _, activity := range us.Activities {
		activity.Name = strings.ReplaceAll(activity.Name, "{{shard_id}}", strconv.Itoa(int(sh.ShardID)))
		activity.Name = strings.ReplaceAll(activity.Name, "{{shardgroup_id}}", strconv.Itoa(int(sh.ShardGroup.ID)))
		activity.State = strings.ReplaceAll(activity.State, "{{shard_id}}", strconv.Itoa(int(sh.ShardID)))
		activity.State = strings.ReplaceAll(activity.State, "{{shardgroup_id}}", strconv.Itoa(int(sh.ShardGroup.ID)))
	}
	return us
}

func (sh *Shard) UpdatePresence(ctx context.Context, us *discord.UpdateStatus) error {
	filledInStatus := sh.fillInUpdateStatus(us)
	jsonStatusBytes, err := sandwichjson.Marshal(filledInStatus)

	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	sh.Logger.Debug().Str("status", string(jsonStatusBytes)).Msg("Sending update status")

	return sh.SendEvent(ctx, discord.GatewayOpStatusUpdate, json.RawMessage(jsonStatusBytes))
}

// SendEvent sends an event to discord.
func (sh *Shard) SendEvent(ctx context.Context, op discord.GatewayOp, data interface{}) error {
	packet, _ := sh.Sandwich.sentPool.Get().(*discord.SentPayload)
	defer sh.Sandwich.sentPool.Put(packet)

	packet.Op = op
	packet.Data = data

	err := sh.WriteJSON(ctx, op, packet)
	if err != nil {
		return fmt.Errorf("sendEvent writeJson: %w", err)
	}

	return nil
}

// WriteJSON writes json data to the websocket.
func (sh *Shard) WriteJSON(ctx context.Context, op discord.GatewayOp, i interface{}) error {
	// In very rare circumstances, we can be writing to the websocket whilst
	// context is being remade. We will recover and dismiss any SIGSEGVs that
	// are raised.
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if ok {
				sh.Logger.Warn().Err(err).Bool("hasWsConn", sh.wsConn != nil).Msg("Recovered panic in WriteJSON")
			} else {
				sh.Logger.Warn().Interface("recovered", r).Bool("hasWsConn", sh.wsConn != nil).Msg("Recovered panic in WriteJSON")
			}
		}
	}()

	if !sh.hasWsConn() {
		// Try to reconnect
		err := sh.Reconnect(websocket.StatusNormalClosure)
		return fmt.Errorf("no websocket connection: %w", err)
	}

	res, err := sandwichjson.Marshal(i)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if op != discord.GatewayOpHeartbeat {
		sh.wsRatelimit.Lock()
	}

	sh.wsConnMu.RLock()
	wsConn := sh.wsConn
	sh.wsConnMu.RUnlock()

	sh.Logger.Trace().Msg("<<< " + gotils_strconv.B2S(res))

	err = wsConn.Write(ctx, websocket.MessageText, res)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// decodeContent converts the stored msg into the passed interface.
func (sh *Shard) decodeContent(msg discord.GatewayPayload, out interface{}) error {
	err := sandwichjson.Unmarshal(msg.Data, &out)
	if err != nil {
		sh.Logger.Error().Err(err).Str("type", msg.Type).Msg("Failed to decode event")

		return err
	}

	return nil
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

	sh.IsReady = false

	sh.SetStatus(sandwich_structs.ShardStatusClosing)

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

	// Try killing the producer's shard as well
	pc := sh.Manager.ProducerClient
	if pc != nil {
		pc.CloseShard(sh.ShardID)
	}

	sh.SetStatus(sandwich_structs.ShardStatusClosed)
}

// CloseWS closes the websocket. This will always return 0 as the error is suppressed.
func (sh *Shard) CloseWS(statusCode websocket.StatusCode) error {
	if sh.hasWsConn() {
		sh.Logger.Debug().Int("code", int(statusCode)).Msg("Closing websocket connection")

		sh.wsConnMu.Lock()
		wsConn := sh.wsConn

		if wsConn != nil {
			err := wsConn.Close(statusCode, "")
			if err != nil && !errors.Is(err, context.Canceled) {
				sh.Logger.Warn().Err(err).Msg("Failed to close websocket connection")
			}
		}

		sh.wsConn = nil
		sh.wsConnMu.Unlock()
	}

	return nil
}

// WaitForReady blocks until the shard is ready.
func (sh *Shard) WaitForReady() {
	// We are already ready
	if sh.IsReady {
		return
	}

	since := time.Now().UTC()
	t := time.NewTicker(WaitForReadyTimeout)

	defer t.Stop()

	sh.RoutineDeadSignal.Started()
	defer sh.RoutineDeadSignal.Done()

	for {
		if sh.IsReady {
			return
		}

		select {
		case <-sh.ready:
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

	sh.SetStatus(sandwich_structs.ShardStatusReconnecting)

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

func (sh *Shard) ChunkAllGuilds() {
	guilds := make([]discord.Snowflake, sh.Guilds.Count())
	i := 0

	sh.Guilds.Range(func(guildID discord.Snowflake, _ bool) bool {
		guilds[i] = guildID
		i++
		return false
	})

	sh.Logger.Info().Int("guilds", len(guilds)).Msg("Started chunking all guilds")

	for _, guildID := range guilds {
		sh.ChunkGuild(guildID, false)
	}

	sh.Logger.Info().Int("guilds", len(guilds)).Msg("Finished chunking all guilds")
}

// ChunkGuilds chunks guilds to discord. It will wait for the operation to complete, or timeout.
func (sh *Shard) ChunkGuild(guildID discord.Snowflake, alwaysChunk bool) error {
	guildChunk, ok := sh.Sandwich.guildChunks.Load(guildID)

	if !ok {
		guildChunk = &GuildChunks{
			Complete:        *atomic.NewBool(false),
			ChunkingChannel: make(chan GuildChunkPartial),
			StartedAt:       *atomic.NewTime(time.Time{}),
			CompletedAt:     *atomic.NewTime(time.Time{}),
		}

		sh.Sandwich.guildChunks.Store(guildID, guildChunk)
	}

	guildChunk.Complete.Store(false)
	guildChunk.StartedAt.Store(time.Now())

	var memberCount int

	gm, ok := sh.Sandwich.State.GuildMembers.Load(guildID)

	if ok {
		memberCount = gm.Members.Count()
	}

	guild, ok := sh.Sandwich.State.Guilds.Load(guildID)

	if !ok {
		return fmt.Errorf("guild not found in state, refusing to chunk")
	}

	needsChunking := guild.MemberCount > int32(memberCount)

	if needsChunking || alwaysChunk {
		nonce := randomHex(16)

		err := sh.SendEvent(sh.ctx, discord.GatewayOpRequestGuildMembers, discord.RequestGuildMembers{
			GuildID: guildID,
			Nonce:   nonce,
		})
		if err != nil {
			return fmt.Errorf("failed to send request guild members event: %w", err)
		}

		chunksReceived := int32(0)
		totalChunks := int32(0)

		timeout := time.NewTimer(MemberChunkTimeout)

	guildChunkLoop:
		for {
			select {
			case guildChunkPartial := <-guildChunk.ChunkingChannel:
				if guildChunkPartial.Nonce != nonce {
					continue
				}

				chunksReceived++
				totalChunks = guildChunkPartial.ChunkCount

				// When receiving a chunk, reset the timeout.
				timeout.Reset(MemberChunkTimeout)

				sh.Logger.Debug().
					Int64("guild_id", int64(guildID)).
					Int32("chunk_index", guildChunkPartial.ChunkIndex).
					Int32("chunk_count", guildChunkPartial.ChunkCount).
					Msg("Received guild member chunk")

				if chunksReceived >= totalChunks {
					sh.Logger.Debug().
						Int64("guild_id", int64(guildID)).
						Int32("total_chunks", totalChunks).
						Msg("Received all guild member chunks")

					break guildChunkLoop
				}
			case <-timeout.C:
				// We have timed out. We will mark the chunking as complete.

				sh.Logger.Warn().
					Int64("guild_id", int64(guildID)).
					Int32("chunks_received", chunksReceived).
					Int32("total_chunks", totalChunks).
					Msg("Timed out receiving guild member chunks")

				break guildChunkLoop
			}
		}

		timeout.Stop()
	}

	guildChunk.Complete.Store(true)
	guildChunk.CompletedAt.Store(time.Now())

	return nil
}

// OnDispatchEvent is called during the dispatch event to call analytics.
func (sh *Shard) OnDispatchEvent(eventType string) {
	sh.OnGuildDispatchEvent(eventType, discord.Snowflake(0))
}

// OnGuildDispatchEvent is called during the dispatch event to call analytics with a guild Id.
func (sh *Shard) OnGuildDispatchEvent(eventType string, guildID discord.Snowflake) {
	sandwichDispatchEventCount.WithLabelValues(sh.Manager.Identifier.Load(), eventType).Inc()
}

// SafeOnGuildDispatchEvent takes a guildID pointer and does handle guild event count if nil.
func (sh *Shard) SafeOnGuildDispatchEvent(eventType string, guildIDPtr *discord.Snowflake) {
	if guildIDPtr != nil {
		sh.OnGuildDispatchEvent(eventType, *guildIDPtr)
	} else {
		sh.OnDispatchEvent(eventType)
	}
}

// SetStatus sets the status of the ShardGroup.
func (sh *Shard) SetStatus(status sandwich_structs.ShardStatus) {
	sh.statusMu.Lock()
	defer sh.statusMu.Unlock()

	sh.Logger.Debug().Int("status", int(status)).Msg("Shard status changed")

	sh.Status = status

	payload, _ := sandwichjson.Marshal(sandwich_structs.ShardStatusUpdate{
		Manager:    sh.Manager.Identifier.Load(),
		ShardGroup: sh.ShardGroup.ID,
		Shard:      sh.ShardID,
		Status:     sh.Status,
	})

	_ = sh.Manager.Sandwich.PublishGlobalEvent(sandwich_structs.SandwichEventShardStatusUpdate, json.RawMessage(payload))
}

// GetStatus returns the status of a ShardGroup.
func (sh *Shard) GetStatus() (status sandwich_structs.ShardStatus) {
	sh.statusMu.RLock()
	defer sh.statusMu.RUnlock()

	return sh.Status
}

func (sh *Shard) hasWsConn() (hasWsConn bool) {
	sh.wsConnMu.RLock()
	hasWsConn = sh.wsConn != nil
	sh.wsConnMu.RUnlock()

	return
}
