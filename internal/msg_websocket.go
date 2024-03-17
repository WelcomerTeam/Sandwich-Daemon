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

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/atomic"
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
	seq        atomic.Int64
	msgs       chan []byte
}

func (s *subscriber) write(p []byte) {
	s.msgs <- p
}

// invalidSession closes the connection with the given reason.
func (cs *chatServer) invalidSession(s *subscriber, reason string, resumable bool) {
	cs.manager.Sandwich.Logger.Error().Msgf("Invalid session: %s, is resumable: %v", reason, resumable)

	if resumable {
		s.write([]byte(`{"op":9,"d":true}`))
		s.c.Close(websocket.StatusCode(4000), "Invalid Session")
	} else {
		s.write([]byte(`{"op":9,"d":false}`))
		s.c.Close(websocket.StatusCode(4000), "Invalid Session")
	}
}

func (cs *chatServer) getShard(shard [2]int32) *Shard {
	cs.manager.shardGroupsMu.RLock()
	defer cs.manager.shardGroupsMu.RUnlock()

	for _, sg := range cs.manager.ShardGroups {
		sg.shardsMu.RLock()
		for _, sh := range sg.Shards {
			if sh.ShardID == shard[0] {
				return sh
			}
		}
		sg.shardsMu.RUnlock()
	}

	return nil
}

