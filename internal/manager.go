package gateway

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/accumulator"
	bucketstore "github.com/TheRockettek/Sandwich-Daemon/pkg/bucketstore"
	"github.com/TheRockettek/Sandwich-Daemon/structs"

	discord "github.com/TheRockettek/Sandwich-Daemon/structs/discord"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/xerrors"
)

const (
	maxClientNumber = 9999
)

// ManagerConfiguration represents the configuration for the manager.
type ManagerConfiguration struct {
	AutoStart bool `json:"auto_start" yaml:"auto_start" msgpack:"auto_start"`
	// Boolean to start the Manager when the daemon starts
	Persist bool `json:"persist" msgpack:"persist"`
	// Boolean to dictate if configuration should be saved

	Identifier  string `json:"identifier" msgpack:"identifier"`
	DisplayName string `json:"display_name" yaml:"display_name" msgpack:"display_name"`
	Token       string `json:"token" msgpack:"token"`

	// Bot specific configuration
	Bot struct {
		DefaultPresence      *discord.UpdateStatus `json:"presence" yaml:"presence"`
		Compression          bool                  `json:"compression" yaml:"compression"`
		GuildSubscriptions   bool                  `json:"guild_subscriptions" yaml:"guild_subscriptions"`
		Retries              int32                 `json:"retries" yaml:"retries"`
		Intents              int                   `json:"intents" yaml:"intents"`
		LargeThreshold       int                   `json:"large_threshold" yaml:"large_threshold"`
		MaxHeartbeatFailures int                   `json:"max_heartbeat_failures" yaml:"max_heartbeat_failures"`
	} `json:"bot" yaml:"bot"`

	Caching struct {
		RedisPrefix string `json:"redis_prefix" yaml:"redis_prefix"`

		RequestChunkSize int  `json:"request_chunk_size" yaml:"request_chunk_size"`
		CacheUsers       bool `json:"cache_users" yaml:"cache_users"`
		CacheMembers     bool `json:"cache_members" yaml:"cache_members"`
		RequestMembers   bool `json:"request_members" yaml:"request_members"`
		StoreMutuals     bool `json:"store_mutuals" yaml:"store_mutuals"`
	} `json:"caching" yaml:"caching"`

	Events struct {
		EventBlacklist   []string `json:"event_blacklist" yaml:"event_blacklist"`     // Events completely ignored
		ProduceBlacklist []string `json:"produce_blacklist" yaml:"produce_blacklist"` // Events not sent to consumers

		// IgnoreBots will not pass MESSAGE_CREATE events to consumers if the author was
		// a bot.
		IgnoreBots bool `json:"ignore_bots" yaml:"ignore_bots"`
		// CheckPrefixes will HGET {REDIS_PREFIX}:prefix with the key GUILDID after receiving
		// a MESSAGE_CREATE and if it is not null and the message content does not start with
		// the prefix, it will not send the message to consumers. Useful if you only want to
		// receive commands.
		CheckPrefixes bool `json:"check_prefixes" yaml:"check_prefixes"`
		// Also allows for a bot mention to be a prefix
		AllowMentionPrefix bool `json:"allow_mention_prefix" yaml:"allow_mention_prefix"`

		// FallbackPrefix is the default prefix along with mention prefix (if enabled) when no
		// entry can be found in redis. If empty it will not treat any message as a prefix and
		// will instead discard.
		FallbackPrefix string `json:"fallback_prefix"`
	} `json:"events" yaml:"events"`

	// Messaging specific configuration
	Messaging struct {
		ClientName string `json:"client_name" yaml:"client_name" msgpack:"client_name"`
		// If empty, this will use SandwichConfiguration.NATS.Channel which all Managers
		// should use by default.
		ChannelName string `json:"channel_name" yaml:"channel_name" msgpack:"channel_name"`
		// UseRandomSuffix will append numbers to the end of the client name in order to
		// reduce likelihood of clashing cluster IDs.
		UseRandomSuffix bool `json:"use_random_suffix" yaml:"use_random_suffix" msgpack:"use_random_suffix"`
	} `json:"messaging" yaml:"messaging"`

	// Sharding specific configuration
	Sharding struct {
		AutoSharded bool `json:"auto_sharded" yaml:"auto_sharded" msgpack:"auto_sharded"`
		ShardCount  int  `json:"shard_count" yaml:"shard_count" msgpack:"shard_count"`

		ClusterCount int `json:"cluster_count" yaml:"cluster_count" msgpack:"cluster_count"`
		ClusterID    int `json:"cluster_id" yaml:"cluster_id" msgpack:"cluster_id"`
	} `json:"sharding" msgpack:"sharding"`
}

