package gateway

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	bucketstore "github.com/TheRockettek/Sandwich-Daemon/pkg/bucketStore"
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/xerrors"
)

type void struct{}

// ManagerConfiguration represents the configuration for the manager
type ManagerConfiguration struct {
	AutoStart bool `json:"autostart" msgpack:"autostart"` // Boolean to stat the Manager when the bot starts
	Persist   bool `json:"persist" msgpack:"persist"`     // Boolean to dictate if configuration should be saved

	Identifier  string `json:"identifier" msgpack:"identifier"`
	DisplayName string `json:"display_name" msgpack:"display_name"`
	Token       string `json:"token" msgpack:"token"`

	// Bot specific configuration
	Bot struct {
		Compression          bool                  `json:"compression" msgpack:"compression"`
		DefaultPresence      *structs.UpdateStatus `json:"presence" msgpack:"presence"`
		GuildSubscriptions   bool                  `json:"guild_subscriptions" msgpack:"guild_subscriptions"`
		Intents              int                   `json:"intents" msgpack:"intents"`
		LargeThreshold       int                   `json:"large_threshold" msgpack:"large_threshold"`
		MaxHeartbeatFailures int                   `json:"max_heartbeat_failures" msgpack:"max_heartbeat_failures"`
	} `json:"bot" msgpack:"bot"`

	Caching struct {
		RedisPrefix string `json:"redis_prefix" msgpack:"redis_prefix"`

		CacheUsers       bool `json:"cache_users" msgpack:"cache_users"`
		CacheMembers     bool `json:"cache_members" msgpack:"cache_members"`
		RequestMembers   bool `json:"request_members" msgpack:"request_members"`
		RequestChunkSize int  `json:"request_chunk_size" msgpack:"request_chunk_size"`
		StoreMutuals     bool `json:"store_mutuals" msgpack:"store_mutuals"`
	} `json:"caching" msgpack:"caching"`

	Events struct {
		EventBlacklist   []string `json:"event_blacklist" msgpack:"event_blacklist"`     // Events completely ignored
		ProduceBlacklist []string `json:"produce_blacklist" msgpack:"produce_blacklist"` // Events not sent to consumers

		// IgnoreBots will not pass MESSAGE_CREATE events to consumers if the author was
		// a bot.
		IgnoreBots bool `json:"ignore_bots" msgpack:"ignore_bots"`
		// CheckPrefixes will HGET {REDIS_PREFIX}:prefix with the key GUILDID after receiving
		// a MESSAGE_CREATE and if it is not null and the message content does not start with
		// the prefix, it will not send the message to consumers. Useful if you only want to
		// receive commands.
		CheckPrefixes bool `json:"check_prefixes" msgpack:"check_prefixes"`
		// Also allows for a bot mention to be a prefix
		AllowMentionPrefix bool `json:"allow_mention_prefix" msgpack:"allow_mention_prefix"`
	} `json:"events" msgpack:"events"`

	// Messaging specific configuration
	Messaging struct {
		ClientName string `json:"client_name" msgpack:"client_name"`
		// If empty, this will use SandwichConfiguration.NATS.Channel which all Managers
		// should use by default.
		ChannelName string `json:"channel_name" msgpack:"channel_name"`
		// UseRandomSuffix will append numbers to the end of the client name in order to
		// reduce likelyhood of clashing cluster IDs.
		UseRandomSuffix bool `json:"use_random_suffix" msgpack:"use_random_suffix"`
	} `json:"messaging" msgpack:"messaging"`

	// Sharding specific configuration
	Sharding struct {
		// ConcurrentClients dictates the ammount of clients that can simultaneously
		// connect. Disabled if set to 0. If enabled, when sessions start and hit this
		// limit, they will have to wait until a session has finished lazy loading guilds.
		ConcurrentClients int `json:"concurrent_clients" msgpack:"concurrent_clients"`

		AutoSharded bool `json:"autosharded" msgpack:"autosharded"`
		ShardCount  int  `json:"shard_count" msgpack:"shard_count"`
		// Useful when testing and you want to force a shardCount. This simply does not round it up.
		Enforce bool `json:"enforce" msgpack:"enforce"`

		ClusterCount int `json:"cluster_count" msgpack:"cluster_count"`
		ClusterID    int `json:"cluster_id" msgpack:"cluster_id"`
	} `json:"sharding" msgpack:"sharding"`
}

