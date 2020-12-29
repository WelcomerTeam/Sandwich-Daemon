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
	sync.RWMutex // used to lock less common variables such as the user

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

	UnavailableMu sync.RWMutex          `json:"-"`
	Unavailable   map[snowflake.ID]bool `json:"-"`

	Start   time.Time `json:"start"`
	Retries *int32    `json:"retries"` // When erroring, how many times to retry connecting until shardgroup is stopped.

	wsConn *websocket.Conn

	mp sync.Pool
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

		UnavailableMu: sync.RWMutex{},

		Start:   time.Now().UTC(),
		Retries: new(int32),

		// Pool of payloads from discord
		mp: sync.Pool{
			New: func() interface{} { return new(structs.ReceivedPayload) },
		},

		// Pool of payloads sent to discord
		rp: sync.Pool{
			New: func() interface{} { return new(structs.SentPayload) },
		},

		// Pool of payloads sent to consumers
		pp: sync.Pool{
			New: func() interface{} { return new(structs.SandwichPayload) },
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

	sh.Logger.Trace().Msgf("Using concurrency bucket %d", concurrencyBucket)

	// if _, ok := sh.ShardGroup.IdentifyBucket[concurrencyBucket]; !ok {
	// 	sh.Logger.Trace().Msgf("Creating new concurrency bucket %d", concurrencyBucket)
	// 	sh.ShardGroup.IdentifyBucket[concurrencyBucket] = &sync.Mutex{}
	// }

	sh.Manager.GatewayMu.RUnlock()

	// If the context has canceled, create new context.
	select {
	case <-sh.ctx.Done():
		sh.Logger.Trace().Msg("Creating new context")
		sh.ctx, sh.cancel = context.WithCancel(context.Background())
	default:
		sh.Logger.Trace().Msg("No need for new context")
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
			if _err := sh.CloseWS(websocket.StatusNormalClosure); _err != nil {
				sh.Logger.Error().Err(_err).Msg("Failed to close websocket")
			}
		}
	}()

	// Todo: Add Concurrent Client Support.
	// This will limit the amount of shards that can be connecting simultaneously.
	// Currently just uses a mutex to allow for only one per maxconcurrency.
	sh.Logger.Trace().Msg("Waiting for identify mutex")

	// // Lock the identification bucket
	// sh.ShardGroup.IdentifyBucket[concurrencyBucket].Lock()
	// defer sh.ShardGroup.IdentifyBucket[concurrencyBucket].Unlock()

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

			go sh.PublishWebhook(fmt.Sprintf("Failed to dial `%s`", gatewayURL), err.Error(), 14431557, false)

			return
		}

		sh.Lock()
		sh.ErrorCh = errorCh
		sh.MessageCh = messageCh
		sh.Unlock()
	} else {
		sh.Logger.Info().Msg("Reusing websocket connection")
	}

	sh.Logger.Trace().Msg("Reading from WS")

	// Read a message from WS which we should expect to be Hello
	msg, err := sh.readMessage()

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

	sh.Lock()
	sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond
	sh.MaxHeartbeatFailures = sh.HeartbeatInterval * time.Duration(sh.Manager.Configuration.Bot.MaxHeartbeatFailures)
	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)
	sh.Unlock()

	seq := atomic.LoadInt64(sh.seq)

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Int("maxfails", sh.Manager.Configuration.Bot.MaxHeartbeatFailures).
		Msg("Retrieved HELLO event from discord")

	// If we have no session ID or the sequence is 0, we can identify instead
	// of resuming.
	if sh.sessionID == "" || seq == 0 {
		err = sh.Identify()
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to identify")

			go sh.PublishWebhook("Gateway `IDENTIFY` failed", err.Error(), 14431557, false)

			return
		}
	} else {
		err = sh.Resume()
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to resume")

			go sh.PublishWebhook("Gateway `RESUME` failed", err.Error(), 14431557, false)

			return
		}

		// We will assume the bot is ready.
		if err := sh.SetStatus(structs.ShardReady); err != nil {
			sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
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

	sh.RLock()
	errorch := sh.ErrorCh
	messagech := sh.MessageCh
	sh.RUnlock()

	select {
	case err = <-errorch:
		sh.Logger.Error().Err(err).Msg("Encountered error whilst connecting")

		go sh.PublishWebhook("Encountered error during connection", err.Error(), 14431557, false)

		return xerrors.Errorf("encountered error whilst connecting: %w", err)
	case msg = <-messagech:
		sh.OnEvent(msg)
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

			now := time.Now().UTC()
			msg := structs.ReceivedPayload{
				TraceTime: now,
				Trace:     make(map[string]int),
			}

			err = json.Unmarshal(buf, &msg)
			if err != nil {
				sh.Logger.Error().Err(err).Msg("Failed to unmarshal message")

				continue
			}

			now = time.Now().UTC()
			msg.AddTrace("unmarshal", now)

			atomic.AddInt64(sh.events, 1)

			messageCh <- msg
		}
	}()

	return errorCh, messageCh, nil
}