// Manager represents a bot instance.
type Manager struct {
	ctx    context.Context
	cancel func()

	ErrorMu sync.RWMutex `json:"-"`
	Error   string       `json:"error"`

	AnalyticsMu sync.RWMutex             `json:"-"`
	Analytics   *accumulator.Accumulator `json:"-"`

	Sandwich *Sandwich      `json:"-"`
	Logger   zerolog.Logger `json:"-"`

	ConfigurationMu sync.RWMutex             `json:"-"`
	Configuration   *ManagerConfiguration    `json:"configuration"`
	Buckets         *bucketstore.BucketStore `json:"-"`

	RedisClient *redis.Client `json:"-"`
	NatsClient  *nats.Conn    `json:"-"`
	StanClient  stan.Conn     `json:"-"`

	Client *Client `json:"-"`

	GatewayMu sync.RWMutex       `json:"-"`
	Gateway   discord.GatewayBot `json:"gateway"`

	pp sync.Pool

	// ShardGroups contain the group of shards the Manager is managing. The reason
	// we have a ShardGroup instead of a map/slice of shards is we can run multiple
	// shard groups at once. This is used during rolling restarts where we would have
	// a shard group of 160 and 176 active at the same time. Once the 176 shardgroup
	// has finished ready, the other shard group will stop. 176 will not relay messages
	// until it has removed the old shardgroup to reduce likelihood of duplicate messages.
	// These messages will just be completely ignored as if it was in the EventBlacklist
	ShardGroups       map[int32]*ShardGroup `json:"shard_groups"`
	ShardGroupsMu     sync.RWMutex          `json:"-"`
	ShardGroupIter    *int32                `json:"-"`
	ShardGroupCounter sync.WaitGroup        `json:"-"`

	EventBlacklistMu sync.RWMutex `json:"-"`
	EventBlacklist   []string     `json:"-"`

	ProduceBlacklistMu sync.RWMutex `json:"-"`
	ProduceBlacklist   []string     `json:"-"`
}

// NewManager creates a new manager.
func (sg *Sandwich) NewManager(configuration *ManagerConfiguration) (mg *Manager, err error) {
	logger := sg.Logger.With().Str("manager", configuration.DisplayName).Logger()
	logger.Info().Msg("Creating new manager")

	mg = &Manager{
		Sandwich: sg,
		Logger:   logger,

		ErrorMu: sync.RWMutex{},
		Error:   "",

		ConfigurationMu: sync.RWMutex{},
		Configuration:   configuration,
		Buckets:         bucketstore.NewBucketStore(),
		GatewayMu:       sync.RWMutex{},
		Gateway:         discord.GatewayBot{},

		pp: sync.Pool{
			New: func() interface{} { return new(structs.SandwichPayload) },
		},

		ShardGroups:       make(map[int32]*ShardGroup),
		ShardGroupsMu:     sync.RWMutex{},
		ShardGroupIter:    new(int32),
		ShardGroupCounter: sync.WaitGroup{},

		EventBlacklistMu: sync.RWMutex{},
		EventBlacklist:   make([]string, 0),

		ProduceBlacklistMu: sync.RWMutex{},
		ProduceBlacklist:   make([]string, 0),
	}

	if sg.RestTunnelEnabled.IsSet() {
		mg.Client = NewClient(configuration.Token, sg.Configuration.RestTunnel.URL, sg.RestTunnelReverse.IsSet(), true)
	} else {
		mg.Client = NewClient(configuration.Token, "", false, true)
	}

	err = mg.NormalizeConfiguration()
	if err != nil {
		mg.ErrorMu.Lock()
		mg.Error = err.Error()
		mg.ErrorMu.Unlock()

		return nil, err
	}

	mg.ctx, mg.cancel = context.WithCancel(context.Background())

	return mg, err
}