// Manager represents a bot instance
type Manager struct {
	ctx    context.Context
	cancel func()

	Sandwich *Sandwich
	Logger   zerolog.Logger

	Configuration *ManagerConfiguration
	Buckets       *bucketstore.BucketStore

	RedisClient *redis.Client
	NatsClient  *nats.Conn
	StanClient  stan.Conn

	Client  *Client
	Gateway structs.GatewayBot

	pp sync.Pool

	// ShardGroups contain the group of shards the Manager is managing. The reason
	// we have a ShardGroup instead of a map/slice of shards is we can run multiple
	// shard groups at once. This is used during rolling restarts where we would have
	// a shard group of 160 and 176 active at the same time. Once the 176 shardgroup
	// has finished ready, the other shard group will stop. 176 will not relay messages
	// until it has removed the old shardgroup to reduce likelyhood of duplicate messages.
	// These messages will just be completely ignored as if it was in the EventBlacklist
	ShardGroups       map[int32]*ShardGroup
	ShardGroupMu      sync.Mutex
	ShardGroupIter    *int32
	ShardGroupCounter sync.WaitGroup

	EventBlacklist   map[string]void
	ProduceBlacklist map[string]void
}

// NewManager creates a new manager
func (s *Sandwich) NewManager(configuration *ManagerConfiguration) (mg *Manager, err error) {
	logger := s.Logger.With().Str("manager", configuration.DisplayName).Logger()
	logger.Info().Msg("Creating new manager")

	mg = &Manager{
		Sandwich: s,
		Logger:   logger,

		Configuration: configuration,
		Buckets:       bucketstore.NewBucketStore(),

		Client:  NewClient(configuration.Token),
		Gateway: structs.GatewayBot{},

		pp: sync.Pool{
			New: func() interface{} { return new(structs.PublishEvent) },
		},

		ShardGroups:       make(map[int32]*ShardGroup),
		ShardGroupMu:      sync.Mutex{},
		ShardGroupIter:    new(int32),
		ShardGroupCounter: sync.WaitGroup{},

		EventBlacklist:   make(map[string]void),
		ProduceBlacklist: make(map[string]void),
	}

	err = mg.NormalizeConfiguration()
	if err != nil {
		return nil, xerrors.Errorf("new manager: %w", err)
	}

	return
}

// NormalizeConfiguration fills in any defaults within the configuration
func (mg *Manager) NormalizeConfiguration() (err error) {
	if mg.Configuration.Token == "" {
		return xerrors.New("Manager configuration missing token")
	}
	mg.Configuration.Token = strings.TrimSpace(mg.Configuration.Token)

	if mg.Configuration.Bot.MaxHeartbeatFailures < 1 {
		mg.Configuration.Bot.MaxHeartbeatFailures = 1
	}

	if mg.Configuration.Caching.RedisPrefix == "" {
		mg.Configuration.Caching.RedisPrefix = strings.ToLower(
			strings.Replace(mg.Configuration.DisplayName, " ", "", -1))
		mg.Logger.Info().Msgf("Using redis prefix '%s' as none was provided", mg.Configuration.Caching.RedisPrefix)
	}

	if mg.Configuration.Messaging.ClientName == "" {
		return xerrors.New("Manager missing client name. Try sandwich")
	}
	if mg.Configuration.Messaging.ChannelName == "" {
		mg.Configuration.Messaging.ChannelName = mg.Sandwich.Configuration.NATS.Channel
		mg.Logger.Info().Msg("Using global messaging channel")
	}
	if mg.Configuration.Messaging.UseRandomSuffix {
		mg.Configuration.Messaging.ClientName += "-" + strconv.Itoa(rand.Intn(9999))
	}
	if mg.Configuration.Caching.RequestChunkSize <= 0 {
		mg.Configuration.Caching.RequestChunkSize = 1
	}

	return
}

