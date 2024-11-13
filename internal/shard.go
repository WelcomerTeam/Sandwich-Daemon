package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WelcomerTeam/RealRock/deadlock"
	"github.com/WelcomerTeam/RealRock/limiter"
	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	"github.com/WelcomerTeam/czlib"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/rs/zerolog"
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
	Logger zerolog.Logger `json:"-"`

	HeartbeatDeadSignal deadlock.DeadSignal `json:"-"`

	ctx    context.Context
	cancel func()

	Start            *atomic.Time  `json:"start"`
	Init             *atomic.Time  `json:"init"`
	RetriesRemaining *atomic.Int32 `json:"-"`

	ResumeGatewayURL *atomic.String `json:"resume_gateway_url"`
	ConnectionURL    *atomic.String `json:"connection_url"`

	Sandwich   *Sandwich   `json:"-"`
	Manager    *Manager    `json:"-"`
	ShardGroup *ShardGroup `json:"-"`

	HeartbeatActive   *atomic.Bool `json:"-"`
	LastHeartbeatAck  *atomic.Time `json:"-"`
	LastHeartbeatSent *atomic.Time `json:"-"`

	Heartbeater *time.Ticker `json:"-"`

	// Map of guilds that are currently unavailable.
	Unavailable *csmap.CsMap[discord.Snowflake, struct{}] `json:"unavailable"`

	// Map of guilds that have were present in ready and not received yet.
	Lazy *csmap.CsMap[discord.Snowflake, struct{}] `json:"lazy"`

	// Stores a local list of all guilds in the shard.
	Guilds *csmap.CsMap[discord.Snowflake, struct{}] `json:"guilds"`

	Sequence  *atomic.Int32  `json:"-"`
	SessionID *atomic.String `json:"-"`

	wsConn *websocket.Conn

	wsRatelimit *limiter.DurationLimiter

	ready chan void

	metadata          *sandwich_structs.SandwichMetadata `json:"-"`
	HeartbeatInterval time.Duration                      `json:"-"`

	// Duration since last heartbeat Ack before reconnecting.
	HeartbeatFailureInterval time.Duration `json:"-"`

	statusMu sync.RWMutex

	wsConnMu sync.RWMutex

	metadataMu sync.RWMutex

	ShardID int32                        `json:"shard_id"`
	Status  sandwich_structs.ShardStatus `json:"status"`

	IsReady bool
}

// NewShard creates a new shard object.
func (sg *ShardGroup) NewShard(shardID int32) (sh *Shard) {
	logger := sg.Logger.With().Int32("shardId", shardID).Logger()
	sh = &Shard{
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
			csmap.WithSize[discord.Snowflake, struct{}](0),
		),

		Lazy: csmap.Create(
			csmap.WithSize[discord.Snowflake, struct{}](0),
		),

		Guilds: csmap.Create(
			csmap.WithSize[discord.Snowflake, struct{}](0),
		),

		statusMu: sync.RWMutex{},
		Status:   sandwich_structs.ShardStatusIdle,

		Sequence:         &atomic.Int32{},
		SessionID:        &atomic.String{},
		ResumeGatewayURL: &atomic.String{},
		ConnectionURL:    &atomic.String{},

		wsConnMu: sync.RWMutex{},

		// We use 110 both to allow heartbeating to not be limited and to allow
		// bursts of 10 messages can be sent.
		wsRatelimit: limiter.NewDurationLimiter(ShardWSRateLimit, 2*time.Minute),

		ready: make(chan void, 1),

		metadataMu: sync.RWMutex{},
		metadata: &sandwich_structs.SandwichMetadata{
			Version:       VERSION,
			Identifier:    sg.Manager.Configuration.ProducerIdentifier,
			Application:   sg.Manager.Identifier.Load(),
			ApplicationID: discord.Snowflake(sg.Manager.UserID.Load()),
			Shard: [3]int32{
				sg.ID,
				shardID,
				sg.ShardCount,
			},
		},
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
		case <-sh.ctx.Done():
			return
		default:
		}
	}
}

