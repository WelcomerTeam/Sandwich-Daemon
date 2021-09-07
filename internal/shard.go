package internal

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/WelcomerTeam/RealRock/limiter"
	"github.com/WelcomerTeam/RealRock/snowflake"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/czlib"
	"github.com/rs/zerolog"
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
)

// Shard represents the shard object.
type Shard struct {
	ctx    context.Context
	cancel func()

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

	unavailableMu sync.RWMutex          `json:"-"`
	Unavailable   map[snowflake.ID]bool `json:"-"`

	MessageCh chan discord.GatewayPayload
	ErrorCh   chan error

	Sequence  *atomic.Int64
	SessionID *atomic.String

	wsConn *websocket.Conn

	wsRatelimit *limiter.DurationLimiter

	ready chan void
}

// NewShard creates a new shard object.
func (sg *ShardGroup) NewShard(shardID int) (sh *Shard) {
	logger := sg.Logger.With().Int("shard_id", shardID).Logger()
	sh = &Shard{
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
		Unavailable:   make(map[snowflake.ID]bool),

		Sequence:  &atomic.Int64{},
		SessionID: &atomic.String{},

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
		err := sh.Listen()
		if xerrors.Is(err, context.Canceled) {
			sh.Logger.Debug().Msg("Shard context canceled")

			return
		}

		select {
		case <-sh.ctx.Done():
			sh.Logger.Debug().Msg("Shard context done")

			return
		default:
		}
	}
}

// Connect connects to the gateway and handles identifying
func (sh *Shard) Connect() (err error) {
	sh.Logger.Debug().Msg("Connecting shard")

	select {
	case <-sh.ctx.Done():
		sh.Logger.Trace().Msg("Creating new context")

		sh.ctx, sh.cancel = context.WithCancel(sh.Manager.ctx)
	default:
		sh.Logger.Trace().Msg("No need for new context")
	}

	sh.Manager.gatewayMu.RLock()
	gatewayURL := sh.Manager.Gateway.URL
	sh.Manager.gatewayMu.RUnlock()

	defer func() {
		if err != nil && sh.wsConn != nil {
			sh.CloseWS(websocket.StatusNormalClosure)
		}
	}()

	if sh.wsConn == nil {
		var errorCh chan error

		var messageCh chan discord.GatewayPayload

		errorCh, messageCh, err = sh.FeedWebsocket(gatewayURL, nil)
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

			return
		}

		sh.ErrorCh = errorCh
		sh.MessageCh = messageCh
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

	helloResponse := discord.Hello{}

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

	if !sh.HeartbeatActive.Load() {
		go sh.Heartbeat()
	}

	sequence := sh.Sequence.Load()
	sessionID := sh.SessionID.Load()

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Int64("sequence", sequence).
		Msg("Received HELLO event")

	if sessionID == "" || sequence == 0 {
		err = sh.Identify()
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
		err = sh.Resume()
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

	// We wait until we either receive a first event, error
	// or we hit our FirstEventTimeout. We do nothing when
	// hitting the FirstEventtimeout.

	sh.Logger.Trace().Msg("Waiting for first event")

	select {
	case err = <-sh.ErrorCh:
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
	case msg = <-sh.MessageCh:
		sh.Logger.Debug().Msgf("Received first event %d %s", msg.Op, msg.Type)

		sh.MessageCh <- msg
	case <-t.C:
	}

	return err
}

// Listen to gateway and process accordingly.
func (sh *Shard) Listen() (err error) {
	wsConn := sh.wsConn

	for {
		select {
		case <-sh.ctx.Done():
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
						EmbedColourDanger,
					)

					sh.ShardGroup.Error.Store(err.Error())

					return err
				default:
					sh.Logger.Warn().Msgf("Websocket was closed with code %d", closeError.Code)
				}
			}

			if wsConn == sh.wsConn {
				// We have likely closed so we should attempt to reconnect
				sh.Logger.Warn().Msg("We have encountered an error whilst in the same connection, reconnecting...")
				err = sh.Reconnect(websocket.StatusNormalClosure)

				if err != nil {
					return err
				}

				return nil
			}

			wsConn = sh.wsConn
		}

		sh.OnEvent(msg)

		// In the event we have reconnected, the wsConn could have changed,
		// we will use the new wsConn if this is the case
		if sh.wsConn != wsConn {
			sh.Logger.Debug().Msg("New wsConn was assigned to shard")
			wsConn = sh.wsConn
		}
	}

	return err
}

