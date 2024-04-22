package internal

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"nhooyr.io/websocket"
)

func init() {
	MQClients = append(MQClients, "websocket")
}

// Given a guild ID, return its shard ID
func getShardIDFromGuildID(guildID string, shardCount int) (uint64, error) {
	gidNum, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint64(gidNum>>22) % uint64(shardCount), nil
}

// chatServer enables broadcasting to a set of subscribers.
type chatServer struct {
	// the expected token
	expectedToken string

	// sandwich state
	manager *Manager

	// address
	address string

	// subscriberMessageBuffer controls the max number
	// of messages that can be queued for a subscriber
	// before it is kicked.
	//
	// Defaults to 16.
	subscriberMessageBuffer int

	// serveMux routes the various endpoints to the appropriate handler.
	serveMux http.ServeMux

	subscribersMu sync.RWMutex
	subscribers   map[[2]int32][]*subscriber
}

// newChatServer constructs a chatServer with the defaults.
func newChatServer() *chatServer {
	cs := &chatServer{
		subscriberMessageBuffer: 100000, // Make it large enough for handling resumes
		subscribers:             make(map[[2]int32][]*subscriber),
	}
	cs.serveMux.HandleFunc("/", cs.subscribeHandler)
	cs.serveMux.HandleFunc("/publish", cs.publishHandler)

	return cs
}

type message struct {
	// What raw bytes to send, this bypasses seq additions etc.
	rawBytes []byte
	// What message to send, note that sequence will be automatically set
	message *structs.SandwichPayload
	// close code, if set will close the connection
	closeCode websocket.StatusCode
	// close string, will be sent on close
	closeString string
}

// subscriber represents a subscriber.
// Messages are sent on the msgs channel and if the client
// cannot keep up with the messages, closeSlow is called.
type subscriber struct {
	c          *websocket.Conn
	cancelFunc context.CancelFunc
	sessionId  string
	shard      [2]int32
	up         bool
	resumed    bool
	seq        int32
	reader     chan *message
	writer     chan *message
}

// invalidSession closes the connection with the given reason.
func (cs *chatServer) invalidSession(s *subscriber, reason string, resumable bool) {
	cs.manager.Sandwich.Logger.Error().Msgf("[WS] Invalid session: %s, is resumable: %v", reason, resumable)

	if resumable {
		s.writer <- &message{
			rawBytes:    []byte(`{"op":9,"d":true}`),
			closeCode:   websocket.StatusCode(4000),
			closeString: "Invalid Session",
		}
	} else {
		s.writer <- &message{
			rawBytes:    []byte(`{"op":9,"d":false}`),
			closeCode:   websocket.StatusCode(4000),
			closeString: "Invalid Session",
		}
	}
}

func (cs *chatServer) getShard(shard [2]int32) *Shard {
	var shardRes *Shard
	cs.manager.ShardGroups.Range(func(k int32, sg *ShardGroup) bool {
		sg.shardsMu.RLock()
		for _, sh := range sg.Shards {
			if sh.ShardID == shard[0] {
				shardRes = sh
				return true
			}
		}
		sg.shardsMu.RUnlock()
		return false
	})

	return shardRes
}