func (cs *chatServer) dispatchInitial(ctx context.Context, s *subscriber) error {
	shard := cs.getShard(s.shard)
	guilds := make([]*structs.StateGuild, 0, len(cs.manager.Sandwich.State.Guilds))

	shard.WaitForReady()

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
		"resume_gateway_url": cs.address,
		"guilds": func() []*discord.UnavailableGuild {
			v := []*discord.UnavailableGuild{}
			cs.manager.Sandwich.State.guildsMu.RLock()
			defer cs.manager.Sandwich.State.guildsMu.RUnlock()

			for _, guild := range cs.manager.Sandwich.State.Guilds {
				shardInt, err := getShardIDFromGuildID(guild.ID.String(), int(s.shard[1]))
				if err != nil {
					cs.manager.Sandwich.State.guildsMu.RUnlock()
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

	fp := map[string]any{
		"op": discord.GatewayOpDispatch,
		"d":  readyPayload,
		"t":  "READY",
		"s":  s.seq.Load(),
	}

	cs.manager.Sandwich.Logger.Info().Msgf("Dispatching ready to shard %d", s.shard[0])

	fpBytes, err := jsoniter.Marshal(fp)
	if err != nil {
		return err
	}

	s.write(fpBytes)

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

			guild.Roles = make([]*structs.StateRole, 0, len(roles.Roles))
			for id, role := range roles.Roles {
				role.ID = discord.Snowflake(id)
				guild.Roles = append(guild.Roles, role)
			}
		}

		fp := map[string]any{
			"op": discord.GatewayOpDispatch,
			"d":  guild,
			"t":  "GUILD_CREATE",
			"s":  s.seq.Load(),
		}

		fpBytes, err := jsoniter.Marshal(fp)
		if err != nil {
			cs.manager.Sandwich.Logger.Error().Msgf("Failed to marshal guild create: %s [shard %d]", err.Error(), s.shard[0])
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.write(fpBytes)

		s.seq.Inc()
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

	cs.publish(shard, msg)

	w.WriteHeader(http.StatusAccepted)
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
		msgs: make(chan []byte, cs.subscriberMessageBuffer),
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

	// Before adding the subscriber for external access, send the initial hello payload and wait for identify
	// If the client does not identify within 5 seconds, close the connection
	c.Write(ctx, websocket.MessageText, []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`))

	// Keep reading messages till we reach an identify
	for {
		typ, ior, err := c.Read(ctx)
		if err != nil {
			c.Close(websocket.StatusCode(4000), `[Sandwich] Unable to decode payload`)
			return err
		}

		switch typ {
		case websocket.MessageText:
			var packet struct {
				Data jsoniter.RawMessage `json:"d"`
				Op   discord.GatewayOp   `json:"op"`
			}

			err := jsoniter.Unmarshal(ior, &packet)
			if err != nil {
				c.Close(websocket.StatusCode(4000), `[Sandwich] Unable to decode payload`)
				return err
			}

			// Read an identify packet
			//
			// Note that resume is not supported at this time
			if packet.Op == discord.GatewayOpIdentify {
				var identify struct {
					Token string   `json:"token"`
					Shard [2]int32 `json:"shard"`
				}

				err = jsoniter.Unmarshal(packet.Data, &identify)
				if err != nil {
					c.Close(websocket.StatusCode(4000), `[Sandwich] Unable to decode payload (identify)`)
					return err
				}

				identify.Token = strings.Replace(identify.Token, "Bot ", "", 1)

				if identify.Token == cs.expectedToken {
					s.sessionId = randomHex(12)
					s.shard = identify.Shard
					s.up = true

					cs.manager.Sandwich.Logger.Info().Msgf("Shard %d is now connected with created session id %s [%s]", s.shard[0], s.sessionId, fmt.Sprint(s.shard))
				} else {
					c.Close(websocket.StatusCode(4000), `[Sandwich] Invalid token`)
					return errors.New("invalid token")
				}
			} else if packet.Op == discord.GatewayOpResume {
				var resume struct {
					Token     string `json:"token"`
					SessionID string `json:"session_id"`
					Seq       int64  `json:"seq"`
				}

				err = jsoniter.Unmarshal(packet.Data, &resume)
				if err != nil {
					c.Close(websocket.StatusCode(4000), `[Sandwich] Unable to decode payload (resume)`)
					return err
				}

				resume.Token = strings.Replace(resume.Token, "Bot ", "", 1)

				if resume.Token == cs.expectedToken {
					// Find session with same session id
					cs.subscribersMu.RLock()
					for _, shardSubs := range cs.subscribers {
						for _, oldSess := range shardSubs {
							if s.sessionId == resume.SessionID {
								s.msgs = oldSess.msgs
								s.shard = oldSess.shard
								s.resumed = true
								s.up = true

								s.seq.Store(resume.Seq)
								break
							}
						}

						if s.up {
							break
						}
					}

					if !s.up {
						c.Close(websocket.StatusCode(4000), `[Sandwich] Invalid session id`)
						return errors.New("invalid session id")
					}

					cs.manager.Sandwich.Logger.Info().Msgf("Shard %d is now connected with resumed session id %s [%s]", s.shard[0], s.sessionId, fmt.Sprint(s.shard))
				} else {
					c.Close(websocket.StatusCode(4000), `[Sandwich] Invalid token`)
					return errors.New("invalid token")
				}
			} else if packet.Op == discord.GatewayOpHeartbeat {
				s.write([]byte(`{"op":11}`))
			}
		}

		if s.up {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	cs.addSubscriber(s, s.shard)
	defer cs.deleteSubscriber(s)

	ctx, cancelFunc := context.WithCancel(ctx)
	s.cancelFunc = cancelFunc
	defer s.cancelFunc()

	defer c.Close(websocket.StatusCode(4000), `{"op":9,"d":true}`)

	if !s.resumed {
		go cs.dispatchInitial(ctx, s)
	}

	go func() {
		for {
			typ, ior, err := c.Read(ctx)
			if err != nil {
				cancelFunc()
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			switch typ {
			case websocket.MessageText:
				var packet discord.SentPayload

				err := jsoniter.Unmarshal(ior, &packet)
				if err != nil {
					cs.invalidSession(s, "failed to unmarshal packet: "+err.Error(), true)
					s.cancelFunc()
					return
				}

				if packet.Op == discord.GatewayOpHeartbeat {
					// cs.manager.Sandwich.Logger.Debug().Msgf("Shard heartbeat recieved for shard %d", s.shard[0])
					s.write([]byte(fmt.Sprintf(`{"op":11,"s":%d}`, s.seq.Load())))
				}
			}
		}
	}()

	for {
		select {
		case msg := <-s.msgs:
			err := c.Write(ctx, websocket.MessageText, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// publish publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (cs *chatServer) publish(shard [2]int32, msg []byte) {
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

		s.msgs <- msg
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

func (mq *WebsocketClient) Publish(ctx context.Context, packet *structs.SandwichPayload, channelName string, data []byte) error {
	go mq.cs.publish(
		[2]int32{packet.Metadata.Shard[1], packet.Metadata.Shard[2]},
		data,
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
				s.write([]byte(`{"op":9,"d":true}`))
				s.c.Close(websocket.StatusCode(4000), "Socket closed")

				s.cancelFunc()

				mq.cs.deleteSubscriber(s)
			}
		}
	}
}

func (mq *WebsocketClient) Close() {
	// Send RESUME to all shards
	for _, shardSubs := range mq.cs.subscribers {
		for _, s := range shardSubs {
			s.write([]byte(`{"op":9,"d":true}`))
			s.c.Close(websocket.StatusCode(4000), "Socket closed")

			s.cancelFunc()

			mq.cs.deleteSubscriber(s)
		}
	}

	mq.cs = nil
}