// OnEvent processes an event.
func (sh *Shard) OnEvent(msg structs.ReceivedPayload) {
	var err error

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

	defer close(fin)

	switch msg.Op {
	case structs.GatewayOpHeartbeat:
		sh.Logger.Debug().Msg("Received heartbeat request")
		err = sh.SendEvent(structs.GatewayOpHeartbeat, atomic.LoadInt64(sh.seq))

		if err != nil {
			go sh.PublishWebhook("Failed to send heartbeat to gateway", err.Error(), 16760839, false)

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

		go sh.PublishWebhook("Received invalid session from gateway", "", 16760839, false)

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

		sh.Lock()
		sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond
		sh.MaxHeartbeatFailures = sh.HeartbeatInterval * time.Duration(sh.Manager.Configuration.Bot.MaxHeartbeatFailures)
		sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)
		sh.Unlock()

		sh.Logger.Debug().
			Dur("interval", sh.HeartbeatInterval).
			Int("maxfails", sh.Manager.Configuration.Bot.MaxHeartbeatFailures).
			Msg("Retrieved HELLO event from discord")

		return
	case structs.GatewayOpReconnect:
		sh.Logger.Info().Msg("Reconnecting in response to gateway")
		err = sh.Reconnect(reconnectCloseCode)

		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to reconnect")
		}

		return
	case structs.GatewayOpDispatch:
		go func() {
			var ticket int

			atomic.AddInt64(sh.Manager.Sandwich.PoolWaiting, 1)

			ticket = sh.Manager.Sandwich.Pool.Wait()
			defer sh.Manager.Sandwich.Pool.FreeTicket(ticket)

			msg.AddTrace("ticket", time.Now().UTC())
			msg.Trace["ticket_id"] = ticket

			atomic.AddInt64(sh.Manager.Sandwich.PoolWaiting, -1)

			err = sh.OnDispatch(msg)
			if err != nil && !xerrors.Is(err, NoHandler) {
				sh.Logger.Error().Err(err).Msg("Failed to handle event")
			}
		}()
	case structs.GatewayOpHeartbeatACK:
		sh.LastHeartbeatMu.Lock()
		sh.LastHeartbeatAck = time.Now().UTC()
		sh.Logger.Debug().
			Int64("RTT", sh.LastHeartbeatAck.Sub(sh.LastHeartbeatSent).Milliseconds()).
			Msg("Received heartbeat ACK")

		sh.LastHeartbeatMu.Unlock()

		return
	case structs.GatewayOpVoiceStateUpdate:
		// Todo: handle
	case structs.GatewayOpIdentify,
		structs.GatewayOpRequestGuildMembers,
		structs.GatewayOpResume,
		structs.GatewayOpStatusUpdate:
	default:
		sh.Logger.Warn().
			Int("op", int(msg.Op)).
			Str("type", msg.Type).
			Msg("Gateway sent unknown packet")

		return
	}

	atomic.StoreInt64(sh.seq, msg.Sequence)
}