func (sh *Shard) readMessage() (payload discord.GatewayPayload, err error) {
	messageType, data, connectionErr := sh.wsConn.Read(sh.ctx)
	if connectionErr != nil {
		select {
		case <-sh.ctx.Done():
			return payload, connectionErr
		default:
		}

		sh.Logger.Error().Err(connectionErr).Msg("Failed to read from gateway")

		return payload, connectionErr
	}

	sandwichEventCount.WithLabelValues(sh.Manager.Identifier.Load()).Add(1)

	if messageType == websocket.MessageBinary {
		data, connectionErr = czlib.Decompress(data)
		if connectionErr != nil {
			sh.Logger.Error().Err(connectionErr).Msg("Failed to decompress data")

			return payload, connectionErr
		}
	}

	msg, _ := sh.Sandwich.receivedPool.Get().(*discord.GatewayPayload)

	connectionErr = json.Unmarshal(data, &msg)
	if connectionErr != nil {
		sh.Logger.Error().Err(connectionErr).Msg("Failed to unmarshal message")

		return payload, connectionErr
	}

	return *msg, nil
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

	sh.HeartbeatDeadSignal.Close("HB")
	sh.HeartbeatDeadSignal.Revive()

	sh.ctx, sh.cancel = context.WithCancel(sh.Manager.ctx)

	defer func() {
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to connect. Closing shard")
			if sh.hasWsConn() {
				err = sh.CloseWS(websocket.StatusNormalClosure)
				if err != nil {
					sh.Logger.Debug().Err(err).Msg("Failed to close websocket")
				}
			}
		}
	}()

	wsURL := sh.ResumeGatewayURL.Load()
	if wsURL == "" {
		wsURL = gatewayURL.String()
	}

	if !sh.hasWsConn() {
		err := sh.dial(sh.ctx, wsURL, nil)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to dial gateway")

			go sh.Sandwich.PublishSimpleWebhook(
				fmt.Sprintf("Failed to dial `%s`", gatewayURL.String()),
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
		sh.Logger.Info().Msg("Reusing websocket connection")
	}

	// Read a message from Gateway, this should be Hello
	msg, err := sh.readMessage()
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to read message")

		return err
	}

	var hello discord.Hello

	err = sh.decodeContent(msg, &hello)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	sh.Start.Store(now)
	sh.LastHeartbeatAck.Store(now)
	sh.LastHeartbeatSent.Store(now)

	if hello.HeartbeatInterval <= 0 {
		sh.Logger.Error().
			Int32("interval", hello.HeartbeatInterval).
			Str("event_type", msg.Type).
			Str("event_data", string(msg.Data)).
			Msg("Invalid heartbeat interval")

		return ErrInvalidHeartbeatInterval
	}

	sh.HeartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures

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

	return err
}

// Heartbeat maintains a heartbeat with discord.
func (sh *Shard) Heartbeat(ctx context.Context) {
	sh.HeartbeatActive.Store(true)
	sh.HeartbeatDeadSignal.Started()

	// We will add jitter to the heartbeat to prevent all shards from sending at the same time.

	hasJitter := true
	heartbeatJitter := time.Duration(rand.Int64N(sh.HeartbeatInterval.Milliseconds())+1) * time.Millisecond
	sh.Heartbeater = time.NewTicker(heartbeatJitter)

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
			if hasJitter {
				sh.Heartbeater.Reset(sh.HeartbeatInterval)
				hasJitter = false
			}

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

				if err != nil {
					sh.Logger.Error().Err(err).Msg("Failed to send heartbeat")
				}

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

	sh.Manager.configurationMu.RLock()
	disableTrace := sh.Manager.Configuration.DisableTrace
	sh.Manager.configurationMu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := sh.readMessage()

		var trace *csmap.CsMap[string, discord.Int64]
		if !disableTrace {
			trace = csmap.Create(
				csmap.WithSize[string, discord.Int64](uint64(1)),
			)

			trace.Store("receive", discord.Int64(time.Now().Unix()))
		}

		if err == nil {
			sh.OnEvent(ctx, msg, trace)
		} else {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				sh.Logger.Error().Err(err).Msg("Context is done. Stopping feed")

				break
			}

			var closeError websocket.CloseError

			sh.Logger.Error().Err(err).Bool("is_close", errors.As(err, &closeError)).Msg("Error reading from gateway")

			if ok := errors.As(err, &closeError); ok {
				sh.Logger.Error().Int("code", int(closeError.Code)).Msg("Shard received closure code")
				// If possible, we will check the close error to determine if we can continue
				switch closeError.Code {
				case discord.CloseAuthenticationFailed, // Authentication failed
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
			} else {
				mj, _ := json.Marshal(msg)
				sh.Logger.Error().Err(err).Str("msg", string(mj)).Msg("Failed with unhandled error")
			}

			sh.wsConnMu.RLock()
			connEqual := wsConn == sh.wsConn
			sh.wsConnMu.RUnlock()

			if connEqual {
				// We have likely closed so we should attempt to reconnect
				sh.Logger.Warn().Err(err).Msg("We have encountered an error whilst in the same connection. Reconnecting")

				err = sh.Reconnect(websocket.StatusNormalClosure)
				if err != nil {
					sh.Logger.Error().Err(err).Msg("Failed to reconnect")

					return err
				}

				return nil
			}
		}
	}

	return nil
}