// NormalizeConfiguration fills in any defaults within the configuration.
func (mg *Manager) NormalizeConfiguration() (err error) {
	mg.ConfigurationMu.RLock()
	defer mg.ConfigurationMu.RUnlock()
	mg.Sandwich.ConfigurationMu.RLock()
	defer mg.Sandwich.ConfigurationMu.RUnlock()

	if mg.Configuration.Token == "" {
		return xerrors.New("Manager configuration missing token")
	}

	mg.Configuration.Token = strings.TrimSpace(mg.Configuration.Token)

	if mg.Configuration.Bot.MaxHeartbeatFailures < 1 {
		mg.Configuration.Bot.MaxHeartbeatFailures = 1
	}

	if mg.Configuration.Bot.Retries < 1 {
		mg.Configuration.Bot.Retries = 1
	}

	if mg.Configuration.Sharding.ClusterCount < 1 {
		mg.Configuration.Sharding.ClusterCount = 1
	}

	if mg.Configuration.Caching.RedisPrefix == "" {
		mg.Configuration.Caching.RedisPrefix = strings.ToLower(
			strings.ReplaceAll(mg.Configuration.DisplayName, " ", ""),
		)
		mg.Logger.Info().Msgf("Using redis prefix '%s' as none was provided", mg.Configuration.Caching.RedisPrefix)
	}

	if mg.Configuration.Messaging.ClientName == "" {
		return xerrors.New("Manager missing client name. Try sandwich")
	}

	if mg.Configuration.Messaging.ChannelName == "" {
		mg.Configuration.Messaging.ChannelName = mg.Sandwich.Configuration.NATS.Channel
		mg.Logger.Info().Msg("Using global messaging channel")
	}

	if mg.Configuration.Caching.RequestChunkSize <= 0 {
		mg.Configuration.Caching.RequestChunkSize = 1
	}

	return err
}

// Open starts up the manager, initializes the config and will create a shardgroup.
func (mg *Manager) Open() (err error) {
	mg.Logger.Info().Msg("Starting up manager")

	if mg.ctx == nil {
		mg.ctx, mg.cancel = context.WithCancel(context.Background())
	}

	mg.Sandwich.ConfigurationMu.RLock()
	defer mg.Sandwich.ConfigurationMu.RUnlock()
	mg.ConfigurationMu.RLock()
	defer mg.ConfigurationMu.RUnlock()

	if mg.Sandwich.Configuration.Redis.UniqueClients {
		mg.RedisClient = redis.NewClient(&redis.Options{
			Addr:     mg.Sandwich.Configuration.Redis.Address,
			Password: mg.Sandwich.Configuration.Redis.Password,
			DB:       mg.Sandwich.Configuration.Redis.DB,
		})
	} else {
		mg.RedisClient = mg.Sandwich.RedisClient
	}

	mg.AnalyticsMu.Lock()
	mg.Analytics = accumulator.NewAccumulator(
		mg.ctx,
		Samples,
		Interval,
	)
	mg.AnalyticsMu.Unlock()

	err = mg.RedisClient.Ping(mg.ctx).Err()
	if err != nil {
		return xerrors.Errorf("manager open verify redis: %w", err)
	}

	mg.NatsClient, err = nats.Connect(mg.Sandwich.Configuration.NATS.Address)
	if err != nil {
		return xerrors.Errorf("manager open nats connect: %w", err)
	}

	var clientName string
	if mg.Configuration.Messaging.UseRandomSuffix {
		clientName = mg.Configuration.Messaging.ClientName + "-" + strconv.Itoa(rand.Intn(maxClientNumber)) //nolint:gosec
	} else {
		clientName = mg.Configuration.Messaging.ClientName
	}

	mg.StanClient, err = stan.Connect(
		mg.Sandwich.Configuration.NATS.Cluster,
		clientName,
		stan.NatsConn(mg.NatsClient),
	)
	if err != nil {
		return xerrors.Errorf("manager open stan connect: %w", err)
	}

	mg.EventBlacklistMu.Lock()
	mg.EventBlacklist = mg.Configuration.Events.EventBlacklist
	mg.EventBlacklistMu.Unlock()

	mg.ProduceBlacklistMu.Lock()
	mg.ProduceBlacklist = mg.Configuration.Events.ProduceBlacklist
	mg.ProduceBlacklistMu.Unlock()

	mg.Gateway, err = mg.GetGateway()

	return err
}