// OnDispatch handles a dispatch event.
func (sh *Shard) OnDispatch(msg structs.ReceivedPayload) (err error) {
	start := time.Now().UTC()

	defer func() {
		now := time.Now().UTC()
		change := now.Sub(start)

		msg.AddTrace("publish", now)

		if change > time.Second {
			l := sh.Logger.Warn()

			if trcrslt, err := json.MarshalToString(msg.Trace); err == nil {
				l = l.Str("trace", trcrslt)
			}

			l.Msgf("%s took %d ms", msg.Type, change.Milliseconds())
		}

		if change > 15*time.Second {
			trcrslt := ""

			for tracer, tracetime := range msg.Trace {
				trcrslt += fmt.Sprintf("%s: **%d**ms\n", tracer, tracetime)
			}

			go sh.PublishWebhook(
				fmt.Sprintf("Packet `%s` took too long. Took `%dms`", msg.Type,
					change.Milliseconds()), trcrslt, 16760839, false)
		}
	}()

	if sh.Manager.StanClient == nil {
		return xerrors.Errorf("no stan client found")
	}

	// Ignore events that are in the event blacklist.
	sh.Manager.EventBlacklistMu.RLock()
	contains := gotils.StringSliceInclude(sh.Manager.EventBlacklist, msg.Type)
	sh.Manager.EventBlacklistMu.RUnlock()

	if contains {
		return
	}

	now := time.Now().UTC()

	msg.AddTrace("dispatch", now)

	results, ok, err := sh.Manager.Sandwich.StateDispatch(&StateCtx{
		Sg: sh.Manager.Sandwich,
		Mg: sh.Manager,
		Sh: sh,
	}, msg)
	if err != nil {
		return xerrors.Errorf("on dispatch failure for %s: %w", msg.Type, err)
	}

	if !ok {
		return
	}

	// Do not publish the event if it is in the produce blacklist,
	// regardless if it has been marked ok.
	sh.Manager.ProduceBlacklistMu.RLock()
	contains = gotils.StringSliceInclude(sh.Manager.ProduceBlacklist, msg.Type)
	sh.Manager.ProduceBlacklistMu.RUnlock()

	if contains {
		return
	}

	now = time.Now().UTC()
	msg.AddTrace("state", now)

	packet := sh.pp.Get().(*structs.SandwichPayload)
	defer sh.pp.Put(packet)

	packet.ReceivedPayload = msg
	packet.Trace = msg.Trace
	packet.Data = results.Data
	packet.Extra = results.Extra

	return sh.PublishEvent(packet)
}

