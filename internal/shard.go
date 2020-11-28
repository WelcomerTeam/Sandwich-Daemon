package gateway

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/TheRockettek/czlib"
	"github.com/rs/zerolog"
	"github.com/savsgio/gotils"
	"github.com/tevino/abool"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

const timeoutDuration = 2 * time.Second

const dispatchTimeout = 30 * time.Second

const waitForReadyTimeout = 10 * time.Second

const identifyRatelimit = (5 * time.Second) + (500 * time.Millisecond)

const websocketReadLimit = 512 << 20

const reconnectCloseCode = 4000

const maxReconnectWait = 600

// Shard represents the shard object.
type Shard struct {
	Status   structs.ShardStatus `json:"status"`
	StatusMu sync.RWMutex        `json:"-"`

	Logger zerolog.Logger `json:"-"`

	ShardID    int         `json:"shard_id"`
	ShardGroup *ShardGroup `json:"-"`
	Manager    *Manager    `json:"-"`

	User *structs.User `json:"user"`
	// Todo: Add deque that can allow for an event queue (maybe).

	ctx    context.Context
	cancel func()

	HeartbeatActive   *abool.AtomicBool `json:"-"`
	LastHeartbeatMu   sync.RWMutex      `json:"-"`
	LastHeartbeatAck  time.Time         `json:"last_heartbeat_ack"`
	LastHeartbeatSent time.Time         `json:"last_heartbeat_sent"`

	Heartbeater          *time.Ticker  `json:"-"`
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`
	MaxHeartbeatFailures time.Duration `json:"max_heartbeat_failures"`

	Unavailable map[snowflake.ID]bool `json:"-"`

	Start   time.Time `json:"start"`
	Retries *int32    `json:"retries"` // When erroring, how many times to retry connecting until shardgroup is stopped.

	wsConn *websocket.Conn

	rp sync.Pool
	pp sync.Pool

	MessageCh chan structs.ReceivedPayload
	ErrorCh   chan error

	events *int64

	seq       *int64
	sessionID string

	// Channel that dictates if the shard has been made ready.
	ready chan void

	// Channel to pipe errors.
	errs chan error
}

// NewShard creates a new shard object.
func (sg *ShardGroup) NewShard(shardID int) *Shard {
	logger := sg.Logger.With().Int("shard", shardID).Logger()
	sh := &Shard{
		Status:   structs.ShardIdle,
		StatusMu: sync.RWMutex{},

		Logger: logger,

		ShardID:    shardID,
		ShardGroup: sg,
		Manager:    sg.Manager,

		HeartbeatActive:   abool.New(),
		LastHeartbeatMu:   sync.RWMutex{},
		LastHeartbeatAck:  time.Now().UTC(),
		LastHeartbeatSent: time.Now().UTC(),

		Start:   time.Now().UTC(),
		Retries: new(int32),

		rp: sync.Pool{
			New: func() interface{} { return new(structs.SentPayload) },
		},
		pp: sync.Pool{
			New: func() interface{} { return new(structs.PublishEvent) },
		},

		events: new(int64),

		seq:       new(int64),
		sessionID: "",

		ready: make(chan void, 1),

		errs: make(chan error),
	}

	if sh.ctx == nil || sh.cancel == nil {
		sh.ctx, sh.cancel = context.WithCancel(context.Background())
	}

	atomic.StoreInt32(sh.Retries, sg.Manager.Configuration.Bot.Retries)

	return sh
}

// Open starts up the shard connection.
func (sh *Shard) Open() {
	for {
		err := sh.Listen()
		if xerrors.Is(err, context.Canceled) {
			return
		}

		// Check if context is done
		select {
		case <-sh.ctx.Done():
			return
		default:
		}
	}
}

// Connect connects to the gateway such as identifying however does not listen to new messages.
func (sh *Shard) Connect() (err error) {
	sh.Logger.Debug().Msg("Starting shard")

	if err := sh.SetStatus(structs.ShardWaiting); err != nil {
		sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
	}

	sh.Manager.GatewayMu.RLock()

	// Fetch the current bucket we should be using for concurrency.
	concurrencyBucket := sh.ShardID % sh.Manager.Gateway.SessionStartLimit.MaxConcurrency

	if _, ok := sh.ShardGroup.IdentifyBucket[concurrencyBucket]; !ok {
		sh.ShardGroup.IdentifyBucket[concurrencyBucket] = &sync.Mutex{}
	}

	// If the context has canceled, create new context.
	select {
	case <-sh.ctx.Done():
		sh.ctx, sh.cancel = context.WithCancel(context.Background())
	default:
	}

	// Create and wait for the websocket bucket.
	sh.Logger.Trace().Msg("Creating buckets")
	sh.Manager.Buckets.CreateBucket(fmt.Sprintf("ws:%d:%d", sh.ShardID, sh.ShardGroup.ShardCount), 120, time.Minute)

	hash, err := QuickHash(sh.Manager.Configuration.Token)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to generate token hash")

		return err
	}

	sh.Manager.Sandwich.Buckets.CreateBucket(fmt.Sprintf("gw:%s:%d", hash, concurrencyBucket), 1, identifyRatelimit)
	sh.Manager.GatewayMu.RUnlock()

	// When an error occurs and we have to reconnect, we make a ready channel by default
	// which seems to cause a problem with WaitForReady. To circumvent this, we will
	// make the ready only when the channel is closed however this may not be necessary
	// as there is now a loop that fires every 10 seconds meaning it will be up to date regardless.

	if sh.ready == nil {
		sh.ready = make(chan void, 1)
	}

	sh.Manager.GatewayMu.RLock()
	gatewayURL := sh.Manager.Gateway.URL
	sh.Manager.GatewayMu.RUnlock()

	defer func() {
		if err != nil && sh.wsConn != nil {
			if err = sh.CloseWS(websocket.StatusNormalClosure); err != nil {
				sh.Logger.Error().Err(err).Msg("Failed to close websocket")
			}
		}
	}()

	// Todo: Add Concurrent Client Support.
	// This will limit the amount of shards that can be connecting simultaneously.
	// Currently just uses a mutex to allow for only one per maxconcurrency.
	sh.Logger.Trace().Msg("Waiting for identify mutex")

	// Lock the identification bucket
	sh.ShardGroup.IdentifyBucket[concurrencyBucket].Lock()
	defer sh.ShardGroup.IdentifyBucket[concurrencyBucket].Unlock()

	sh.Logger.Trace().Msg("Starting connecting")

	if err := sh.SetStatus(structs.ShardConnecting); err != nil {
		sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
	}

	// If there is no active ws connection, create a new connection to discord.
	if sh.wsConn == nil {
		var errorCh chan error
		var messageCh chan structs.ReceivedPayload

		errorCh, messageCh, err = sh.FeedWebsocket(sh.ctx, gatewayURL, nil)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to dial")

			return
		}

		sh.ErrorCh = errorCh
		sh.MessageCh = messageCh

	} else {
		sh.Logger.Info().Msg("Reusing websocket connection")
	}

	sh.Logger.Trace().Msg("Reading from WS")

	// Read a message from WS which we should expect to be Hello
	msg, err := sh.readMessage(sh.ctx)

	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to read message")

		return
	}

	hello := structs.Hello{}
	err = sh.decodeContent(msg, &hello)

	sh.LastHeartbeatMu.Lock()
	sh.LastHeartbeatAck = time.Now().UTC()
	sh.LastHeartbeatSent = time.Now().UTC()
	sh.LastHeartbeatMu.Unlock()

	sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond

	sh.MaxHeartbeatFailures = sh.HeartbeatInterval * time.Duration(sh.Manager.Configuration.Bot.MaxHeartbeatFailures)

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Int("maxfails", sh.Manager.Configuration.Bot.MaxHeartbeatFailures).
		Msg("Retrieved HELLO event from discord")

	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)
	seq := atomic.LoadInt64(sh.seq)

	// If we have no session ID or the sequence is 0, we can identify instead
	// of resuming.
	if sh.sessionID == "" || seq == 0 {
		err = sh.Identify()
		if err := sh.SetStatus(structs.ShardConnecting); err != nil {
			sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
		}

		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to identify")

			return
		}
	} else {
		err = sh.Resume()
		// We will assume the bot is ready.
		if err := sh.SetStatus(structs.ShardReady); err != nil {
			sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
		}

		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to resume")

			return
		}
	}

	sh.Manager.ConfigurationMu.RLock()
	hash, err = QuickHash(sh.Manager.Configuration.Token)

	if err != nil {
		sh.Manager.ConfigurationMu.RUnlock()
		sh.Logger.Error().Err(err).Msg("Failed to generate token hash")

		return
	}

	// Reset the bucket we used for gateway
	bucket := fmt.Sprintf("gw:%s:%d", hash, sh.ShardID%sh.Manager.Gateway.SessionStartLimit.MaxConcurrency)
	sh.Manager.Buckets.ResetBucket(bucket)
	sh.Manager.ConfigurationMu.RUnlock()

	t := time.NewTicker(time.Second * 5)

	// Wait 5 seconds for the first event or errors in websocket to
	// ensure there are no error messages such as disallowed intents.

	if sh.HeartbeatActive.IsNotSet() {
		go sh.Heartbeat()
	}

	sh.Logger.Trace().Msg("Waiting for first event")
	select {
	case err = <-sh.ErrorCh:
		sh.Logger.Error().Err(err).Msg("Encountered error whilst connecting")
		return xerrors.Errorf("encountered error whilst connecting: %w", err)
	case msg = <-sh.MessageCh:
		if err = sh.OnEvent(msg); err != nil {
			sh.Logger.Error().Err(err).Msg("Encountered error dispatching event")
			return xerrors.Errorf("encountered error handling event: %w", err)
		}
	case <-t.C:
	}

	sh.Logger.Trace().Msg("Finished connecting")

	return err
}

// FeedWebsocket reads websocket events and feeds them through a channel.
func (sh *Shard) FeedWebsocket(ctx context.Context, u string,
	opts *websocket.DialOptions) (errorCh chan error, messageCh chan structs.ReceivedPayload, err error) {
	messageCh = make(chan structs.ReceivedPayload, 64)
	errorCh = make(chan error, 1)

	conn, _, err := websocket.Dial(ctx, u, opts)

	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to dial websocket")

		return errorCh, messageCh, xerrors.Errorf("failed to connect to websocket: %w", err)
	}

	conn.SetReadLimit(websocketReadLimit)
	sh.wsConn = conn

	go func() {
		var msg structs.ReceivedPayload

		for {
			mt, buf, err := conn.Read(ctx)

			select {
			case <-ctx.Done():
				return
			default:
			}

			if err != nil {
				errorCh <- xerrors.Errorf("readMessage read: %w", err)
				return
			}

			if mt == websocket.MessageBinary {
				buf, err = czlib.Decompress(buf)
				if err != nil {
					errorCh <- xerrors.Errorf("readMessage decompress: %w", err)
					return
				}
			}

			err = json.Unmarshal(buf, &msg)
			atomic.AddInt64(sh.events, 1)

			messageCh <- msg
		}
	}()

	return errorCh, messageCh, err
}

// OnEvent processes an event.
func (sh *Shard) OnEvent(msg structs.ReceivedPayload) (err error) {
	// This goroutine shows events that are taking too long.
	fin := make(chan void)

	go func() {
		since := time.Now()
		t := time.NewTimer(dispatchTimeout)

		for {
			select {
			case <-fin:
				return
			case <-t.C:
				sh.Logger.Warn().
					Str("type", msg.Type).
					Int("op", int(msg.Op)).
					Str("data", gotils.B2S(msg.Data)).
					Msgf("Event %s is taking too long. Been executing for %f seconds. Possible deadlock?",
						msg.Type, time.Since(since).
							Round(time.Second).Seconds(),
					)
				t.Reset(dispatchTimeout)
			}
		}
	}()

	defer func() {
		close(fin)
	}()

	switch msg.Op {
	case structs.GatewayOpHeartbeat:
		sh.Logger.Debug().Msg("Received heartbeat request")
		err = sh.SendEvent(structs.GatewayOpHeartbeat, atomic.LoadInt64(sh.seq))

		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to send heartbeat in response to gateway, reconnecting...")
			err = sh.Reconnect(websocket.StatusNormalClosure)

			if err != nil {
				sh.Logger.Error().Err(err).Msg("Failed to reconnect")
			}

			return
		}
	case structs.GatewayOpInvalidSession:
		resumable := json.Get(msg.Data, "d").ToBool()
		if !resumable {
			sh.sessionID = ""
			atomic.StoreInt64(sh.seq, 0)
		}

		sh.Logger.Warn().Bool("resumable", resumable).Msg("Received invalid session from gateway")
		err = sh.Reconnect(reconnectCloseCode)

		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to reconnect")
		}
	case structs.GatewayOpHello:
		hello := structs.Hello{}
		err = sh.decodeContent(msg, &hello)

		sh.LastHeartbeatMu.Lock()
		sh.LastHeartbeatAck = time.Now().UTC()
		sh.LastHeartbeatSent = time.Now().UTC()
		sh.LastHeartbeatMu.Unlock()

		sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond
		sh.MaxHeartbeatFailures = sh.HeartbeatInterval * time.Duration(sh.Manager.Configuration.Bot.MaxHeartbeatFailures)

		sh.Logger.Debug().
			Dur("interval", sh.HeartbeatInterval).
			Int("maxfails", sh.Manager.Configuration.Bot.MaxHeartbeatFailures).
			Msg("Retrieved HELLO event from discord")

		sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

		return
	case structs.GatewayOpReconnect:
		sh.Logger.Info().Msg("Reconnecting in response to gateway")
		err = sh.Reconnect(reconnectCloseCode)

		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to reconnect")
		}

		return
	case structs.GatewayOpDispatch:
		err = sh.OnDispatch(msg)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Error whilst dispatch event")

			return
		}
	case structs.GatewayOpHeartbeatACK:
		sh.LastHeartbeatMu.Lock()
		sh.LastHeartbeatAck = time.Now().UTC()
		sh.Logger.Debug().
			Int64("RTT", sh.LastHeartbeatAck.Sub(sh.LastHeartbeatSent).Milliseconds()).
			Msg("Received heartbeat ACK")

		sh.LastHeartbeatMu.Unlock()

		return
	default:
		sh.Logger.Warn().
			Int("op", int(msg.Op)).
			Str("type", msg.Type).
			Msg("Gateway sent unknown packet")

		return
	}

	atomic.StoreInt64(sh.seq, msg.Sequence)

	return err
}

// OnDispatch handles a dispatch event.
func (sh *Shard) OnDispatch(msg structs.ReceivedPayload) (err error) {
	start := time.Now().UTC()

	defer func() {
		change := time.Now().UTC().Sub(start)
		if change > time.Second {
			sh.Logger.Warn().Msgf("%s took %d ms", msg.Type, change.Milliseconds())
		}
	}()

	switch msg.Type {
	case "READY":
		readyPayload := structs.Ready{}
		if err = sh.decodeContent(msg, &readyPayload); err != nil {
			return
		}

		sh.User = readyPayload.User
		sh.sessionID = readyPayload.SessionID
		sh.Logger.Info().Msg("Received READY payload")

		sh.Unavailable = make(map[snowflake.ID]bool)
		events := make([]structs.ReceivedPayload, 0)

		guildIDs := make([]int64, 0)

		for _, guild := range readyPayload.Guilds {
			sh.Unavailable[guild.ID] = guild.Unavailable
		}

		guildCreateEvents := 0

		// If true will only run events once finished loading.
		// Todo: Add to sandwich configuration.
		preemptiveEvents := false

		t := time.NewTicker(waitForReadyTimeout)

	ready:
		for {
			select {
			case err := <-sh.ErrorCh:
				if !xerrors.Is(err, context.Canceled) {
					sh.Logger.Error().Err(err).Msg("Errored whilst waiting lazy loading")
				}

				break ready
			case msg := <-sh.MessageCh:
				switch msg.Type {
				case "GUILD_CREATE":
					guildCreateEvents++

					guildPayload := structs.GuildCreate{}

					if err = sh.decodeContent(msg, &guildPayload); err != nil {
						sh.Logger.Error().Err(err).Msg("Failed to unmarshal GUILD_CREATE")
					} else {
						guildIDs = append(guildIDs, guildPayload.ID.Int64())
					}

					if _, err = sh.Manager.StateGuildCreate(guildPayload); err != nil {
						sh.Logger.Error().Err(err).Msg("Failed to handle GUILD_CREATE event")
					}

					t.Reset(waitForReadyTimeout)
				default:
					if preemptiveEvents {
						events = append(events, msg)
					} else if err = sh.OnDispatch(msg); err != nil {
						sh.Logger.Error().Err(err).Msg("Failed dispatching event")
					}
				}
			case <-t.C:
				sh.Manager.Sandwich.ConfigurationMu.RLock()

				if sh.Manager.Configuration.Caching.RequestMembers {
					var chunk []int64

					chunkSize := sh.Manager.Configuration.Caching.RequestChunkSize

					for len(guildIDs) >= chunkSize {
						chunk, guildIDs = guildIDs[:chunkSize], guildIDs[chunkSize:]

						sh.Logger.Trace().Msgf("Requesting guild members for %d guild(s)", len(chunk))

						if err := sh.SendEvent(structs.GatewayOpRequestGuildMembers, structs.RequestGuildMembers{
							GuildID: chunk,
							Query:   "",
							Limit:   0,
						}); err != nil {
							sh.Logger.Error().Err(err).Msgf("Failed to request guild members")
						}
					}

					if len(guildIDs) > 0 {
						sh.Logger.Trace().Msgf("Requesting guild members for %d guild(s)", len(chunk))

						if err := sh.SendEvent(structs.GatewayOpRequestGuildMembers, structs.RequestGuildMembers{
							GuildID: guildIDs,
							Query:   "",
							Limit:   0,
						}); err != nil {
							sh.Logger.Error().Err(err).Msgf("Failed to request guild members")
						}
					}
				}
				sh.Manager.Sandwich.ConfigurationMu.RUnlock()

				break ready
			}
		}

		sh.ready <- void{}
		if err := sh.SetStatus(structs.ShardReady); err != nil {
			sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
		}

		if preemptiveEvents {
			sh.Logger.Debug().Int("events", len(events)).Msg("Dispatching preemptive events")

			for _, event := range events {
				sh.Logger.Debug().Str("type", event.Type).Send()

				if err = sh.OnDispatch(event); err != nil {
					sh.Logger.Error().Err(err).Msg("Failed whilst dispatching preemptive events")
				}
			}

			sh.Logger.Debug().Msg("Finished dispatching events")
		}

		return

	case "GUILD_CREATE":
		guildPayload := structs.GuildCreate{}
		if err = sh.decodeContent(msg, &guildPayload); err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to unmarshal GUILD_CREATE")
		} else if err = sh.HandleGuildCreate(guildPayload, false); err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to process guild create")
		}

		return

	case "GUILD_MEMBERS_CHUNK":
		guildMembersPayload := structs.GuildMembersChunk{}
		if err = sh.decodeContent(msg, &guildMembersPayload); err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to unmarshal GUILD_MEMBERS_CHUNK")
		} else if err = sh.Manager.StateGuildMembersChunk(guildMembersPayload); err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to process state")
		}

		return

	default:
	}

	// Todo: Uncomment this when we have proper event handling
	// default:
	// 	sh.Logger.Warn().Str("type", msg.Type).Msg("No handler for dispatch message")

	return err
}

// Latency returns the heartbeat latency in milliseconds.
func (sh *Shard) Latency() (latency int64) {
	sh.LastHeartbeatMu.RLock()
	defer sh.LastHeartbeatMu.RUnlock()

	return sh.LastHeartbeatAck.Sub(sh.LastHeartbeatSent).Round(time.Millisecond).Milliseconds()
}

// HandleGuildCreate handles the guild create event.
func (sh *Shard) HandleGuildCreate(payload structs.GuildCreate, lazy bool) (err error) {
	if ok, unavailable := sh.Unavailable[payload.ID]; ok && unavailable {
		// Guild has been lazy loaded
		sh.Logger.Trace().Msgf("Lazy loaded guild ID %d", payload.ID)
		payload.Lazy = true || lazy
	}
	delete(sh.Unavailable, payload.ID)

	_, err = sh.Manager.StateGuildCreate(payload)

	return
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

		msg, err := sh.readMessage(sh.ctx)
		if err != nil {
			if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
				break
			}

			sh.Logger.Error().Err(err).Msg("Error reading from gateway")

			var closeError *websocket.CloseError

			if errors.As(err, &closeError) {
				// If possible, we will check the close error to determine if we can continue
				switch closeError.Code {
				case structs.CloseNotAuthenticated: // Not authenticated
				case structs.CloseInvalidShard: // Invalid shard
				case structs.CloseShardingRequired: // Sharding required
				case structs.CloseInvalidAPIVersion: // Invalid API version
				case structs.CloseInvalidIntents: // Invalid Intent(s)
				case structs.CloseDisallowedIntents: // Disallowed intent(s)
					sh.Logger.Warn().Msgf(
						"Closing ShardGroup as cannot continue without valid token. Received code %d",
						closeError.Code,
					)

					// We cannot continue so we will kill the ShardGroup
					sh.ShardGroup.ErrorMu.Lock()
					sh.ShardGroup.Error = err.Error()
					sh.ShardGroup.ErrorMu.Unlock()
					sh.ShardGroup.Close()

					if err := sh.ShardGroup.SetStatus(structs.ShardGroupError); err != nil {
						sh.ShardGroup.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
					}

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

		err = sh.OnEvent(msg)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Encountered error dispatching event")
		}

		// In the event we have reconnected, the wsConn could have changed,
		// we will use the new wsConn if this is the case
		if sh.wsConn != wsConn {
			sh.Logger.Debug().Msg("New wsConn was assigned to shard")
			wsConn = sh.wsConn
		}
	}

	return err
}

// Heartbeat maintains a heartbeat with discord
// TODO: Make a shardgroup specific heartbeat function to heartbeat on behalf of all running shards.
func (sh *Shard) Heartbeat() {
	sh.HeartbeatActive.Set()
	defer sh.HeartbeatActive.UnSet()

	for {
		select {
		case <-sh.ctx.Done():
			return
		case <-sh.Heartbeater.C:
			sh.Logger.Debug().Msg("Heartbeating")
			seq := atomic.LoadInt64(sh.seq)

			err := sh.SendEvent(structs.GatewayOpHeartbeat, seq)

			sh.LastHeartbeatMu.Lock()
			_time := time.Now().UTC()
			sh.LastHeartbeatSent = _time
			lastAck := sh.LastHeartbeatAck
			sh.LastHeartbeatMu.Unlock()

			if err != nil || _time.Sub(lastAck) > sh.MaxHeartbeatFailures {
				if err != nil {
					sh.Logger.Error().Err(err).Msg("Failed to heartbeat. Reconnecting")
				} else {
					sh.Manager.Sandwich.ConfigurationMu.RLock()
					sh.Logger.Warn().Err(err).
						Msgf(
							"Gateway failed to ACK and has passed MaxHeartbeatFailures of %d. Reconnecting",
							sh.Manager.Configuration.Bot.MaxHeartbeatFailures)
					sh.Manager.Sandwich.ConfigurationMu.RUnlock()
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

// decodeContent converts the stored msg into the passed interface.
func (sh *Shard) decodeContent(msg structs.ReceivedPayload, out interface{}) (err error) {
	err = json.Unmarshal(msg.Data, &out)

	return
}

// readMessage fills the shard msg buffer from a websocket message.
func (sh *Shard) readMessage(ctx context.Context) (msg structs.ReceivedPayload, err error) {
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
		return msg, nil
	}
}

// CloseWS closes the websocket.
func (sh *Shard) CloseWS(statusCode websocket.StatusCode) (err error) {
	if sh.wsConn != nil {
		sh.Logger.Debug().Str("code", statusCode.String()).Msg("Closing websocket connection")
		err = sh.wsConn.Close(statusCode, "")
		sh.wsConn = nil
	}

	return
}

// Resume sends the resume packet to gateway.
func (sh *Shard) Resume() (err error) {
	sh.Logger.Debug().Msg("Sending resume")

	sh.Manager.Sandwich.ConfigurationMu.RLock()
	defer sh.Manager.Sandwich.ConfigurationMu.RUnlock()

	sh.Manager.ConfigurationMu.RLock()
	defer sh.Manager.ConfigurationMu.RUnlock()

	err = sh.SendEvent(structs.GatewayOpResume, structs.Resume{
		Token:     sh.Manager.Configuration.Token,
		SessionID: sh.sessionID,
		Sequence:  atomic.LoadInt64(sh.seq),
	})

	return
}

// Identify sends the identify packet to gateway.
func (sh *Shard) Identify() (err error) {
	sh.Logger.Debug().Msg("Sending identify")

	sh.Manager.GatewayMu.Lock()
	sh.Manager.Gateway.SessionStartLimit.Remaining--
	sh.Manager.GatewayMu.Unlock()

	sh.Manager.ConfigurationMu.RLock()
	defer sh.Manager.ConfigurationMu.RUnlock()

	sh.Manager.GatewayMu.RLock()

	hash, err := QuickHash(sh.Manager.Configuration.Token)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to generate token hash")
		sh.Manager.GatewayMu.RUnlock()

		return err
	}

	err = sh.Manager.Sandwich.Buckets.WaitForBucket(
		fmt.Sprintf("gw:%s:%d", hash, sh.ShardID%sh.Manager.Gateway.SessionStartLimit.MaxConcurrency),
	)

	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to wait for bucket")
	}

	sh.Manager.GatewayMu.RUnlock()

	err = sh.SendEvent(structs.GatewayOpIdentify, structs.Identify{
		Token: sh.Manager.Configuration.Token,
		Properties: &structs.IdentifyProperties{
			OS:      runtime.GOOS,
			Browser: "Sandwich " + VERSION,
			Device:  "Sandwich " + VERSION,
		},
		Compress:           sh.Manager.Configuration.Bot.Compression,
		LargeThreshold:     sh.Manager.Configuration.Bot.LargeThreshold,
		Shard:              [2]int{sh.ShardID, sh.ShardGroup.ShardCount},
		Presence:           sh.Manager.Configuration.Bot.DefaultPresence,
		GuildSubscriptions: sh.Manager.Configuration.Bot.GuildSubscriptions,
		Intents:            sh.Manager.Configuration.Bot.Intents,
	})

	return err
}

// PublishEvent sends an event to consumers.
func (sh *Shard) PublishEvent(eventType string, eventData interface{}) (err error) {
	packet := sh.pp.Get().(*structs.PublishEvent)
	defer sh.pp.Put(packet)

	sh.Manager.Sandwich.ConfigurationMu.RLock()
	defer sh.Manager.Sandwich.ConfigurationMu.RUnlock()

	packet.Data = eventData
	packet.From = sh.Manager.Configuration.Identifier
	packet.Type = eventType

	data, err := msgpack.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("publishEvent marshal: %w", err)
	}

	if sh.Manager.StanClient != nil {
		err = sh.Manager.StanClient.Publish(
			sh.Manager.Configuration.Messaging.ChannelName,
			data,
		)
		if err != nil {
			return xerrors.Errorf("publishEvent publish: %w", err)
		}
	} else {
		return xerrors.New("publishEvent publish: No active stanClient")
	}

	return nil
}

// SendEvent sends an event to discord.
func (sh *Shard) SendEvent(op structs.GatewayOp, data interface{}) (err error) {
	packet := sh.rp.Get().(*structs.SentPayload)
	defer sh.rp.Put(packet)

	packet.Op = int(op)
	packet.Data = data

	err = sh.WriteJSON(packet)
	if err != nil {
		return xerrors.Errorf("sendEvent writeJson: %w", err)
	}

	return
}

// WriteJSON writes json data to the websocket.
func (sh *Shard) WriteJSON(i interface{}) (err error) {
	res, err := json.Marshal(i)
	if err != nil {
		return xerrors.Errorf("writeJSON marshal: %w", err)
	}

	err = sh.Manager.Buckets.WaitForBucket(
		fmt.Sprintf("ws:%d:%d", sh.ShardID, sh.ShardGroup.ShardCount),
	)
	if err != nil {
		sh.Logger.Warn().Err(err).Msg("Tried to wait for websocket bucket but it does not exist")
	}

	sh.Manager.Sandwich.ConfigurationMu.RLock()
	sh.Logger.Trace().Msg(strings.ReplaceAll(gotils.B2S(res), sh.Manager.Configuration.Token, "..."))
	sh.Manager.Sandwich.ConfigurationMu.RUnlock()

	if sh.wsConn != nil {
		err = sh.wsConn.Write(sh.ctx, websocket.MessageText, res)
		if err != nil {
			return xerrors.Errorf("writeJSON write: %w", err)
		}
	}

	return
}

// WaitForReady waits until the shard is ready.
func (sh *Shard) WaitForReady() {
	since := time.Now().UTC()
	t := time.NewTicker(waitForReadyTimeout)

	for {
		select {
		case <-sh.ready:
			sh.Logger.Debug().Msg("Shard ready due to channel closure")

			return
		case <-sh.ctx.Done():
			sh.Logger.Debug().Msg("Shard ready due to context done")

			return
		case <-t.C:
			sh.StatusMu.RLock()
			status := sh.Status
			sh.StatusMu.RUnlock()

			if status == structs.ShardReady {
				sh.Logger.Warn().Msg("Shard ready due to status change")

				return
			}

			sh.Logger.Debug().
				Err(sh.ctx.Err()).
				Dur("since", time.Now().UTC().Sub(since).Round(time.Second)).
				Msg("Still waiting for shard to be ready")
		}
	}
}

// Reconnect attempts to reconnect to the gateway.
func (sh *Shard) Reconnect(code websocket.StatusCode) error {
	wait := time.Second

	sh.Close(code)

	if err := sh.SetStatus(structs.ShardReconnecting); err != nil {
		sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
	}

	for {
		sh.Logger.Info().Msg("Trying to reconnect to gateway")

		err := sh.Connect()
		if err == nil {
			atomic.StoreInt32(sh.Retries, sh.Manager.Configuration.Bot.Retries)
			sh.Logger.Info().Msg("Successfully reconnected to gateway")

			return nil
		}

		retries := atomic.AddInt32(sh.Retries, -1)
		if retries <= 0 {
			sh.Logger.Warn().Msg("Ran out of retries whilst connecting. Attempting to reconnect client.")
			sh.Close(code)
			err = sh.Connect()

			return err
		}

		sh.Logger.Warn().Err(err).Dur("retry", wait).Msg("Failed to reconnect to gateway")
		<-time.After(wait)

		wait *= 2
		if wait > maxReconnectWait {
			wait = maxReconnectWait
		}
	}
}

// SetStatus changes the Shard status.
func (sh *Shard) SetStatus(status structs.ShardStatus) (err error) {
	sh.StatusMu.Lock()
	sh.Status = status
	sh.StatusMu.Unlock()

	err = sh.PublishEvent("SHARD_STATUS", structs.MessagingStatusUpdate{ShardID: sh.ShardID, Status: int32(status)})

	return
}

// Close closes the shard connection.
func (sh *Shard) Close(code websocket.StatusCode) {
	// Ensure that if we close during shardgroup connecting, it will not
	// feedback loop.
	// cancel is only defined when Connect() has been ran on a shard.
	// If the ShardGroup was closed before this happens, it would segmentation fault.
	if sh.ctx != nil && sh.cancel != nil {
		sh.cancel()
	}

	if sh.wsConn != nil {
		if err := sh.CloseWS(code); err != nil {
			// It is highly common we are closing an already closed websocket
			// and at this point if we error closing it, its fair game. It would
			// be nice if the errAlreadyWroteClose error was public in the websocket
			// library so we could only suppress that error but what can you do.
			sh.Logger.Debug().Err(err).Msg("Encountered error closing websocket")
		}
	}

	if err := sh.SetStatus(structs.ShardClosed); err != nil {
		sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
	}
}