// FeedWebsocket reads websocket events and feeds them through a channel.
func (sh *Shard) FeedWebsocket(u string,
	opts *websocket.DialOptions) (errorCh chan error, messageCh chan discord.GatewayPayload, err error) {
	messageCh = make(chan discord.GatewayPayload, MessageChannelBuffer)
	errorCh = make(chan error, 1)

	conn, _, err := websocket.Dial(sh.ctx, u, opts)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to dial websocket")

		return errorCh, messageCh, xerrors.Errorf("failed to connect to websocket: %w", err)
	}

	conn.SetReadLimit(WebsocketReadLimit)
	sh.wsConn = conn

	go func() {
		for {
			_, buf, err := conn.Read(sh.ctx)

			select {
			case <-sh.ctx.Done():
				return
			default:
			}

			if err != nil {
				errorCh <- err

				return
			}

			buf, err = czlib.Decompress(buf)
			if err != nil {
				errorCh <- err

				return
			}

			msg := sh.Sandwich.receivedPool.Get().(discord.GatewayPayload)

			err = json.Unmarshal(buf, &msg)
			if err != nil {
				sh.Logger.Error().Err(err).Msg("Failed to unmarshal message")

				continue
			}

			// TODO Add Analytics

			messageCh <- msg
		}
	}()

	return errorCh, messageCh, nil
}

// Identify sends the identify packet to discord.
func (sh *Shard) Identify() (err error) {
	sh.Manager.gatewayMu.Lock()
	sh.Manager.Gateway.SessionStartLimit.Remaining--
	sh.Manager.gatewayMu.Unlock()

	sh.Manager.WaitForIdentify(sh.ShardID, sh.ShardGroup.ShardCount)
	sh.Logger.Debug().Msg("Wait for identify completed")

	sh.Logger.Debug().Msg("Sending identify")

	err = sh.SendEvent(discord.GatewayOpIdentify, discord.Identify{
		Token: sh.Manager.Configuration.Token,
		Properties: &discord.IdentifyProperties{
			OS:      runtime.GOOS,
			Browser: "Sandwich " + VERSION,
			Device:  "Sandwich " + VERSION,
		},
		Compress:       true,
		LargeThreshold: GatewayLargeThreshold,
		Shard:          [2]int{sh.ShardID, sh.ShardGroup.ShardCount},
		Presence:       sh.Manager.Configuration.Bot.DefaultPresence,
		Intents:        sh.Manager.Configuration.Bot.Intents,
	})

	return err
}

// SendEvent sends an event to discord.
func (sh *Shard) SendEvent(op discord.GatewayOp, data interface{}) (err error) {
	packet := sh.Sandwich.sentPool.Get().(*discord.SentPayload)
	defer sh.Sandwich.sentPool.Put(packet)

	packet.Op = op
	packet.Data = data

	err = sh.WriteJSON(op, packet)
	if err != nil {
		return xerrors.Errorf("sendEvent writeJson: %w", err)
	}

	return
}

// WriteJSON writes json data to the websocket.
func (sh *Shard) WriteJSON(op discord.GatewayOp, i interface{}) (err error) {
	res, err := json.Marshal(i)
	if err != nil {
		return err
	}

	if op != discord.GatewayOpHeartbeat {
		sh.wsRatelimit.Lock()
	}

	err = sh.wsConn.Write(sh.ctx, websocket.MessageText, res)

	return err
}

// decodeContent converts the stored msg into the passed interface.
func (sh *Shard) decodeContent(msg discord.GatewayPayload, out interface{}) (err error) {
	err = json.Unmarshal(msg.Data, &out)

	return
}

// readMessage fills the shard msg buffer from a websocket message.
func (sh *Shard) readMessage() (msg discord.GatewayPayload, err error) {
	// Prioritize errors
	select {
	case err = <-sh.ErrorCh:
		return msg, err
	default:
	}

	select {
	case err = <-sh.ErrorCh:
		return msg, err
	case msg = <-sh.MessageCh:
		msg.AddTrace("read", time.Now().UTC())

		return msg, nil
	}
}

// CloseWS closes the websocket. This will always return 0 as the error is suppressed.
func (sh *Shard) CloseWS(statusCode websocket.StatusCode) (err error) {
	if sh.wsConn != nil {
		sh.Logger.Debug().Int("code", int(statusCode)).Msg("Closing websocket connection")

		err = sh.wsConn.Close(statusCode, "")
		if err != nil && !xerrors.Is(err, context.Canceled) {
			sh.Logger.Warn().Err(err).Msg("Failed to close websocket connection")
		}

		sh.wsConn = nil
	}

	return nil
}

// Connect to gateway and setup message channels
// Listen handles reading from websocket, errors and basic reconnection
// Feed reads from the gateway and decompresses messages and push to message channel
// OnEvent handles gateway ops and dispatch
// OnDispatch handles cheking blacklists, handling dispatch and publishing
// Heartbeat maintains Heartbeat
// Reconnect reconnects to gateway
// Close sends a close code

// Resume
// Identify
// Reconnect

// SendEvent sends a sentpayload packet
// WriteJSON sends a message to discord respecting ratelimits

// WaitForReady returns when shard is ready

// SetStatus

// ChunkGuild chunks a guild

// readMessage returns a message or error from channels
// decodeContent unmarshals received payload