// PublishEvent publishes a SandwichPayload.
func (sh *Shard) PublishEvent(packet *structs.SandwichPayload) (err error) {
	sh.Manager.ConfigurationMu.RLock()
	defer sh.Manager.ConfigurationMu.RUnlock()

	packet.Metadata = structs.SandwichMetadata{
		Version:    VERSION,
		Identifier: sh.Manager.Configuration.Identifier,
		Shard: [3]int{
			int(sh.ShardGroup.ID),
			sh.ShardID,
			sh.ShardGroup.ShardCount,
		},
	}

	payload, err := msgpack.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("failed to marshal payload: %w", err)
	}

	sh.Logger.Trace().Str("event", gotils.B2S(payload)).Msgf("Processed %s event", packet.Type)

	// Compression testing of large payloads. In the future this *may* be
	// added however in its current state it is uncertain. With using a 1mb
	// msgpack payload, compression can be brought down to 48kb using brotli
	// level 11 however will take around 1.5 seconds. However, it is likely
	// level 0 or 6 will be used which produce 95kb in 3ms and 54kb in 20ms
	// respectively. It is likely the actual data portion of the payload will
	// be compressed so the metadata and the rest of the data can be preserved
	// then pass in the metadata it is compressed instead of using magic bytes
	// or guessing by consumers.

	// Whilst compression can prove a benefit, having it enabled for all events
	// do not provide any benefit and only affect larger payloads which is
	// not common apart from GUILD_CREATE events.

	// Sample testing of a GUILD_CREATE event:

	// METHOD | Level        | Ms   | Resulting Payload Size
	// -------|--------------|------|-----------------------
	// NONE   |              |      | 1011967
	// BROTLI | 0  (speed)   | 3    | 95908   ( 9.5%)
	// BROTLI | 6  (default) | 20   | 54545   ( 5.4%)
	// BROTLI | 11 (best)    | 1245 | 47044   ( 4.6%)
	// GZIP   | 1  (speed)   | 3    | 115799  (11.5%)
	// GZIP   | -1 (default) | 8    | 82336   ( 8.1%)
	// GZIP   | 9  (best)    | 19   | 78253   ( 7.7%)

	// This may not be the most efficient way but it was useful for testing many
	// payloads. More cohesive benchmarking will take place if this is ever properly
	// implemented and may be a 1.0 feature however it is unlikely to be necessary..

	// if len(payload) > 100000 {
	// 	println("NONE", len(payload))

	// 	var b bytes.Buffer

	// 	br := brotli.NewWriterLevel(&b, brotli.BestSpeed)
	// 	a := time.Now()
	// 	br.Write(payload)
	// 	br.Close()
	// 	println("BROTLI", brotli.BestSpeed, time.Now().Sub(a).Milliseconds(), b.Len())
	// 	b.Reset()

	// 	br = brotli.NewWriterLevel(&b, brotli.DefaultCompression)
	// 	a = time.Now()
	// 	br.Write(payload)
	// 	br.Close()
	// 	println("BROTLI", brotli.DefaultCompression, time.Now().Sub(a).Milliseconds(), b.Len())
	// 	b.Reset()

	// 	br = brotli.NewWriterLevel(&b, brotli.BestCompression)
	// 	a = time.Now()
	// 	br.Write(payload)
	// 	br.Close()
	// 	println("BROTLI", brotli.BestCompression, time.Now().Sub(a).Milliseconds(), b.Len())
	// 	b.Reset()

	// 	gz, _ := gzip.NewWriterLevel(&b, gzip.BestSpeed)
	// 	a = time.Now()
	// 	gz.Write(payload)
	// 	gz.Close()
	// 	println("GZIP  ", gzip.BestSpeed, time.Now().Sub(a).Milliseconds(), b.Len())
	// 	b.Reset()

	// 	gz, _ = gzip.NewWriterLevel(&b, gzip.DefaultCompression)
	// 	a = time.Now()
	// 	gz.Write(payload)
	// 	gz.Close()
	// 	println("GZIP  ", gzip.DefaultCompression, time.Now().Sub(a).Milliseconds(), b.Len())
	// 	b.Reset()

	// 	gz, _ = gzip.NewWriterLevel(&b, gzip.BestCompression)
	// 	a = time.Now()
	// 	gz.Write(payload)
	// 	gz.Close()
	// 	println("GZIP  ", gzip.BestCompression, time.Now().Sub(a).Milliseconds(), b.Len())
	// 	b.Reset()
	// }

	err = sh.Manager.StanClient.Publish(
		sh.Manager.Configuration.Messaging.ChannelName,
		payload,
	)
	if err != nil {
		return xerrors.Errorf("publishEvent publish: %w", err)
	}

	return nil
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
				case structs.CloseNotAuthenticated, // Not authenticated
					structs.CloseInvalidShard,      // Invalid shard
					structs.CloseShardingRequired,  // Sharding required
					structs.CloseInvalidAPIVersion, // Invalid API version
					structs.CloseInvalidIntents,    // Invalid Intent(s)
					structs.CloseDisallowedIntents: // Disallowed intent(s)
					sh.Logger.Warn().Msgf(
						"Closing ShardGroup as cannot continue without valid token. Received code %d",
						closeError.Code,
					)

					go sh.PublishWebhook("ShardGroup is closing due to invalid token being passed", "", 16760839, false)

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

