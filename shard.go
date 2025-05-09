package sandwich

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/url"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/RealRock/limiter"
	"github.com/WelcomerTeam/czlib"
	"github.com/coder/websocket"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

var (
	// Number of retries to attempt before giving up on a shard
	ShardConnectRetries = int32(3)

	// Number of heartbeats that can fail before the shard is considered dead
	ShardMaxHeartbeatFailures = int32(5)

	GatewayLargeThreshold = int32(100)

	MemberChunkTimeout = time.Second * 3
)

var gatewayURL = url.URL{
	Scheme: "wss",
	Host:   "gateway.discord.gg",
}

type Shard struct {
	logger *slog.Logger

	sandwich *Sandwich
	manager  *Manager

	shardID int32

	retriesRemaining *atomic.Int32
	startedAt        *atomic.Pointer[time.Time]
	initializedAt    *atomic.Pointer[time.Time]

	heartbeatActive   *atomic.Bool
	lastHeartbeatAck  *atomic.Pointer[time.Time]
	lastHeartbeatSent *atomic.Pointer[time.Time]

	heartbeater              *time.Ticker
	heartbeatInterval        *atomic.Pointer[time.Duration]
	heartbeatFailureInterval *atomic.Pointer[time.Duration]

	unavailableGuilds *csmap.CsMap[discord.Snowflake, bool]
	lazyGuilds        *csmap.CsMap[discord.Snowflake, bool]
	guilds            *csmap.CsMap[discord.Snowflake, bool]

	sequence  *atomic.Int32
	sessionID *atomic.Pointer[string]

	websocketConn *websocket.Conn

	websocketRatelimit *limiter.DurationLimiter

	resumeGatewayURL *atomic.Pointer[string]

	ready chan struct{}
	stop  chan struct{}
	error chan error
}

func NewShard(sandwich *Sandwich, manager *Manager, shardID int32) *Shard {
	shard := &Shard{
		logger: manager.logger.With("shard_id", shardID),

		sandwich: sandwich,
		manager:  manager,

		shardID: shardID,

		retriesRemaining: &atomic.Int32{},
		startedAt:        &atomic.Pointer[time.Time]{},
		initializedAt:    &atomic.Pointer[time.Time]{},

		heartbeatActive:   &atomic.Bool{},
		lastHeartbeatAck:  &atomic.Pointer[time.Time]{},
		lastHeartbeatSent: &atomic.Pointer[time.Time]{},

		heartbeater:              nil,
		heartbeatInterval:        &atomic.Pointer[time.Duration]{},
		heartbeatFailureInterval: &atomic.Pointer[time.Duration]{},

		unavailableGuilds: csmap.Create[discord.Snowflake, bool](),
		lazyGuilds:        csmap.Create[discord.Snowflake, bool](),
		guilds:            csmap.Create[discord.Snowflake, bool](),

		sequence:  &atomic.Int32{},
		sessionID: &atomic.Pointer[string]{},

		websocketConn: nil,

		// We have a ratelimit of 120 messages per minutes we can send to the gateway.
		// We subtract 2 to account for heartbeating.
		websocketRatelimit: limiter.NewDurationLimiter(120-2, time.Minute),

		resumeGatewayURL: &atomic.Pointer[string]{},

		ready: make(chan struct{}, 1),
		stop:  make(chan struct{}, 1),
		error: make(chan error, 1),
	}

	shard.retriesRemaining.Store(ShardConnectRetries)

	now := time.Now()
	shard.initializedAt.Store(&now)

	return shard
}

func (shard *Shard) ConnectWithRetry(ctx context.Context) error {
	for {
		err := shard.Connect(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			newValue := shard.retriesRemaining.Add(-1)
			if newValue <= 0 {
				return fmt.Errorf("%w: %w", ErrShardConnectFailed, err)
			}

			shard.logger.Error("Failed to connect to shard", "error", err, "retries_remaining", newValue)
		} else if err == nil {
			break
		}
	}

	return nil
}

