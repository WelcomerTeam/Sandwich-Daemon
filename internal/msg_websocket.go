package internal

import (
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

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	"nhooyr.io/websocket"
)

var (
	heartbeatAck               = []byte(`{"op":11}`)
	helloPayload               = []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`)
	resumableInvalidSession    = []byte(`{"op":9,"d":true}`)
	nonresumableInvalidSession = []byte(`{"op":9,"d":false}`)
	invalidSessionOpCode       = websocket.StatusCode(4000)
	heartbeatTimeout           = 120 * time.Second // Give it 2 minutes to heartbeat
	heartbeatCheckInterval     = 5 * time.Second
	resumeTimeout              = 5 * time.Minute // Give 5 minutes to resume
)

func init() {
	MQClients = append(MQClients, "websocket")
}

// chatServer enables broadcasting to a set of subscribers.
type chatServer struct {
	// sandwich state
	manager *Manager

	subscribers map[[2]int32][]*subscriber
	// the expected token
	expectedToken string

	// external address (used for resuming)
	externalAddress string

	// address
	address string

	// serveMux routes the various endpoints to the appropriate handler.
	serveMux http.ServeMux

	// subscriberMessageBuffer controls the max number
	// of messages that can be queued for a subscriber
	// before it is kicked.
	//
	// Defaults to 100000.
	subscriberMessageBuffer int

	subscribersMu sync.RWMutex
}

type subscriberStatusCode int

const (
	subscriberStatusInit       subscriberStatusCode = iota
	subscriberStatusReady      subscriberStatusCode = iota
	subscriberStatusIdentified subscriberStatusCode = iota
	subscriberStatusResuming   subscriberStatusCode = iota
	subscriberStatusMoving     subscriberStatusCode = iota
	subscriberStatusDead       subscriberStatusCode = iota
)

type subscriberStatusMeta struct {
	status        subscriberStatusCode
	lastHeartbeat time.Time
}

// Sends a close message
type closeMessage struct {
	// close string to be sent
	closeString string
	// close code
	closeCode websocket.StatusCode
}

// subscriber represents a subscriber.
// Messages are sent via writer channels and if the client
// cannot keep up with the messages, closeSlow is called.
type subscriber struct {
	cs                *chatServer
	c                 *websocket.Conn
	context           context.Context
	cancelFunc        context.CancelFunc
	reader            chan structs.SandwichPayload
	writeNormal       chan structs.SandwichPayload // Normal message channel
	writeBytes        chan []byte                  // Bytes message channel
	writeCloseMessage chan closeMessage            // Close message channel
	writeHeartbeat    chan void
	sessionId         string
	shard             [2]int32
	seq               int32
	meta              subscriberStatusMeta
}

// newChatServer constructs a chatServer with the defaults.
func newChatServer() *chatServer {
	cs := &chatServer{
		subscribers: make(map[[2]int32][]*subscriber),
	}
	cs.serveMux.HandleFunc("/", cs.subscribeHandler)
	cs.serveMux.HandleFunc("/publish", cs.publishHandler)

	return cs
}

// addSubscriber registers a subscriber.
func (cs *chatServer) addSubscriber(s *subscriber, shard [2]int32) {
	cs.subscribersMu.Lock()
	defer cs.subscribersMu.Unlock()

	if subs, ok := cs.subscribers[shard]; ok {
		cs.subscribers[shard] = append(subs, s)
	} else {
		cs.subscribers[shard] = []*subscriber{s}
	}
}

// deleteSubscriber deletes the given subscriber.
func (cs *chatServer) deleteSubscriber(s *subscriber) {
	cs.subscribersMu.Lock()
	defer cs.subscribersMu.Unlock()

	if sub, ok := cs.subscribers[s.shard]; ok {
		for i, is := range sub {
			if is.sessionId == s.sessionId || is == s {
				is.cancelFunc()
				cs.subscribers[s.shard] = append(sub[:i], sub[i+1:]...)
			}
		}
	}
}

func newSubscriberStatusMeta() subscriberStatusMeta {
	return subscriberStatusMeta{
		status:        subscriberStatusInit,
		lastHeartbeat: time.Now(),
	}
}

// invalidSession closes the connection with the given reason.
func (cs *chatServer) invalidSession(s *subscriber, reason string, resumable bool) {
	cs.manager.Logger.Error().Msgf("[WS] Invalid session: %s, is resumable: %v", reason, resumable)

	if resumable {
		s.writeBytes <- resumableInvalidSession
	} else {
		s.writeBytes <- nonresumableInvalidSession
	}

	s.writeCloseMessage <- closeMessage{
		closeCode:   invalidSessionOpCode,
		closeString: reason,
	}
}

func (s *subscriber) close() {
	close(s.reader)
	close(s.writeNormal)
	close(s.writeBytes)
	close(s.writeCloseMessage)
	close(s.writeHeartbeat)
	s.cancelFunc()
}

func (s *subscriber) dispatchInitial() error {
	s.cs.manager.Logger.Info().Msgf("[WS] Shard %d/%d (now dispatching events) %v", s.shard[0], s.shard[1], s.shard)

	// Get all guilds
	var guildIdShardIdMap = make(map[discord.GuildID]int32)

	unavailableGuilds := make([]discord.UnavailableGuild, 0)

	s.cs.manager.Sandwich.State.Guilds.Range(func(id discord.GuildID, _ discord.Guild) bool {
		shardId := int32(s.cs.manager.GetShardIdOfGuild(id, s.cs.manager.ConsumerShardCount()))
		guildIdShardIdMap[id] = shardId // We need this when dispatching guilds
		if shardId == s.shard[0] {
			unavailableGuilds = append(unavailableGuilds, discord.UnavailableGuild{
				ID:          id,
				Unavailable: false,
			})
		}
		return false
	})

	// First send READY event with our initial state
	readyPayload := map[string]any{
		"v":          10,
		"user":       s.cs.manager.User,
		"session_id": s.sessionId,
		"shard":      []int32{s.shard[0], s.shard[1]},
		"application": map[string]any{
			"id":    s.cs.manager.User.ID,
			"flags": int32(s.cs.manager.User.Flags),
		},
		"resume_gateway_url": s.cs.externalAddress,
		"guilds":             unavailableGuilds,
	}

	select {
	case <-s.context.Done():
		return nil
	default:
	}

	serializedReadyPayload, err := sandwichjson.Marshal(readyPayload)

	if err != nil {
		s.cs.manager.Logger.Error().Msgf("[WS] Failed to marshal ready payload: %s", err.Error())
		return err
	}

	s.cs.manager.Logger.Info().Msgf("[WS] Dispatching ready to shard %d", s.shard[0])

	s.writeNormal <- structs.SandwichPayload{
		Op:   discord.GatewayOpDispatch,
		Data: serializedReadyPayload,
		Type: "READY",
	}

	// Next dispatch guilds
	s.cs.manager.Sandwich.State.Guilds.Range(func(id discord.GuildID, _ discord.Guild) bool {
		shardId, ok := guildIdShardIdMap[id]

		if !ok {
			// Get shard id
			shardId = int32(s.cs.manager.GetShardIdOfGuild(id, s.cs.manager.ConsumerShardCount()))
		}

		if shardId != s.shard[0] {
			return false // Skip to next guild if the shard id is not the same
		}

		guild, ok := s.cs.manager.Sandwich.State.GetGuild(id)

		if !ok {
			s.cs.manager.Logger.Error().Msgf("[WS] Failed to get guild: %d", id)
			return false
		}

		serializedGuild, err := sandwichjson.Marshal(guild)

		if err != nil {
			s.cs.manager.Logger.Error().Msgf("[WS] Failed to marshal guild: %s [shard %d]", err.Error(), s.shard[0])
			return false
		}

		s.writeNormal <- structs.SandwichPayload{
			Op:   discord.GatewayOpDispatch,
			Data: serializedGuild,
			Type: "GUILD_CREATE",
		}

		select {
		case <-s.context.Done():
			return true
		default:
			return false
		}
	})

	s.cs.manager.Logger.Info().Msgf("[WS] Shard %d (initial state dispatched successfully)", s.shard[0])

	return err
}

func (cs *chatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cs.serveMux.ServeHTTP(w, r)
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (cs *chatServer) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	cs.subscribe(r.Context(), w, r)
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

	err = sandwichjson.Unmarshal(msg, &payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	cs.publish(shard, &payload)

	w.WriteHeader(http.StatusAccepted)
}

// identifyClient tries to identify a incoming connection
//
// Note that identifyClient will only return nil, nil on success or if the main context dies
func (s *subscriber) identifyClient() (oldSess *subscriber, err error) {
	// Send the initial hello payload and wait for identify
	// If the client does not identify within 5 seconds, close the connection
	s.writeBytes <- helloPayload

	// Keep reading messages till we reach an identify
	for {
		select {
		case <-s.context.Done():
			return nil, nil
		case <-time.After(5 * time.Second):
			return nil, errors.New("timed out waiting for identify")
		case packet := <-s.reader:
			// Read an identify packet
			//
			// Note that resume is not supported at this time
			if packet.Op == discord.GatewayOpIdentify {
				var identify struct {
					Token string   `json:"token"`
					Shard [2]int32 `json:"shard"`
				}

				err := sandwichjson.Unmarshal(packet.Data, &identify)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal identify packet: %w", err)
				}

				if len(identify.Shard) != 2 {
					return nil, errors.New("invalid shard")
				}

				identify.Token = strings.Replace(identify.Token, "Bot ", "", 1)

				if identify.Token != s.cs.expectedToken {
					return nil, errors.New("invalid token")
				}

				s.sessionId = randomHex(12)

				csc := s.cs.manager.ConsumerShardCount() // Get the consumer shard count to avoid unneeded casts

				// dpy/serenity workaround
				if identify.Shard[1] <= 0 {
					identify.Shard[1] = csc
				}

				if identify.Shard[1] > csc {
					return nil, fmt.Errorf("invalid shard count: %d > %d", identify.Shard[1], csc)
				} else if identify.Shard[0] > csc {
					return nil, fmt.Errorf("invalid shard id: %d > %d", identify.Shard[0], csc)
				}

				s.shard = identify.Shard

				s.cs.manager.Logger.Info().Msgf("[WS] Shard %d is now identified with created session id %s [%s]", s.shard[0], s.sessionId, fmt.Sprint(s.shard))
				return nil, nil
			} else if packet.Op == discord.GatewayOpResume {
				var resume struct {
					Token     string `json:"token"`
					SessionID string `json:"session_id"`
					Seq       int32  `json:"seq"`
				}

				err := sandwichjson.Unmarshal(packet.Data, &resume)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal resume packet: %w", err)
				}

				resume.Token = strings.Replace(resume.Token, "Bot ", "", 1)

				if resume.Token == s.cs.expectedToken {
					csc := s.cs.manager.ConsumerShardCount() // Get the consumer shard count to avoid unneeded casts

					// Find session with same session id
					s.cs.subscribersMu.RLock()
					for _, shardSubs := range s.cs.subscribers {
						for _, oldSess := range shardSubs {
							if s.sessionId == resume.SessionID {
								s.cs.manager.Logger.Info().Msgf("[WS] Shard %d is now identified with resumed session id %s [%s]", s.shard[0], s.sessionId, fmt.Sprint(s.shard))
								s.seq = resume.Seq
								s.shard = oldSess.shard
								if s.shard[1] <= 0 {
									s.shard[1] = csc // Ensure csc is set correctly on resume
								}
								s.cs.subscribersMu.RUnlock()
								return oldSess, nil
							}
						}
					}
					s.cs.subscribersMu.RUnlock()
					return nil, errors.New("invalid session id")
				} else {
					return nil, errors.New("invalid token")
				}
			}
		}
	}
}

// handleTimeoutWatchdog is a special watchdog that looks for timing out functions and clears them from the session
func (s *subscriber) handleTimeoutWatchdog() {
	defer func() {
		if err := recover(); err != nil {
			s.cs.manager.Logger.Error().Msgf("[WS] Shard %d panicked on handleTimeoutWatchdog: %v", s.shard[0], err)
			return
		}
	}()

	defer s.cancelFunc()

	for {
		select {
		case <-s.context.Done():
			return
		case <-time.After(heartbeatCheckInterval):
			if time.Since(s.meta.lastHeartbeat) > heartbeatTimeout || s.meta.status == subscriberStatusIdentified {
				s.cs.manager.Logger.Error().Msgf("[WS] Shard %d timed out", s.shard[0])
				s.cancelFunc()
			}
		}
	}
}

// readMessages reads messages from subscribe and sends them to the reader
// Note that there must be only one reader reading from the goroutine
func (s *subscriber) readMessages() {
	defer func() {
		if err := recover(); err != nil {
			s.cs.manager.Logger.Error().Msgf("[WS] Shard %d panicked on readMessages: %v", s.shard[0], err)
			return
		}
	}()

	defer s.cancelFunc()

	for {
		select {
		case <-s.context.Done():
			return
		default:
			_, ior, err := s.c.Read(s.context)

			if err != nil {
				return
			}

			var payload structs.SandwichPayload

			err = sandwichjson.Unmarshal(ior, &payload)

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to unmarshal packet: %s", err.Error())
				s.cs.invalidSession(s, "failed to unmarshal packet: "+err.Error(), true)
				return
			}

			if payload.Op == discord.GatewayOpHeartbeat {
				s.meta.lastHeartbeat = time.Now()
				s.writeHeartbeat <- struct{}{}
			} else {
				s.reader <- payload
			}
		}
	}
}

func (cs *chatServer) getShard(shardId int32) *Shard {
	var shardRes *Shard
	cs.manager.ShardGroups.Range(func(k int32, sg *ShardGroup) bool {
		sg.Shards.Range(func(i int32, sh *Shard) bool {
			if sh.ShardID == shardId {
				shardRes = sh
				return true
			}
			return false
		})

		return shardRes != nil
	})

	return shardRes
}

// handleReadMessages handles messages from reader
func (s *subscriber) handleReadMessages() {
	defer func() {
		if err := recover(); err != nil {
			s.cs.manager.Logger.Error().Msgf("[WS] Shard %d panicked on handleReadMessages: %v", s.shard[0], err)
			return
		}
	}()

	defer s.cancelFunc()

	for {
		select {
		case <-s.context.Done():
			return
		case msg := <-s.reader:
			// Send to discord directly
			s.cs.manager.Logger.Debug().Msgf("[WS] Shard %d got/found packet: %v %s", s.shard[0], msg, string(msg.Data))

			// Try finding guild_id
			var shardId = s.shard[0]
			if s.shard[1] != s.cs.manager.noShards {
				s.cs.manager.Logger.Info().Msgf("Shard %d is not using global shard count, remapping to real shard for read message %v", s.shard[0], msg)

				var guildId struct {
					GuildID discord.GuildID `json:"guild_id"`
				}

				err := sandwichjson.Unmarshal(msg.Data, &guildId)

				if err != nil || guildId.GuildID == 0 {
					s.cs.manager.Logger.Info().Msgf("No guild_id found in recieved packet %s", msg.Data)
					continue
				}

				shardId = int32(s.cs.manager.GetShardIdOfGuild(guildId.GuildID, s.cs.manager.noShards))
				s.cs.manager.Logger.Info().Msgf("Remapped shard id %d to %d", s.shard[0], shardId)
			}

			// Find the shard corresponding to the guild_id
			sh := s.cs.getShard(shardId)

			if s == nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Shard %d not found", shardId)
				continue
			}

			err := sh.SendEvent(sh.ctx, msg.Op, msg.Data)

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to send event: %s", err.Error())
			}
		}
	}
}

// writeMessages reads messages from the writer and sends them to the WebSocket
func (s *subscriber) writeMessages() {
	defer func() {
		if err := recover(); err != nil {
			s.cs.manager.Logger.Error().Msgf("[WS] Shard %d panicked on writeMessages: %v", s.shard[0], err)
			return
		}
	}()

	defer s.cancelFunc()

	for {
		select {
		// Case 1: Done is closed, try closing the connection and quitting
		case <-s.context.Done():
			s.c.Write(s.context, websocket.MessageText, []byte(`{"op":9,"d":true}`))

			err := s.c.Close(invalidSessionOpCode, string(resumableInvalidSession))

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to close session: %s", err.Error())
			}

			return // Closed context
		// Case 2: Normal message
		case msg := <-s.writeNormal:
			if msg.Op == discord.GatewayOpDispatch {
				msg.Sequence = s.seq
				s.seq++
			} else {
				msg.Sequence = 0
			}

			serializedMessage, err := sandwichjson.Marshal(msg)

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to marshal message: %s", err.Error())
				continue
			}

			err = s.c.Write(s.context, websocket.MessageText, serializedMessage)

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to write message [serialized]: %s", err.Error())
				s.c.Close(websocket.StatusInternalError, "Failed to write message [serialized]")
				return
			}
		// Case 3: Optimized write bytes
		case msg := <-s.writeBytes:
			err := s.c.Write(s.context, websocket.MessageText, msg)

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to write message [rawBytes]: %s", err.Error())
				s.c.Close(websocket.StatusInternalError, "Failed to write message [rawBytes]")
				return
			}
		// Case 4: Heartbeat
		case <-s.writeHeartbeat:
			err := s.c.Write(s.context, websocket.MessageText, heartbeatAck)

			if err != nil {
				s.cs.manager.Logger.Error().Msgf("[WS] Failed to write heartbeat: %s", err.Error())
				return
			}
		// Case 5: Close message
		case msg := <-s.writeCloseMessage:
			s.meta.status = subscriberStatusDead
			s.cancelFunc()
			s.c.Close(msg.closeCode, msg.closeString)
			return
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
	if !cs.manager.AllReady() {
		http.Error(w, "{\"error\":\"Manager is not yet ready to accept connections\"}", http.StatusServiceUnavailable)
		return errors.New("manager is not yet ready to accept connections")
	}

	cs.manager.Logger.Info().Str("url", r.URL.String()).Msgf("[WS] Shard %d is now subscribing", 0)

	var c *websocket.Conn
	s := &subscriber{
		cs:                cs,
		reader:            make(chan structs.SandwichPayload, cs.subscriberMessageBuffer),
		writeNormal:       make(chan structs.SandwichPayload, cs.subscriberMessageBuffer),
		writeCloseMessage: make(chan closeMessage, cs.subscriberMessageBuffer),
		writeBytes:        make(chan []byte, cs.subscriberMessageBuffer),
		writeHeartbeat:    make(chan void, cs.subscriberMessageBuffer),
		meta:              newSubscriberStatusMeta(),
	}

	// Create cancellable ctx
	s.context, s.cancelFunc = context.WithCancel(ctx)

	var err error
	c, err = websocket.Accept(w, r, nil)

	if err != nil {
		return err
	}

	c.SetReadLimit(WebsocketReadLimit)

	s.c = c

	if cs.manager.Sandwich == nil {
		c.Close(websocket.StatusInternalError, "sandwich is nil")
		return errors.New("sandwich is nil")
	}

	defer c.Close(invalidSessionOpCode, string(resumableInvalidSession))

	// Start the reader, writer and watchdog
	go s.writeMessages()
	go s.readMessages()
	go s.handleTimeoutWatchdog()

	cs.manager.Logger.Info().Msgf("[WS] Shard %d is now launched (reader+writer UP)", s.shard[0])

	// Now identifyClient
	oldSess, err := s.identifyClient()

	if err != nil {
		cs.invalidSession(s, err.Error(), false)
		return err
	}

	cs.addSubscriber(s, s.shard)

	// SAFETY: There should be no other reader at this point, so start up handleReadMessages
	go s.handleReadMessages()

	if oldSess != nil {
		s.meta.status = subscriberStatusResuming
		oldSess.meta.status = subscriberStatusMoving

		for msg := range oldSess.writeNormal {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-oldSess.context.Done():
				return oldSess.context.Err()
			default:
			}

			if msg.Op != discord.GatewayOpDispatch {
				continue
			}

			s.writeNormal <- msg
		}

		s.meta.status = subscriberStatusReady
		oldSess.close() // Cleanup old session
	} else {
		s.meta.status = subscriberStatusIdentified // Now dispatch the initial data
	}

	cs.manager.Logger.Info().Msgf("[WS] Shard %d is now connected (oldSess fanout done)", s.shard[0])

	if s.meta.status == subscriberStatusResuming {
		// Send a RESUMED event
		s.writeNormal <- structs.SandwichPayload{
			Op:   discord.GatewayOpDispatch,
			Data: []byte(`{}`),
			Type: "RESUMED",
		}
	} else {
		s.dispatchInitial()
	}

	// Set the status to ready
	s.meta.status = subscriberStatusReady

	// Wait for the context to be cancelled
	<-s.context.Done()

	cs.manager.Logger.Info().Msgf("[WS] Shard %d is now disconnected (but can be resumed)", s.shard[0])

	// Give one minute for resumes
	time.Sleep(resumeTimeout)

	if s.meta.status != subscriberStatusMoving {
		s.close() // Close if not being moved
	}

	// Delete the subscriber
	s.cs.deleteSubscriber(s)
	return nil

}

// publish publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (cs *chatServer) publish(shard [2]int32, msg *structs.SandwichPayload) {
	cs.manager.Logger.Trace().Msgf("[WS] Shard %d is now publishing message", shard[0])

	cs.subscribersMu.RLock()
	defer cs.subscribersMu.RUnlock()

	for subShard, sub := range cs.subscribers {
		if subShard[1] != shard[1] && msg.EventDispatchIdentifier.GuildID != nil {
			if subShard[1] <= 0 {
				// 0 shards is impossible, close the connection
				for _, s := range sub {
					cs.invalidSession(s, fmt.Sprintf("Invalid Shard Count %d", subShard[1]), false)
				}
				continue
			}

			// Shard count used by subscriber is not the same as the shard count used by the message
			// We need to remap the shard id based on the subscriber's shard id
			msgShardId := cs.manager.GetShardIdOfGuild(*msg.EventDispatchIdentifier.GuildID, subShard[1])

			if msgShardId != shard[0] {
				continue // Skip if the remapped shard id is not the same
			}
		} else if subShard[0] != shard[0] {
			continue // Skip if the shard id is not the same
		}

		for _, s := range sub {
			cs.manager.Logger.Trace().Msgf("[WS] Shard %d is now publishing message to %d subscribers", shard[0], len(sub))

			s.writeNormal <- *msg
		}
	}
}

// publishGlobal publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (cs *chatServer) publishGlobal(msg *structs.SandwichPayload) {
	cs.manager.Logger.Trace().Msg("[WS] Global is now publishing message")

	cs.subscribersMu.RLock()
	defer cs.subscribersMu.RUnlock()

	for _, shardSubs := range cs.subscribers {
		for _, s := range shardSubs {
			cs.manager.Logger.Trace().Msgf("[WS] Global is now publishing message to %d subscribers", len(shardSubs))

			s.writeNormal <- *msg
		}
	}
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

// Supported options:
//
// address (string): the address to listen on
// expectedToken (string): the expected token for identify
// externalAddress (string): the external address to use for resuming, defaults to ws://address if unset
func (mq *WebsocketClient) Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error {
	var ok bool

	var address string
	var externalAddress string
	var expectedToken string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return errors.New("websocketMQ connect: string type assertion failed for Address")
	}

	externalAddress, ok = GetEntry(args, "ExternalAddress").(string)

	if !ok {
		if !strings.HasPrefix(address, "ws") {
			externalAddress = "ws://" + address
		} else {
			externalAddress = address
		}
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
	mq.cs.externalAddress = externalAddress
	s := &http.Server{Handler: mq.cs}

	switch subscriberMessageBuffer := GetEntry(args, "SubscriberMessageBuffer").(type) {
	case int:
		mq.cs.subscriberMessageBuffer = subscriberMessageBuffer
	case int64:
		mq.cs.subscriberMessageBuffer = int(subscriberMessageBuffer)
	case float64:
		mq.cs.subscriberMessageBuffer = int(subscriberMessageBuffer)
	case string:
		buffer, err := strconv.ParseInt(subscriberMessageBuffer, 10, 64)

		if err != nil {
			return errors.New("websocketMQ connect: failed to parse SubscriberMessageBuffer: " + err.Error())
		}

		mq.cs.subscriberMessageBuffer = int(buffer)
	default:
		manager.Logger.Warn().Msg("SubscriberMessageBuffer not set, defaulting to 16")
		mq.cs.subscriberMessageBuffer = 100000
	}

	go func() {
		s.Serve(l)
	}()

	return nil
}

func (mq *WebsocketClient) Publish(ctx context.Context, packet *structs.SandwichPayload, channelName string) error {
	if len(packet.Metadata.Shard) < 3 {
		mq.cs.publishGlobal(
			packet,
		)
	} else {
		mq.cs.publish(
			[2]int32{packet.Metadata.Shard[1], packet.Metadata.Shard[2]},
			packet,
		)
	}

	return nil
}

func (mq *WebsocketClient) IsClosed() bool {
	return mq.cs == nil
}

func (mq *WebsocketClient) CloseShard(shardID int32, reason MQCloseShardReason) {
	if reason == MQCloseShardReasonGateway {
		return // No-op if the reason is a gateway reconnect
	}

	mq.cs.subscribersMu.RLock()
	defer mq.cs.subscribersMu.RUnlock()

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
	mq.cs.subscribersMu.RLock()
	defer mq.cs.subscribersMu.RUnlock()

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

func (mq *WebsocketClient) StopSession(sessionID string) {
	mq.cs.subscribersMu.RLock()
	defer mq.cs.subscribersMu.RUnlock()

	for _, shardSubs := range mq.cs.subscribers {
		for _, s := range shardSubs {
			if s.sessionId == sessionID {
				mq.cs.invalidSession(s, "Session stopped", true)
				mq.cs.deleteSubscriber(s)
			}
		}
	}
}