// Heartbeat maintains a heartbeat with discord
// TODO: Make a shardgroup specific heartbeat function to heartbeat on behalf of all running shards.
func (sh *Shard) Heartbeat() {
	sh.HeartbeatActive.Set()
	defer sh.HeartbeatActive.UnSet()

	for {
		sh.RLock()
		heartbeater := sh.Heartbeater
		sh.RUnlock()

		select {
		case <-sh.ctx.Done():
			return
		case <-heartbeater.C:
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

					go sh.PublishWebhook("Failed to heartbeat. Reconnecting", "", 16760839, false)
				} else {
					sh.Manager.Sandwich.ConfigurationMu.RLock()
					sh.Logger.Warn().Err(err).
						Msgf(
							"Gateway failed to ACK and has passed MaxHeartbeatFailures of %d. Reconnecting",
							sh.Manager.Configuration.Bot.MaxHeartbeatFailures)

					go sh.PublishWebhook(fmt.Sprintf(
						"Gateway failed to ACK and has passed MaxHeartbeatFailures of %d. Reconnecting",
						sh.Manager.Configuration.Bot.MaxHeartbeatFailures), "", 1548214, false)

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
func (sh *Shard) readMessage() (msg structs.ReceivedPayload, err error) {
	// Prioritize errors
	select {
	case err = <-sh.ErrorCh:
		return msg, err
	default:
	}

	sh.RLock()
	errorch := sh.ErrorCh
	messagech := sh.MessageCh
	sh.RUnlock()

	select {
	case err = <-errorch:
		return msg, err
	case msg = <-messagech:
		msg.AddTrace("read", time.Now().UTC())

		return msg, nil
	}
}

// CloseWS closes the websocket. This will always return 0 as the error is suppressed.
func (sh *Shard) CloseWS(statusCode websocket.StatusCode) (err error) {
	if sh.wsConn != nil {
		sh.Logger.Debug().Str("code", statusCode.String()).Msg("Closing websocket connection")

		err = sh.wsConn.Close(statusCode, "")
		if err != nil && !xerrors.Is(err, context.Canceled) {
			sh.Logger.Warn().Err(err).Msg("Failed to close websocket connection")
		}

		sh.wsConn = nil
	}

	return nil
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

	hash, err := QuickHash(sh.Manager.Configuration.Token)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to generate token hash")

		return err
	}

	sh.Manager.GatewayMu.RLock()
	err = sh.Manager.Sandwich.Buckets.WaitForBucket(
		fmt.Sprintf("gw:%s:%d", hash, sh.ShardID%sh.Manager.Gateway.SessionStartLimit.MaxConcurrency),
	)
	sh.Manager.GatewayMu.RUnlock()

	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to wait for bucket")
	}

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
			if err != nil {
				go sh.PublishWebhook("Failed to reconnect to gateway", err.Error(), 14431557, false)
			}

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

	sh.Logger.Debug().
		Str("manager", sh.Manager.Configuration.Identifier).
		Int32("shardgroup", sh.ShardGroup.ID).
		Int("shard", sh.ShardID).
		Msgf("Status changed to %s (%d)", status.String(), status)

	switch status {
	case structs.ShardReady, structs.ShardReconnecting:
		sh.Manager.ConfigurationMu.RLock()
		isMinimal := sh.Manager.Sandwich.Configuration.Logging.MinimalWebhooks
		sh.Manager.ConfigurationMu.RUnlock()

		go sh.PublishWebhook(fmt.Sprintf("Shard is now **%s**", status.String()), "", status.Colour(), isMinimal)
	case structs.ShardIdle,
		structs.ShardWaiting,
		structs.ShardConnecting,
		structs.ShardConnected,
		structs.ShardClosed:
	}

	packet := sh.pp.Get().(*structs.SandwichPayload)
	defer sh.pp.Put(packet)

	packet.ReceivedPayload = structs.ReceivedPayload{
		Type: "SHARD_STATUS",
	}

	packet.Data = structs.MessagingStatusUpdate{
		ShardID: sh.ShardID,
		Status:  int32(status),
	}

	return sh.PublishEvent(packet)
}

// Latency returns the heartbeat latency in milliseconds.
func (sh *Shard) Latency() (latency int64) {
	sh.LastHeartbeatMu.RLock()
	defer sh.LastHeartbeatMu.RUnlock()

	return sh.LastHeartbeatAck.Sub(sh.LastHeartbeatSent).Round(time.Millisecond).Milliseconds()
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

// PublishWebhook is the same as sg.PublishWebhook but has extra sugar for
// displaying information about the shard.
func (sh *Shard) PublishWebhook(title string, description string, colour int, raw bool) {
	var message structs.WebhookMessage

	if raw {
		message = structs.WebhookMessage{
			Content: fmt.Sprintf("[**%s - %d/%d**] %s %s",
				sh.Manager.Configuration.DisplayName,
				sh.ShardGroup.ID, sh.ShardID, title, description),
		}

		sh.RLock()
		if sh.User != nil && message.AvatarURL == "" && message.Username == "" {
			message.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png",
				sh.User.ID.String(), sh.User.Avatar)
			message.Username = sh.User.Username
		}
		sh.RUnlock()
	} else {
		message = structs.WebhookMessage{
			Embeds: []structs.Embed{
				{
					Title:       title,
					Description: description,
					Color:       colour,
					Timestamp:   WebhookTime(time.Now().UTC()),
					Footer: &structs.EmbedFooter{
						Text: fmt.Sprintf("Manager %s | ShardGroup %d | Shard %d",
							sh.Manager.Configuration.DisplayName,
							sh.ShardGroup.ID, sh.ShardID),
					},
				},
			},
		}

		sh.RLock()
		if sh.User != nil && message.AvatarURL == "" && message.Username == "" {
			message.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png",
				sh.User.ID.String(), sh.User.Avatar)
			message.Username = sh.User.Username
		}
		sh.RUnlock()
	}

	sh.Manager.Sandwich.PublishWebhook(context.Background(), message)
}