func (cs *chatServer) dispatchInitial(ctx context.Context, s *subscriber) error {
	cs.manager.Sandwich.Logger.Info().Msgf("[WS] Shard %d/%d (now dispatching events) %v", s.shard[0], s.shard[1], s.shard)

	shard := cs.getShard(s.shard)
	guilds := make([]*discord.Guild, 0, len(cs.manager.Sandwich.State.Guilds))

	if shard != nil {
		shard.WaitForReady()
	} else {
		cs.manager.Sandwich.Logger.Info().Msgf("[WS] Shard %d is nil", s.shard[0])

	}

	// First send READY event with our initial state
	readyPayload := map[string]any{
		"v":          10,
		"user":       cs.manager.User,
		"session_id": shard.SessionID,
		"shard":      []int32{s.shard[0], s.shard[1]},
		"application": map[string]any{
			"id":    cs.manager.User.ID,
			"flags": int32(cs.manager.User.Flags),
		},
		"resume_gateway_url": func() string {
			if !strings.HasPrefix(cs.address, "ws") {
				return "ws://" + cs.address
			} else {
				return cs.address
			}
		}(),
		"guilds": func() []*discord.UnavailableGuild {
			v := []*discord.UnavailableGuild{}
			cs.manager.Sandwich.State.guildsMu.RLock()
			defer cs.manager.Sandwich.State.guildsMu.RUnlock()

			for _, guild := range cs.manager.Sandwich.State.Guilds {
				shardInt, err := getShardIDFromGuildID(guild.ID.String(), int(s.shard[1]))
				if err != nil {
					cs.invalidSession(s, "[Sandwich] Failed to get shard ID from guild ID: "+err.Error(), true)
					s.cancelFunc()
					return v
				}

				if int32(shardInt) != s.shard[0] {
					continue
				}

				guilds = append(guilds, guild)
				v = append(v, &discord.UnavailableGuild{
					ID:          guild.ID,
					Unavailable: true,
				})
			}

			return v
		}(),
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	serializedReadyPayload, err := jsoniter.Marshal(readyPayload)

	if err != nil {
		cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to marshal ready payload: %s", err.Error())
		return err
	}

	cs.manager.Sandwich.Logger.Info().Msgf("[WS] Dispatching ready to shard %d", s.shard[0])

	s.writer <- &message{
		message: &structs.SandwichPayload{
			Op:   discord.GatewayOpDispatch,
			Data: serializedReadyPayload,
			Type: "READY",
		},
	}

	for _, guild := range guilds {
		if guild.AFKChannelID == nil {
			guild.AFKChannelID = &guild.ID
		}

		// Send initial guild_create's
		if len(guild.Roles) == 0 {
			// Lock RolesMu
			cs.manager.Sandwich.State.guildRolesMu.RLock()
			roles := cs.manager.Sandwich.State.GuildRoles[guild.ID]
			cs.manager.Sandwich.State.guildRolesMu.RUnlock()

			guild.Roles = make([]*discord.Role, 0, len(roles.Roles))
			for id, role := range roles.Roles {
				role.ID = discord.Snowflake(id)
				guild.Roles = append(guild.Roles, role)
			}
		}

		serializedGuild, err := jsoniter.Marshal(guild)

		if err != nil {
			cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to marshal guild: %s [shard %d]", err.Error(), s.shard[0])
			continue
		}

		s.writer <- &message{
			message: &structs.SandwichPayload{
				Op:   discord.GatewayOpDispatch,
				Data: serializedGuild,
				Type: "GUILD_CREATE",
			},
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

func (cs *chatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cs.serveMux.ServeHTTP(w, r)
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (cs *chatServer) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	err := cs.subscribe(r.Context(), w, r)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		return
	}
}

// publishHandler reads the request body with a limit of 8192 bytes and then publishes
// the received message.
func (cs *chatServer) publishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Get shard from query params
	var shard [2]int32

	shardStr := r.URL.Query().Get("shard")

	if shardStr != "" {
		_, err := fmt.Sscanf(shardStr, "%d-%d", &shard[0], &shard[1])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	body := http.MaxBytesReader(w, r.Body, 8192)
	msg, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	var payload structs.SandwichPayload

	err = jsoniter.Unmarshal(msg, &payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	cs.publish(shard, &payload)

	w.WriteHeader(http.StatusAccepted)
}

// identifyClient tries to identify a incoming connection
func (cs *chatServer) identifyClient(ctx context.Context, s *subscriber) (oldSess *subscriber, err error) {
	// Before adding the subscriber for external access, send the initial hello payload and wait for identify
	// If the client does not identify within 5 seconds, close the connection
	s.writer <- &message{
		rawBytes: []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`),
	}

	// Keep reading messages till we reach an identify
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return nil, errors.New("timed out waiting for identify")
		case msg := <-s.reader:
			if msg.message == nil {
				continue
			}

			packet := msg.message

			// Read an identify packet
			//
			// Note that resume is not supported at this time
			if packet.Op == discord.GatewayOpIdentify {
				var identify struct {
					Token string   `json:"token"`
					Shard [2]int32 `json:"shard"`
				}

				err := jsoniter.Unmarshal(packet.Data, &identify)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal identify packet: %w", err)
				}

				identify.Token = strings.Replace(identify.Token, "Bot ", "", 1)

				if identify.Token == cs.expectedToken {
					s.sessionId = randomHex(12)

					// dpy workaround
					if identify.Shard[1] == 0 {
						identify.Shard[1] = cs.manager.noShards
					}

					s.shard = identify.Shard
					s.up = true

					cs.manager.Sandwich.Logger.Info().Msgf("[WS] Shard %d is now identified with created session id %s [%s]", s.shard[0], s.sessionId, fmt.Sprint(s.shard))
					return nil, nil
				} else {
					return nil, errors.New("invalid token")
				}
			} else if packet.Op == discord.GatewayOpResume {
				var resume struct {
					Token     string `json:"token"`
					SessionID string `json:"session_id"`
					Seq       int32  `json:"seq"`
				}

				err := jsoniter.Unmarshal(packet.Data, &resume)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal resume packet: %w", err)
				}

				resume.Token = strings.Replace(resume.Token, "Bot ", "", 1)

				if resume.Token == cs.expectedToken {
					// Find session with same session id
					cs.subscribersMu.RLock()
					for _, shardSubs := range cs.subscribers {
						for _, oldSess := range shardSubs {
							if s.sessionId == resume.SessionID {
								cs.manager.Sandwich.Logger.Info().Msgf("[WS] Shard %d is now identified with resumed session id %s [%s]", s.shard[0], s.sessionId, fmt.Sprint(s.shard))
								s.seq = resume.Seq
								s.shard = oldSess.shard
								s.resumed = true
								s.up = true
								cs.subscribersMu.RUnlock()
								return oldSess, nil
							}
						}
					}
					cs.subscribersMu.RUnlock()

					if !s.up {
						return nil, errors.New("invalid session id")
					}
				} else {
					return nil, errors.New("invalid token")
				}
			}
		}
	}
}

// reader reads messages from subscribe and sends them to the reader
// Note that there must be only one reader reading from the goroutine
func (cs *chatServer) readMessages(ctx context.Context, s *subscriber) {
	for {
		typ, ior, err := s.c.Read(ctx)

		if err != nil {
			s.cancelFunc()
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		switch typ {
		case websocket.MessageText:
			var payload *structs.SandwichPayload

			err := jsoniter.Unmarshal(ior, &payload)

			if err != nil {
				cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to unmarshal packet: %s", err.Error())
				cs.invalidSession(s, "failed to unmarshal packet: "+err.Error(), true)
				return
			}

			cs.manager.Sandwich.Logger.Debug().Msgf("[WS] Shard %d received packet: %v", s.shard[0], payload)

			if payload.Op == discord.GatewayOpHeartbeat {
				s.writer <- &message{
					rawBytes: []byte(`{"op":11}`),
				}
			} else {
				s.reader <- &message{
					message: payload,
				}
			}
		case websocket.MessageBinary:
			// ZLIB compressed message sigh
			newReader, err := zlib.NewReader(bytes.NewReader(ior))

			if err != nil {
				cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to decompress message: %s", err.Error())
				cs.invalidSession(s, "failed to decompress message: "+err.Error(), true)
				return
			}

			var payload *structs.SandwichPayload

			err = jsoniter.NewDecoder(newReader).Decode(&payload)

			if err != nil {
				cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to unmarshal packet: %s", err.Error())
				cs.invalidSession(s, "failed to unmarshal packet: "+err.Error(), true)
				return
			}

			if payload.Op == discord.GatewayOpHeartbeat {
				s.writer <- &message{
					rawBytes: []byte(`{"op":11}`),
				}
			} else {
				s.reader <- &message{
					message: payload,
				}
			}
		}
	}
}

// handleReadMessages handles messages from reader
func (cs *chatServer) handleReadMessages(ctx context.Context, s *subscriber) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.reader:
			// Send to discord directly
			cs.manager.Sandwich.Logger.Debug().Msgf("[WS] Shard %d got/found packet: %v", s.shard[0], msg)

			sh := cs.getShard(s.shard)

			if sh == nil {
				cs.manager.Sandwich.Logger.Error().Msgf("[WS] Shard %d is nil", s.shard[0])
				continue
			}

			err := sh.SendEvent(ctx, msg.message.Op, msg.message.Data)

			if err != nil {
				cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to send event: %s", err.Error())
			}
		}
	}
}

// writeMessages reads messages from the writer and sends them to the WebSocket
func (cs *chatServer) writeMessages(ctx context.Context, s *subscriber) {
	for {
		select {
		// Case 1: Context is cancelled
		case <-ctx.Done():
			return
		// Case 2: Message is received
		case msg := <-s.writer:
			if len(msg.rawBytes) > 0 {
				err := s.c.Write(ctx, websocket.MessageText, msg.rawBytes)

				if err != nil {
					cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to write message [rawBytes]: %s", err.Error())
					s.c.Close(websocket.StatusInternalError, "Failed to write message [rawBytes]")
					s.cancelFunc()
					return
				}
			}

			if msg.message != nil {
				if msg.message.Op == discord.GatewayOpDispatch {
					msg.message.Sequence = s.seq
					s.seq++
				} else {
					msg.message.Sequence = 0
				}

				serializedMessage, err := jsoniter.Marshal(msg.message)

				if err != nil {
					cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to marshal message: %s", err.Error())
					continue
				}

				err = s.c.Write(ctx, websocket.MessageText, serializedMessage)

				if err != nil {
					cs.manager.Sandwich.Logger.Error().Msgf("[WS] Failed to write message [serialized]: %s", err.Error())
					s.c.Close(websocket.StatusInternalError, "Failed to write message [serialized]")
					s.cancelFunc()
					return
				}
			}

			if msg.closeCode != 0 {
				s.up = false
				s.c.Close(msg.closeCode, msg.closeString)
				s.cancelFunc()
				return
			}
		}
	}

}

// subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a subscriber with a buffered msgs chan to give some room to slower
// connections and then registers the subscriber. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (cs *chatServer) subscribe(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var c *websocket.Conn
	s := &subscriber{
		reader: make(chan *message, cs.subscriberMessageBuffer),
		writer: make(chan *message, cs.subscriberMessageBuffer),
	}

	var err error
	c, err = websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}

	s.c = c

	if cs.manager.Sandwich == nil {
		c.Close(websocket.StatusInternalError, "sandwich is nil")
		return errors.New("sandwich is nil")
	}

	newCtx, cancelFunc := context.WithCancel(ctx)
	s.cancelFunc = cancelFunc
	defer s.cancelFunc()

	defer c.Close(websocket.StatusCode(4000), `{"op":9,"d":true}`)

	// Start the reader+writer bit
	go cs.writeMessages(newCtx, s)
	time.Sleep(5 * time.Millisecond)
	go cs.readMessages(newCtx, s)
	time.Sleep(5 * time.Millisecond)

	cs.manager.Sandwich.Logger.Info().Msgf("[WS] Shard %d is now launched (reader+writer UP)", s.shard[0])

	// Now identifyClient
	oldSess, err := cs.identifyClient(newCtx, s)

	if err != nil {
		cs.invalidSession(s, err.Error(), false)
		return err
	}

	cs.addSubscriber(s, s.shard)
	defer cs.deleteSubscriber(s)

	// SAFETY: at this point, the identifyClient loop has ended, so there should be no more subscribers to reader
	//
	// Calling handleReadMessages is hence safe
	go cs.handleReadMessages(newCtx, s)

	if oldSess != nil {
		cs.invalidSession(oldSess, "New session identified", true)
		for msg := range oldSess.reader {
			select {
			case <-newCtx.Done():
				return newCtx.Err()
			default:
			}

			if msg.message == nil {
				continue
			}

			if msg.message.Op != discord.GatewayOpDispatch {
				continue
			}

			s.writer <- msg
		}
	}

	cs.manager.Sandwich.Logger.Info().Msgf("[WS] Shard %d is now connected (oldSess fanout done)", s.shard[0])

	if !s.resumed {
		cs.dispatchInitial(ctx, s)
	} else {
		// Send a RESUMED event
		s.writer <- &message{
			message: &structs.SandwichPayload{
				Op:   discord.GatewayOpDispatch,
				Data: jsoniter.RawMessage([]byte(`{}`)),
				Type: "RESUMED",
			},
		}
	}

	// Wait for the context to be cancelled
	// readMessages and writeMessages will handle the rest
	for range newCtx.Done() {
		return newCtx.Err()
	}

	return nil
}

// publish publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (cs *chatServer) publish(shard [2]int32, msg *structs.SandwichPayload) {
	cs.manager.Sandwich.Logger.Trace().Msgf("[WS] Shard %d is now publishing message", shard[0])

	cs.subscribersMu.RLock()
	defer cs.subscribersMu.RUnlock()

	shardSubs, ok := cs.subscribers[shard]

	if !ok {
		return
	}

	for _, s := range shardSubs {
		if !s.up {
			continue
		}

		cs.manager.Sandwich.Logger.Trace().Msgf("[WS] Shard %d is now publishing message to %d subscribers", shard[0], len(shardSubs))

		s.writer <- &message{
			message: msg,
		}
	}
}

// addSubscriber registers a subscriber.
func (cs *chatServer) addSubscriber(s *subscriber, shard [2]int32) {
	cs.subscribersMu.Lock()

	if subs, ok := cs.subscribers[shard]; ok {
		cs.subscribers[shard] = append(subs, s)
	} else {
		cs.subscribers[shard] = []*subscriber{s}
	}

	cs.subscribersMu.Unlock()
}

// deleteSubscriber deletes the given subscriber.
func (cs *chatServer) deleteSubscriber(s *subscriber) {
	cs.subscribersMu.Lock()

	if sub, ok := cs.subscribers[s.shard]; ok {
		for i, is := range sub {
			if is.sessionId == s.sessionId {
				cs.subscribers[s.shard] = append(sub[:i], sub[i+1:]...)
				break
			}
		}
	}

	cs.subscribersMu.Unlock()
}

type WebsocketClient struct {
	cs *chatServer
}

func (mg *WebsocketClient) String() string {
	return "websocket"
}

func (mq *WebsocketClient) Channel() string {
	return "websocket"
}

func (mq *WebsocketClient) Cluster() string {
	return "websocket"
}

func (mq *WebsocketClient) Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error {
	var ok bool

	var address string
	var expectedToken string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return errors.New("websocketMQ connect: string type assertion failed for Address")
	}

	if expectedToken, ok = GetEntry(args, "ExpectedToken").(string); !ok {
		return errors.New("websocketMQ connect: string type assertion failed for ExpectedToken")
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		return errors.New("websocketMQ listen: " + err.Error())
	}

	mq.cs = newChatServer()
	mq.cs.expectedToken = expectedToken
	mq.cs.manager = manager
	mq.cs.address = address
	s := &http.Server{Handler: mq.cs}

	go func() {
		s.Serve(l)
	}()

	return nil
}

func (mq *WebsocketClient) Publish(ctx context.Context, packet *structs.SandwichPayload, channelName string) error {
	go mq.cs.publish(
		[2]int32{packet.Metadata.Shard[1], packet.Metadata.Shard[2]},
		packet,
	)

	return nil
}

func (mq *WebsocketClient) IsClosed() bool {
	return mq.cs == nil
}

func (mq *WebsocketClient) CloseShard(shardID int32) {
	// Send RESUME for single shard
	for _, shardSubs := range mq.cs.subscribers {
		for _, s := range shardSubs {
			if s.shard[0] == shardID {
				mq.cs.invalidSession(s, "Shard closed", true)
				mq.cs.deleteSubscriber(s)
			}
		}
	}
}

func (mq *WebsocketClient) Close() {
	// Send RESUME to all shards
	for _, shardSubs := range mq.cs.subscribers {
		for _, s := range shardSubs {
			mq.cs.invalidSession(s, "Connection closed", true)
			s.cancelFunc()
			mq.cs.deleteSubscriber(s)
		}
	}

	mq.cs = nil
}