// dial reads connects to the discord gateway.
func (sh *Shard) dial(ctx context.Context, u string, opts *websocket.DialOptions) (err error) {
	urlp, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("failed to parse url: %w", err)
	}

	// Add version and encoding
	urlp.RawQuery = rawQuery
	u = urlp.String()

	conn, _, err := websocket.Dial(ctx, u, opts)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to dial websocket")
		sh.ResumeGatewayURL.Store("")

		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	sh.Logger.Info().Str("url", u).Msg("Connecting to websocket")

	conn.SetReadLimit(-1)

	sh.wsConnMu.Lock()
	sh.wsConn = conn
	sh.wsConnMu.Unlock()

	return nil
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
		Properties: discord.IdentifyProperties{
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

	sh.Logger.Debug().
		Str("token", token).
		Str("session_id", sh.SessionID.Load()).
		Int32("sequence", sh.Sequence.Load()).
		Msg("Sending resume")

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
		err := sh.Reconnect(WebsocketReconnectCloseCode)
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

	if op != discord.GatewayOpHeartbeat {
		sh.Logger.Debug().Int("op", int(op)).Str("data", string(res)).Msg("Sending gateway event")
	}

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

// Close closes the shard connection.
func (sh *Shard) Close(code websocket.StatusCode, intermittentGwIssue bool) {
	sh.Logger.Info().Int("code", int(code)).Msg("Closing shard")

	sh.IsReady = false

	sh.SetStatus(sandwich_structs.ShardStatusClosing)

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
		if intermittentGwIssue {
			pc.CloseShard(sh.ShardID, MQCloseShardReasonGateway)
		} else {
			pc.CloseShard(sh.ShardID, MQCloseShardReasonOther)

		}
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

	for {
		if sh.IsReady {
			return
		}

		select {
		case <-sh.ready:
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

	sh.Close(code, true)

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
			sh.Close(code, false)

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

	sh.Guilds.Range(func(guildID discord.Snowflake, _ struct{}) bool {
		guilds[i] = guildID
		i++
		return false
	})

	sh.Logger.Info().Int("guilds", len(guilds)).Msg("Started chunking all guilds")

	for _, guildID := range guilds {
		_, err := sh.ChunkGuild(guildID, false, nil)
		if err != nil {
			sh.Logger.Error().Err(err).Int64("guild_id", int64(guildID)).Msg("Failed to chunk guild")
		}
	}

	sh.Logger.Info().Int("guilds", len(guilds)).Msg("Finished chunking all guilds")
}

// ChunkGuilds chunks guilds to discord. It will wait for the operation to complete, or timeout.
func (sh *Shard) ChunkGuild(
	guildID discord.Snowflake,
	alwaysChunk bool,
	chunkReq *discord.RequestGuildMembers,
) (madeChunks bool, err error) {
	guildChunk, ok := sh.Sandwich.guildChunks.Load(guildID)

	if !ok {
		guildChunk = GuildChunks{
			Complete:        *atomic.NewBool(false),
			ChunkingChannel: make(chan *discord.GuildMembersChunk),
			StartedAt:       *atomic.NewTime(time.Time{}),
			CompletedAt:     *atomic.NewTime(time.Time{}),
			ChunkCount:      *atomic.NewInt32(0),
		}

		sh.Sandwich.guildChunks.Store(guildID, guildChunk)
	}

	guildChunk.Complete.Store(false)
	guildChunk.StartedAt.Store(time.Now())

	var memberCount int
	var needsChunking bool

	gm, ok := sh.Sandwich.State.GuildMembers.Load(guildID)

	if ok {
		memberCount = gm.Members.Count()
	}

	guild, ok := sh.Sandwich.State.Guilds.Load(guildID)

	if !ok {
		sh.Sandwich.Logger.Warn().Int64("guild_id", int64(guildID)).Msg("Guild not found in state")
		needsChunking = true
	} else {
		needsChunking = guild.MemberCount > int32(memberCount)
	}

	if needsChunking || alwaysChunk {
		var nonce string
		var req *discord.RequestGuildMembers

		if chunkReq != nil {
			nonce = chunkReq.Nonce
			req = chunkReq
		} else {
			nonce = randomHex(16)
			req = &discord.RequestGuildMembers{
				GuildID: guildID,
				Nonce:   nonce,
			}
		}

		err := sh.SendEvent(sh.ctx, discord.GatewayOpRequestGuildMembers, req)
		if err != nil {
			return false, fmt.Errorf("failed to send request guild members event: %w", err)
		}

		chunksReceived := int32(0)
		totalChunks := int32(0)

		timeout := time.NewTimer(MemberChunkTimeout)

	guildChunkLoop:
		for {
			select {
			case guildChunk := <-guildChunk.ChunkingChannel:
				if guildChunk.Nonce != nonce {
					continue
				}

				chunksReceived++
				totalChunks = guildChunk.ChunkCount

				// When receiving a chunk, reset the timeout.
				timeout.Reset(MemberChunkTimeout)

				sh.Logger.Debug().
					Int64("guild_id", int64(guildID)).
					Int32("chunk_index", guildChunk.ChunkIndex).
					Int32("chunk_count", guildChunk.ChunkCount).
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

	return needsChunking || alwaysChunk, nil
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

	err := sh.Manager.Sandwich.PublishGlobalEvent(sandwich_structs.SandwichEventShardStatusUpdate, json.RawMessage(payload))
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to publish shard status update")
	}
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