// GatherShardCount returns the expected shardcount using the gateway object stored.
func (mg *Manager) GatherShardCount() (shardCount int) {
	mg.Sandwich.ConfigurationMu.RLock()
	defer mg.Sandwich.ConfigurationMu.RUnlock()
	mg.ConfigurationMu.RLock()
	defer mg.ConfigurationMu.RUnlock()
	mg.GatewayMu.RLock()
	defer mg.GatewayMu.RUnlock()

	if mg.Configuration.Sharding.AutoSharded {
		shardCount = mg.Gateway.Shards
	} else {
		shardCount = mg.Configuration.Sharding.ShardCount
	}

	shardCount = int(math.Ceil(float64(shardCount)/float64(mg.Gateway.SessionStartLimit.MaxConcurrency))) *
		mg.Gateway.SessionStartLimit.MaxConcurrency

	return
}

// Scale creates a new ShardGroup and removes old ones once it has finished.
func (mg *Manager) Scale(shardIDs []int, shardCount int, start bool) (ready chan bool, err error) {
	iter := atomic.AddInt32(mg.ShardGroupIter, 1) - 1
	sg := mg.NewShardGroup(iter)
	mg.ShardGroupsMu.Lock()
	mg.ShardGroups[iter] = sg
	mg.ShardGroupsMu.Unlock()

	if start {
		ready, err = sg.Open(shardIDs, shardCount)
	}

	return
}

// PublishEvent sends an event to consumers.
func (mg *Manager) PublishEvent(eventType string, eventData interface{}) (err error) {
	packet := mg.pp.Get().(*structs.SandwichPayload)
	defer mg.pp.Put(packet)

	mg.ConfigurationMu.RLock()
	defer mg.ConfigurationMu.RUnlock()

	packet.Type = eventType
	packet.Op = discord.GatewayOpDispatch
	packet.Data = eventData

	packet.Metadata = structs.SandwichMetadata{
		Version:    VERSION,
		Identifier: mg.Configuration.Identifier,
	}

	// Clear extra values
	packet.Sequence = 0
	packet.Extra = nil
	packet.Trace = nil

	data, err := msgpack.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("publishEvent marshal: %w", err)
	}

	if mg.StanClient != nil {
		err = mg.StanClient.Publish(
			mg.Configuration.Messaging.ChannelName,
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

// GenerateShardIDs returns a slice of shard ids the bot will use and accounts for clusters.
func (mg *Manager) GenerateShardIDs(shardCount int) (shardIDs []int) {
	mg.ConfigurationMu.RLock()
	defer mg.ConfigurationMu.RUnlock()
	deployedShards := shardCount / mg.Configuration.Sharding.ClusterCount

	currentShard := (deployedShards * mg.Configuration.Sharding.ClusterID)
	maxShard := (deployedShards * (mg.Configuration.Sharding.ClusterID + 1))

	for i := currentShard; i < maxShard; i++ {
		shardIDs = append(shardIDs, i)
	}

	return
}

// Close will stop all shardgroups running.
func (mg *Manager) Close() {
	mg.Logger.Info().Msg("Closing down manager")

	mg.ShardGroupsMu.RLock()
	for _, shardGroup := range mg.ShardGroups {
		shardGroup.Close()
	}
	mg.ShardGroupsMu.RUnlock()

	// cancel is not defined when a manager does not autostart
	if mg.cancel != nil {
		mg.cancel()
	}
}

// GetGateway returns response from /gateway/bot.
func (mg *Manager) GetGateway() (resp discord.GatewayBot, err error) {
	_, err = mg.Client.FetchJSON(mg.ctx, "GET", "/gateway/bot", nil, nil, &resp)
	if err != nil {
		return resp, xerrors.Errorf("get gateway fetchjson: %w", err)
	}

	return
}
