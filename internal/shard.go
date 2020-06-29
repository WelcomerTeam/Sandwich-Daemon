package gateway

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/TheRockettek/czlib"
	"github.com/rs/zerolog"
	"nhooyr.io/websocket"
)

// var json = jsoniter.ConfigCompatibleWithStandardLibrary

const identifyRatelimit = 5 * time.Second

// Shard represents the shard object
type Shard struct {
	Logger zerolog.Logger

	ShardID    int
	ShardGroup *ShardGroup
	Manager    *Manager

	ctx    context.Context
	cancel func()

	LastHeartbeatAck  time.Time
	LastHeartbeatSent time.Time

	Heartbeater          *time.Ticker
	HeartbeatInterval    time.Duration
	MaxHeartbeatFailures time.Duration

	wsConn  *websocket.Conn
	wsMutex sync.Mutex

	msg structs.ReceivedPayload
	buf []byte

	seq       *int64
	sessionID string

	// Channel that dictates if the shard has been made ready
	ready chan void
	// Channel to pipe errors
	errs chan error
}

// NewShard creates a new shard object
func (sg *ShardGroup) NewShard(shardID int) *Shard {
	logger := sg.Logger.With().Int("shard", shardID).Logger()
	sh := &Shard{
		ShardID:    shardID,
		ShardGroup: sg,
		Manager:    sg.Manager,
		Logger:     logger,

		ctx: context.Background(),

		LastHeartbeatAck:  time.Now().UTC(),
		LastHeartbeatSent: time.Now().UTC(),

		msg: structs.ReceivedPayload{},
		buf: make([]byte, 0),

		seq:       new(int64),
		sessionID: "",

		ready: make(chan void),
		errs:  make(chan error),
	}
	sh.ctx, sh.cancel = context.WithCancel(context.Background())
	return sh
}

// Open starts up the shard connection
func (sh *Shard) Open() (errs chan error) {
	sh.Logger.Info().Msg("Starting shard")
	sh.ctx, sh.cancel = context.WithCancel(context.Background())

	gatewayURL := sh.Manager.Gateway.URL
	sh.Logger.Debug().Str("gurl", gatewayURL).Msg("Connecting to gateway")

	// Allow for a buffer of 1 so we can add to the channel without blocking
	errs = make(chan error, 1)
	sh.errs = errs

	err := sh.Manager.Sandwich.Buckets.CreateWaitForBucket(fmt.Sprintf("gw:%s:%d", sh.Manager.Configuration.Token, sh.ShardID%sh.Manager.Gateway.SessionStartLimit.MaxConcurrency), 1, identifyRatelimit)
	if err != nil {
		errs <- err
		return
	}

	// TODO: Add Concurrent Client Support
	// This will limit the ammount of shards that can be connecting simultaneously
	// May be abandoned as this boy is fast af :pepega:
	// Could help with a shit ton running at once whilst scaling

	conn, _, err := websocket.Dial(sh.ctx, gatewayURL, nil)
	if err != nil {
		errs <- err
		return
	}
	conn.SetReadLimit(512 << 20)
	sh.wsConn = conn

	err = sh.readMessage(sh.wsConn)
	if err != nil {
		errs <- err
		return
	}

	hello := structs.Hello{}
	err = sh.decodeContent(&hello)

	sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond
	sh.MaxHeartbeatFailures = sh.HeartbeatInterval * (time.Duration(sh.Manager.Configuration.Bot.MaxHeartbeatFailures) * time.Millisecond)

	sh.Logger.Info().Dur("interval", sh.HeartbeatInterval).Int("maxfails", sh.Manager.Configuration.Bot.MaxHeartbeatFailures).Msg("Retrieved HELLO event from discord")
	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

	seq := atomic.LoadInt64(sh.seq)
	if sh.sessionID == "" && seq == 0 {
		sh.Logger.Debug().Msg("Sending identify")
		err = sh.WriteJSON(structs.SentPayload{
			Op: 2,
			Data: structs.Identify{
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
			},
		})
	} else {
		sh.Logger.Debug().Msg("Sending resume")
		err = sh.WriteJSON(structs.SentPayload{
			Op: 6,
			Data: structs.Resume{
				Token:     sh.Manager.Configuration.Token,
				SessionID: sh.sessionID,
				Seq:       seq,
			},
		})
	}
	if err != nil {
		errs <- err

		return
	}

	go sh.Heartbeat()
	go sh.Listen()

	return errs
}

