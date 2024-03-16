package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/time/rate"
	"nhooyr.io/websocket"
)

func init() {
	MQClients = append(MQClients, "websocket")
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

	// publishLimiter controls the rate limit applied to the publish endpoint.
	//
	// Defaults to one publish every 100ms with a burst of 8.
	publishLimiter *rate.Limiter

	// logf controls where logs are sent.
	// Defaults to log.Printf.
	logf func(f string, v ...interface{})

	// serveMux routes the various endpoints to the appropriate handler.
	serveMux http.ServeMux

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}
}

// newChatServer constructs a chatServer with the defaults.
func newChatServer() *chatServer {
	cs := &chatServer{
		subscriberMessageBuffer: 16,
		logf:                    log.Printf,
		subscribers:             make(map[*subscriber]struct{}),
		publishLimiter:          rate.NewLimiter(rate.Every(time.Millisecond*100), 1000),
	}
	cs.serveMux.HandleFunc("/", cs.subscribeHandler)
	cs.serveMux.HandleFunc("/publish", cs.publishHandler)

	return cs
}

// subscriber represents a subscriber.
// Messages are sent on the msgs channel and if the client
// cannot keep up with the messages, closeSlow is called.
type subscriber struct {
	sessionId  string
	shard      [2]int32
	identified bool
	msgs       chan []byte
	closeSlow  func()
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
		cs.logf("%v", err)
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
	var mu sync.Mutex
	var c *websocket.Conn
	var closed bool
	s := &subscriber{
		msgs: make(chan []byte, cs.subscriberMessageBuffer),
		closeSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			closed = true
			if c != nil {
				c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
			}
		},
	}
	cs.addSubscriber(s)
	defer cs.deleteSubscriber(s)

	c2, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}

	if cs.manager.Sandwich == nil {
		c2.Close(websocket.StatusInternalError, "sandwich is nil")
		return errors.New("sandwich is nil")
	}

	mu.Lock()
	if closed {
		mu.Unlock()
		return net.ErrClosed
	}
	c = c2
	mu.Unlock()
	defer c.CloseNow()

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	var writeMut sync.Mutex

	write := func(ctx context.Context, typ websocket.MessageType, msg []byte) error {
		writeMut.Lock()
		defer writeMut.Unlock()

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		return c.Write(ctx, typ, msg)
	}

	write(ctx, websocket.MessageText, []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`))

	// Given a guild ID, return its shard ID
	getShardIDFromGuildID := func(guildID string, shardCount int) (uint64, error) {
		gidNum, err := strconv.ParseInt(guildID, 10, 64)
		if err != nil {
			return 0, err
		}

		return uint64(gidNum>>22) % uint64(shardCount), nil
	}

	// Publish calls shard.OnDispatch
	getShard := func() *Shard {
		cs.manager.shardGroupsMu.RLock()
		defer cs.manager.shardGroupsMu.RUnlock()

		for _, sg := range cs.manager.ShardGroups {
			sg.shardsMu.RLock()
			for _, sh := range sg.Shards {
				if sh.ShardID == s.shard[0] {
					return sh
				}
			}
			sg.shardsMu.RUnlock()
		}

		return nil
	}

	var shard *Shard

	go func() {
		// Call this function to close the connection and return
		invalidSession := func(reason string) {
			cs.manager.Sandwich.Logger.Error().Msgf("Invalid session: %s", reason)
			write(ctx, websocket.MessageText, []byte(`{"op":9,"d":false}`))
			c.Close(websocket.StatusCode(4000), "Invalid Session")
		}

		dispatchInitial := func() error {
			guilds := make([]*structs.StateGuild, 0, len(cs.manager.Sandwich.State.Guilds))

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

					if len(cs.manager.Sandwich.State.Guilds) == 0 {
						invalidSession("no guilds available")
						cancelFunc()
						return v
					}

					for _, guild := range cs.manager.Sandwich.State.Guilds {
						shardInt, err := getShardIDFromGuildID(guild.ID.String(), int(s.shard[1]))
						if err != nil {
							cs.manager.Sandwich.State.guildsMu.RUnlock()
							invalidSession("failed to get shard ID from guild ID: " + err.Error())
							cancelFunc()
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

			fp := map[string]any{
				"op": discord.GatewayOpDispatch,
				"d":  readyPayload,
				"t":  "READY",
				"s":  shard.Sequence.Load(),
			}

			cs.manager.Sandwich.Logger.Info().Msgf("Dispatching ready to shard %d", s.shard[0])

			fpBytes, err := jsoniter.Marshal(fp)
			if err != nil {
				return err
			}

			write(ctx, websocket.MessageText, fpBytes)

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
					"s":  shard.Sequence.Load(),
				}

				fpBytes, err := jsoniter.Marshal(fp)
				if err != nil {
					cs.manager.Sandwich.Logger.Error().Msgf("Failed to marshal guild create: %s [shard %d]", err.Error(), s.shard[0])
					continue
				}

				write(ctx, websocket.MessageText, fpBytes)

				shard.Sequence.Inc()
			}

			return nil
		}

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
					invalidSession("failed to unmarshal packet: " + err.Error())
					cancelFunc()
					return
				}

				if !s.identified {
					// Read an identify packet
					//
					// Note that resume is not supported at this time
					if packet.Op == discord.GatewayOpIdentify {
						bytes, err := jsoniter.Marshal(packet.Data)
						if err != nil {
							invalidSession("failed to marshal packet data: " + err.Error())
							cancelFunc()
							return
						}

						var identify struct {
							Token string   `json:"token"`
							Shard [2]int32 `json:"shard"`
						}

						err = jsoniter.Unmarshal(bytes, &identify)
						if err != nil {
							invalidSession("failed to unmarshal identify: " + err.Error())
							cancelFunc()
							return
						}

						identify.Token = strings.Replace(identify.Token, "Bot ", "", 1)

						if identify.Token == cs.expectedToken {
							s.identified = true
							s.sessionId = randomHex(12)
							s.shard = identify.Shard

							shard = getShard()
							cs.manager.Sandwich.Logger.Info().Msgf("Shard %d is now connected with shard session id %s [%s]", s.shard[0], shard.SessionID, fmt.Sprint(s.shard))

							go func() {
								err := dispatchInitial()
								if err != nil {
									invalidSession("failed to dispatch initial: " + err.Error())
									cancelFunc()
									return
								}
							}()
							continue
						}

						invalidSession("invalid token, got " + identify.Token)
						cancelFunc()
						return
					}
				}

				if packet.Op == discord.GatewayOpHeartbeat {
					if !s.identified {
						write(ctx, websocket.MessageText, []byte(`{"op":11}`))
						continue
					}

					// cs.manager.Sandwich.Logger.Debug().Msgf("Shard heartbeat recieved for shard %d", s.shard[0])
					write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"op":11,"s":%d}`, shard.Sequence.Load())))
				}
			}
		}
	}()

	for {
		select {
		case msg := <-s.msgs:
			err := write(ctx, websocket.MessageText, msg)
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
	cs.subscribersMu.Lock()
	defer cs.subscribersMu.Unlock()

	cs.publishLimiter.Wait(context.Background())

	for s := range cs.subscribers {
		if !s.identified || s.shard != shard {
			continue
		}

		select {
		case s.msgs <- msg:
		default:
			go s.closeSlow()
		}
	}
}

// addSubscriber registers a subscriber.
func (cs *chatServer) addSubscriber(s *subscriber) {
	cs.subscribersMu.Lock()
	cs.subscribers[s] = struct{}{}
	cs.subscribersMu.Unlock()
}

// deleteSubscriber deletes the given subscriber.
func (cs *chatServer) deleteSubscriber(s *subscriber) {
	cs.subscribersMu.Lock()
	delete(cs.subscribers, s)
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
	s := &http.Server{
		Handler:      mq.cs,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

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