func (shard *Shard) Connect(ctx context.Context) error {
	shard.logger.Debug("Shard is connecting")

	// Empties the ready channel.
readyConsumer:
	for {
		select {
		case <-shard.ready:
		default:
			break readyConsumer
		}
	}

	var err error

	defer func() {
		if err != nil {
			if shard.websocketConn != nil {
				shard.closeWS(ctx, websocket.StatusNormalClosure)
			}
		}
	}()

	var websocketURL string

	resumeGatewayURL := shard.resumeGatewayURL.Load()
	if resumeGatewayURL == nil || *resumeGatewayURL == "" {
		websocketURL = gatewayURL.String()
	} else {
		websocketURL = *resumeGatewayURL
	}

	if shard.websocketConn != nil {
		err = shard.closeWS(ctx, websocket.StatusNormalClosure)
		if err != nil {
			shard.logger.Error("Failed to close websocket", "error", err)

			return fmt.Errorf("failed to close websocket: %w", err)
		}
	}

	// We need to append the v10 and encoding=json to the URL.
	websocketURL = websocketURL + "?v=10&encoding=json"

	shard.logger.Debug("Dialing websocket", "url", websocketURL)

	conn, _, err := websocket.Dial(ctx, websocketURL, nil)
	if err != nil {
		shard.logger.Error("Failed to dial websocket", "error", err)

		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	conn.SetReadLimit(-1)

	// TODO: how can i improve this?
	shard.websocketConn = conn

	// Read the initial payload
	payload, err := shard.read(ctx, conn)
	if err != nil {
		shard.logger.Error("Failed to read initial payload", "error", err)

		return fmt.Errorf("failed to read initial payload: %w", err)
	}

	var hello discord.Hello

	err = unmarshalPayload(payload, &hello)
	if err != nil {
		shard.logger.Error("Failed to unmarshal hello", "error", err)

		return fmt.Errorf("failed to unmarshal hello: %w", err)
	}

	if hello.HeartbeatInterval <= 0 {
		return ErrShardInvalidHeartbeatInterval
	}

	now := time.Now()
	shard.startedAt.Store(&now)
	shard.lastHeartbeatAck.Store(&now)
	shard.lastHeartbeatSent.Store(&now)

	heartbeatInterval := time.Duration(hello.HeartbeatInterval) * time.Millisecond
	shard.heartbeatInterval.Store(&heartbeatInterval)

	heartbeatFailureInterval := heartbeatInterval * time.Duration(ShardMaxHeartbeatFailures)
	shard.heartbeatFailureInterval.Store(&heartbeatFailureInterval)

	shard.logger.Debug("Shard is ready", "heartbeat_interval", heartbeatInterval.Milliseconds(), "heartbeat_failure_interval", heartbeatFailureInterval.Milliseconds())

	go shard.heartbeat(ctx)

	sequence := shard.sequence.Load()
	sessionID := shard.sessionID.Load()

	if sequence == 0 || (sessionID == nil || *sessionID == "") {
		err = shard.identify(ctx)
		if err != nil {
			return fmt.Errorf("failed to identify: %w", err)
		}
	} else {
		err = shard.resume(ctx)
		if err != nil {
			return fmt.Errorf("failed to resume: %w", err)
		}
	}

	return nil
}

func (shard *Shard) Start(ctx context.Context) error {
	shard.logger.Debug("Shard is starting")

	for {
		err := shard.Listen(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrShardStopping) {
				return nil
			}

			shard.error <- err

			var closeError websocket.CloseError

			// If the status code is not recoverable, we need to return the error.
			if ok := errors.As(err, &closeError); ok {
				if !isStatusCodeRecoverable(closeError.Code) {
					return err
				}
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
}

func (shard *Shard) Stop(ctx context.Context, code websocket.StatusCode) {
	shard.logger.Debug("Shard is stopping")

	shard.stop <- struct{}{}

	shard.manager.producer.Close()
	shard.closeWS(ctx, code)
}

func (shard *Shard) Listen(ctx context.Context) error {
	shard.logger.Debug("Shard is listening")

	websocketConn := shard.websocketConn

	for {
		msg, err := shard.read(ctx, websocketConn)

		select {
		case <-shard.stop:
			return ErrShardStopping
		case <-ctx.Done():
			return nil
		default:
		}

		if err == nil {
			trace := Trace{}
			trace.Set("receive", time.Now().UnixNano())

			err = shard.OnEvent(ctx, msg, &trace)
			if err != nil {
				shard.logger.Error("Failed to handle event", "error", err)
			}

			continue
		}

		// If the context is done, we can just return.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}

		var closeError websocket.CloseError

		if ok := errors.As(err, &closeError); ok {
			if !isStatusCodeRecoverable(closeError.Code) {
				shard.logger.Error("Shard received close event", "error", closeError)

				return fmt.Errorf("shard %d received close event: %w", shard.shardID, closeError)
			}
		}

		msgs, merr := json.Marshal(msg)
		if merr != nil {
			shard.logger.Error("Failed to marshal message", "error", merr)
		}

		shard.logger.Error("Shard received error", "error", err, "message", string(msgs))

		// If the websocket connection is the same as the one we're using, we need to reconnect.
		if websocketConn == shard.websocketConn {
			err = shard.reconnect(ctx, websocket.StatusNormalClosure)
			if err != nil {
				shard.logger.Error("Failed to reconnect", "error", err)

				return err
			}
		}

		return nil
	}
}

func isStatusCodeRecoverable(code websocket.StatusCode) bool {
	return code != discord.CloseNotAuthenticated &&
		code != discord.CloseAuthenticationFailed &&
		code != discord.CloseAlreadyAuthenticated &&
		code != discord.CloseInvalidShard &&
		code != discord.CloseShardingRequired &&
		code != discord.CloseInvalidAPIVersion &&
		code != discord.CloseInvalidIntents &&
		code != discord.CloseDisallowedIntents
}

func (shard *Shard) reconnect(ctx context.Context, code websocket.StatusCode) error {
	shard.logger.Debug("Shard is reconnecting")

	err := shard.closeWS(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to close websocket: %w", err)
	}

	wait := time.Second

	for {
		err := shard.Connect(ctx)
		if err == nil {
			shard.retriesRemaining.Store(ShardConnectRetries)

			return nil
		}

		retries := shard.retriesRemaining.Add(-1)
		if retries <= 0 {
			_ = shard.closeWS(ctx, code)

			err = shard.Connect(ctx)
			if err != nil {
				return fmt.Errorf("failed to reconnect: %w", err)
			}

			return nil
		}

		time.Sleep(wait)

		wait *= 2
		if wait > time.Minute {
			wait = time.Minute
		}
	}
}

func (shard *Shard) closeWS(_ context.Context, code websocket.StatusCode) error {
	shard.logger.Debug("Shard is closing websocket", "code", code)

	if shard.websocketConn == nil {
		return nil
	}

	err := shard.websocketConn.Close(code, "")
	if err != nil && !errors.Is(err, net.ErrClosed) {
		shard.logger.Error("Failed to close websocket", "error", err)
	}

	return nil
}

func (shard *Shard) waitForReady() error {
	shard.logger.Debug("Shard is waiting for ready")

	since := time.Now()
	ticker := time.NewTicker(time.Second * 15)

	for {
		select {
		case <-shard.ready:
			return nil
		case err := <-shard.error:
			return err
		case <-ticker.C:
			shard.logger.Error("Shard not ready", "duration", time.Since(since))
		}
	}
}

func (shard *Shard) heartbeat(ctx context.Context) {
	shard.logger.Debug("Shard is heartbeating")

	shard.heartbeatActive.Store(true)
	defer shard.heartbeatActive.Store(false)

	// We will use a jitter to avoid the heartbeat interval from being the same for all shards.
	hasJitter := true
	heartbeatJitter := time.Millisecond * time.Duration(rand.Int64N(shard.heartbeatInterval.Load().Milliseconds()+1))

	if shard.heartbeater == nil {
		shard.heartbeater = time.NewTicker(heartbeatJitter)
	} else {
		shard.heartbeater.Reset(heartbeatJitter)
	}

	shard.logger.Debug("Shard is heartbeating", "heartbeat_jitter", int(heartbeatJitter.Milliseconds()))

	for {
		select {
		case <-ctx.Done():
			return
		case <-shard.heartbeater.C:
			if hasJitter {
				hasJitter = false

				shard.heartbeater.Reset(*shard.heartbeatInterval.Load())

				shard.logger.Debug("Shard is heartbeating", "heartbeat_interval", *shard.heartbeatInterval.Load())
			}

			shard.logger.Debug("Sending heartbeat", "sequence", shard.sequence.Load())

			err := shard.SendEvent(ctx, discord.GatewayOpHeartbeat, shard.sequence.Load())

			now := time.Now()
			shard.lastHeartbeatSent.Store(&now)

			if err != nil || now.Sub(*shard.lastHeartbeatAck.Load()) > *shard.heartbeatFailureInterval.Load() {
				if err != nil {
					shard.logger.Error("Heartbeat failed", "error", err)
				} else {
					shard.logger.Error("Heartbeat failed", "error", "timeout")
				}

				return
			}
		}
	}
}

func (shard *Shard) identify(ctx context.Context) error {
	configuration := shard.manager.configuration.Load()
	shardCount := shard.manager.shardCount.Load()

	shard.logger.Debug("Shard is identifying", "shard_id", shard.shardID, "shard_count", shardCount)

	shard.manager.gatewaySessionStartLimitRemaining.Add(-1)

	err := shard.waitForIdentify(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for identify: %w", err)
	}

	return shard.SendEvent(ctx, discord.GatewayOpIdentify, discord.Identify{
		Properties: discord.IdentifyProperties{
			OS:      runtime.GOOS,
			Browser: "Sandwich " + Version,
			Device:  "Sandwich " + Version,
		},
		Presence:       &configuration.DefaultPresence,
		Token:          configuration.BotToken,
		Shard:          [2]int32{shard.shardID, shardCount},
		LargeThreshold: GatewayLargeThreshold,
		Intents:        configuration.Intents,
		Compress:       true,
	})
}

func (shard *Shard) waitForIdentify(ctx context.Context) error {
	shard.logger.Debug("Shard is waiting for identify")

	err := shard.sandwich.identifyProvider.Identify(ctx, shard)
	if err != nil {
		return fmt.Errorf("failed to identify: %w", err)
	}

	return nil
}

func (shard *Shard) resume(ctx context.Context) error {
	shard.logger.Debug("Shard is resuming")

	configuration := shard.manager.configuration.Load()

	return shard.SendEvent(ctx, discord.GatewayOpResume, discord.Resume{
		Token:     configuration.BotToken,
		SessionID: *shard.sessionID.Load(),
		Sequence:  shard.sequence.Load(),
	})
}

func (shard *Shard) SendEvent(ctx context.Context, gatewayOp discord.GatewayOp, data any) error {
	packet := discord.SentPayload{
		Op:   gatewayOp,
		Data: data,
	}

	return shard.send(ctx, gatewayOp, packet)
}

func (shard *Shard) send(ctx context.Context, gatewayOp discord.GatewayOp, data any) error {
	defer func() {
		if r := recover(); r != nil {
			if shard.sandwich.panicHandler != nil {
				shard.sandwich.panicHandler(shard.sandwich, r)
			}
		}
	}()

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// We don't need to ratelimit heartbeats.
	if gatewayOp != discord.GatewayOpHeartbeat {
		shard.websocketRatelimit.Lock()
	}

	shard.logger.Debug("Sending payload", "payload", string(payload))

	err = shard.websocketConn.Write(ctx, websocket.MessageText, payload)
	if err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return nil
}

func (shard *Shard) read(ctx context.Context, websocketConn *websocket.Conn) (discord.GatewayPayload, error) {
	messageType, data, err := websocketConn.Read(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return discord.GatewayPayload{}, context.Canceled
		}

		return discord.GatewayPayload{}, fmt.Errorf("failed to read message: %w", err)
	}

	if messageType == websocket.MessageBinary {
		data, err = czlib.Decompress(data)
		if err != nil {
			return discord.GatewayPayload{}, fmt.Errorf("failed to decompress payload: %w", err)
		}
	}

	var payload discord.GatewayPayload

	err = json.Unmarshal(data, &payload)
	if err != nil {
		return discord.GatewayPayload{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return payload, nil
}

func (shard *Shard) OnEvent(ctx context.Context, msg discord.GatewayPayload, trace *Trace) error {
	if f, ok := gatewayEvents[msg.Op]; ok {
		return f(ctx, shard, msg, trace)
	}

	return ErrNoGatewayHandler
}

func (shard *Shard) OnDispatch(ctx context.Context, msg discord.GatewayPayload, trace *Trace) error {
	defer func() {
		if r := recover(); r != nil {
			if shard.sandwich.panicHandler != nil {
				shard.sandwich.panicHandler(shard.sandwich, r)
			}
		}
	}()

	// Dispatch the event to the event provider.
	err := shard.sandwich.eventProvider.Dispatch(ctx, shard, msg, trace)
	if err != nil {
		shard.logger.Error("Failed to dispatch event", "error", err)
	}

	return nil
}

func (shard *Shard) chunkAllGuilds(ctx context.Context) {
	shard.logger.Debug("Chunking all guilds")

	guildIDs := make([]discord.Snowflake, 0)

	shard.guilds.Range(func(key discord.Snowflake, _ bool) bool {
		guildIDs = append(guildIDs, key)

		return true
	})

	shard.logger.Debug("Chunking all guilds", "count", len(guildIDs))

	for _, guildID := range guildIDs {
		err := shard.chunkGuild(ctx, guildID, false)
		if err != nil {
			shard.logger.Error("Failed to chunk guild", "error", err)
		}
	}

	shard.logger.Debug("Chunked all guilds")
}

func (shard *Shard) chunkGuild(ctx context.Context, guildID discord.Snowflake, always bool) error {
	shard.logger.Debug("Chunking guild", "guildID", guildID)

	guildChunk, ok := shard.sandwich.guildChunks.Load(guildID)

	if !ok {
		guildChunk = &GuildChunk{
			complete:        &atomic.Bool{},
			chunkingChannel: make(chan GuildChunkPartial),
			startedAt:       &atomic.Pointer[time.Time]{},
			completedAt:     &atomic.Pointer[time.Time]{},
		}

		shard.sandwich.guildChunks.Store(guildID, guildChunk)
	}

	guildChunk.complete.Store(false)

	now := time.Now()
	guildChunk.startedAt.Store(&now)

	guildMembers, _ := shard.sandwich.stateProvider.GetGuildMembers(ctx, guildID)
	memberCount := len(guildMembers)

	guild, _ := shard.sandwich.stateProvider.GetGuild(ctx, guildID)

	if always || int(guild.MemberCount) > memberCount {
		nonce := randomHex(16)

		err := shard.SendEvent(ctx, discord.GatewayOpRequestGuildMembers, discord.RequestGuildMembers{
			GuildID: guildID,
			Nonce:   nonce,
		})
		if err != nil {
			return fmt.Errorf("failed to request guild members: %w", err)
		}

		var chunksReceived int32

		var totalChunks int32

		timeout := time.NewTimer(MemberChunkTimeout)

	guildChunkLoop:
		for {
			select {
			case guildChunkPartial := <-guildChunk.chunkingChannel:
				if guildChunkPartial.nonce != nonce {
					continue
				}

				chunksReceived++
				totalChunks = guildChunkPartial.chunkCount

				// Reset the timeout.
				timeout.Reset(MemberChunkTimeout)

				shard.logger.Debug("Received chunk", "chunksReceived", chunksReceived, "totalChunks", totalChunks)

				if chunksReceived >= totalChunks {
					shard.logger.Debug("Received all chunks", "guildID", guildID)

					break guildChunkLoop
				}
			case <-timeout.C:
				shard.logger.Error("Timeout while waiting for guild members", "guildID", guildID)

				break guildChunkLoop
			}
		}

		timeout.Stop()
	}

	guildChunk.complete.Store(true)

	now = time.Now()
	guildChunk.completedAt.Store(&now)

	shard.logger.Debug("Chunked guild", "guildID", guildID)

	return nil
}