// Open starts up the manager, initializes the config and will create a shardgroup
func (mg *Manager) Open() (err error) {
	mg.Logger.Info().Msg("Starting up manager")
	mg.ctx, mg.cancel = context.WithCancel(context.Background())

	if mg.Sandwich.Configuration.Redis.UniqueClients {
		mg.RedisClient = redis.NewClient(&redis.Options{
			Addr:     mg.Sandwich.Configuration.Redis.Address,
			Password: mg.Sandwich.Configuration.Redis.Password,
			DB:       mg.Sandwich.Configuration.Redis.DB,
		})
	} else {
		mg.RedisClient = mg.Sandwich.RedisClient
	}

	err = mg.RedisClient.Ping(mg.ctx).Err()
	if err != nil {
		return xerrors.Errorf("manager open verify redis: %w", err)
	}

	//
	//
	//

	err = mg.StateGuildMembersChunk(structs.GuildMembersChunk{
		GuildID: snowflake.ID(1),
		Members: []*structs.GuildMember{
			{
				Nick: "test",
				User: &structs.User{
					ID:       snowflake.ID(0),
					Username: "testAccount",
				},
			},
		},
	})
	mg.Logger.Fatal().Msgf("eval result %s", err.Error())

	//
	//
	//

	mg.NatsClient, err = nats.Connect(mg.Sandwich.Configuration.NATS.Address)
	if err != nil {
		return xerrors.Errorf("manager open nats connect: %w", err)
	}

	mg.StanClient, err = stan.Connect(
		mg.Sandwich.Configuration.NATS.Cluster,
		mg.Configuration.Messaging.ClientName,
		stan.NatsConn(mg.NatsClient),
	)
	if err != nil {
		return xerrors.Errorf("manager open stan connect: %w", err)
	}

	for _, value := range mg.Configuration.Events.EventBlacklist {
		mg.EventBlacklist[value] = void{}
	}

	for _, value := range mg.Configuration.Events.ProduceBlacklist {
		mg.ProduceBlacklist[value] = void{}
	}

	sg := mg.NewShardGroup()
	iter := atomic.AddInt32(mg.ShardGroupIter, 1)
	mg.ShardGroups[iter] = sg

	mg.Gateway, err = mg.GetGateway()
	if err != nil {
		return xerrors.Errorf("manager open get gateway: %w", err)
	}
	sg.Logger.Info().Int("sessions", mg.Gateway.SessionStartLimit.Remaining).Msg("Retrieved gateway information")

	var shardCount int
	if mg.Configuration.Sharding.AutoSharded || (mg.Configuration.Sharding.ShardCount < mg.Gateway.Shards/2 && !mg.Configuration.Sharding.Enforce) {
		shardCount = mg.Gateway.Shards
	} else {
		shardCount = mg.Configuration.Sharding.ShardCount
	}

	if !mg.Configuration.Sharding.Enforce {
		// We will round up the shard count depending on the concurrent clients specified
		shardCount = int(math.Ceil(float64(shardCount)/float64(mg.Gateway.SessionStartLimit.MaxConcurrency))) * mg.Gateway.SessionStartLimit.MaxConcurrency
	}

	if shardCount >= mg.Gateway.SessionStartLimit.Remaining {
		return xerrors.Errorf("manager open", ErrSessionLimitExhausted)
	}

	ready, err := sg.Open(mg.GenerateShardIDs(shardCount), shardCount)
	if err != nil {
		return
	}

	// Wait for all shards in ShardGroup to be ready
	<-ready

	return
}

// PublishEvent sends an event to consaumers
func (mg *Manager) PublishEvent(Type string, Data interface{}) (err error) {
	packet := mg.pp.Get().(*structs.PublishEvent)
	defer mg.pp.Put(packet)

	packet.Data = Data
	packet.From = mg.Configuration.Identifier
	packet.From = Type

	data, err := msgpack.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("publishEvent marshal: %w", err)
	}

	err = mg.StanClient.Publish(
		mg.Configuration.Messaging.ChannelName,
		data,
	)
	if err != nil {
		return xerrors.Errorf("publishEvent publish: %w", err)
	}

	return
}

// GenerateShardIDs returns a slice of shard ids the bot will use and accounts for clusters
func (mg *Manager) GenerateShardIDs(shardCount int) (shardIDs []int) {
	deployedShards := shardCount / mg.Configuration.Sharding.ClusterCount
	for i := (deployedShards * mg.Configuration.Sharding.ClusterID); i < (deployedShards * (mg.Configuration.Sharding.ClusterID + 1)); i++ {
		shardIDs = append(shardIDs, i)
	}
	return
}

// Close will stop all shardgroups running
func (mg *Manager) Close() {
	mg.Logger.Info().Msg("Closing down manager")

	for _, shardGroup := range mg.ShardGroups {
		shardGroup.Close()
	}

	// cancel is not defined when a manager does not autostart
	if mg.cancel != nil {
		mg.cancel()
	}

	return
}

// GetGateway returns response from /gateway/bot
func (mg *Manager) GetGateway() (resp structs.GatewayBot, err error) {
	err = mg.Client.FetchJSON("GET", "/gateway/bot", nil, &resp)
	if err != nil {
		return resp, xerrors.Errorf("get gateway fetchjson: %w", err)
	}

	return
}