// Listen to gateway and process accordingly
func (sh *Shard) Listen() {
	wsConn := sh.wsConn
	evnts := int64(0)
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-sh.ctx.Done():
			return
		case <-t.C:
			atomic.AddInt64(sh.ShardGroup.Events, evnts)
			evnts = 0
		default:
		}

		err := sh.readMessage(wsConn)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Error reading from gateway")
			if wsConn == sh.wsConn {
				// We have likely closed so we should attempt to reconnect
				sh.Logger.Warn().Msg("We have encountered an error whilst in the same connection, reconnecting...")
				sh.Reconnect()
			}
			wsConn = sh.wsConn
		}

		evnts++
		// TODO: Actually do something here :)
		// println(sh.ShardID, sh.msg.Op, sh.msg.Type, len(sh.msg.Data))
	}
}

// Heartbeat maintains a heartbeat with discord
// TODO: Make a shardgroup specific heartbeat function to heartbeat on behalf of all running shards
func (sh *Shard) Heartbeat() {
	for {
		select {
		case <-sh.ctx.Done():
			return
		case <-sh.Heartbeater.C:
			sh.Logger.Debug().Msg("Heartbeating")
			seq := atomic.LoadInt64(sh.seq)
			err := sh.WriteJSON(structs.SentPayload{
				Op:   int(structs.GatewayOpHeartbeat),
				Data: seq,
			})
			if err != nil || time.Now().UTC().Sub(sh.LastHeartbeatAck) > sh.MaxHeartbeatFailures {
				sh.Logger.Error().Err(err).Msg("Failed to heartbeat")
				sh.CloseWS(1000)
				sh.Close()
			}
		}
	}
}

// readMessage fills the shard msg buffer from a websocket message
func (sh *Shard) readMessage(wsConn *websocket.Conn) (err error) {
	var mt websocket.MessageType

	mt, sh.buf, err = wsConn.Read(sh.ctx)
	if err != nil {
		sh.Logger.Error().Msg("Failed to read websocket")
		return
	}

	if mt == websocket.MessageBinary {
		sh.buf, err = czlib.Decompress(sh.buf)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to decompress buffer")
			return
		}
	}

	err = json.Unmarshal(sh.buf, &sh.msg)
	return
}

// decodeContent converts the stored msg into the passed interface
func (sh *Shard) decodeContent(out interface{}) (err error) {
	err = json.Unmarshal(sh.msg.Data, &out)
	return
}

// CloseWS closes the websocket
func (sh *Shard) CloseWS(statusCode websocket.StatusCode) (err error) {
	sh.Logger.Info().Str("code", statusCode.String()).Msg("Closing websocket connection")

	if sh.wsConn != nil {
		err = sh.wsConn.Close(statusCode, "")
		sh.wsConn = nil
	}
	return
}

// WriteJSON writes json data to the websocket
func (sh *Shard) WriteJSON(i interface{}) (err error) {
	res, err := json.Marshal(i)
	if err != nil {
		return
	}
	err = sh.wsConn.Write(sh.ctx, websocket.MessageText, res)
	return
}

// Reconnect will reconnect the session when closing
func (sh *Shard) Reconnect() {
	sh.CloseWS(websocket.StatusNormalClosure)
	sh.Close()
	sh.errs <- ErrReconnect
}

// Close closes the shard connection
func (sh *Shard) Close() {
	// Ensure that if we close during shardgroup connecting, it will not
	// feedback loop.
	sh.cancel()
}

// WaitUntilReady waits until the shard is ready
func (sh *Shard) WaitUntilReady() {
	select {
	case <-sh.ready:
	case <-sh.ctx.Done():
	}
	return
}

// Recoverable returns a boolean if the shard can be recovered
func (sh *Shard) Recoverable(err error) bool {
	return err == ErrReconnect
	// return !contains(websocket.CloseStatus(err), structs.CloseShardingRequired, structs.CloseAuthenticationFailed, structs.CloseInvalidShard, websocket.StatusNormalClosure) &&
	// 	!contains(err, ErrSessionLimitExhausted, ErrInvalidToken)
}
