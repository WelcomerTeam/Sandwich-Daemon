package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
)

// TODO: Make these config options
const (
	ShardMaxRetries              = 5
	ShardCompression             = true
	ShardLargeThreshold          = 250
	ShardMaxHeartbeatFailures    = 50
	MessagingMaxClientNameNumber = 9999

	StandardIdentifyLimit = 5
	IdentifyRetry         = (StandardIdentifyLimit * time.Second)
	IdentifyRateLimit     = (StandardIdentifyLimit * time.Second) + (500 * time.Millisecond)
)

// Manager represents a single application.
type Manager struct {
	Logger zerolog.Logger `json:"-"`

	ctx context.Context

	ProducerClient MQClient `json:"-"`

	cancel func()

	Error *atomic.String `json:"error" yaml:"error"`

	Identifier *atomic.String `json:"-"`

	Sandwich      *Sandwich             `json:"-"`
	Configuration *ManagerConfiguration `json:"configuration" yaml:"configuration"`

	ShardGroups *csmap.CsMap[int32, *ShardGroup] `json:"shard_groups" yaml:"shard_groups"`

	Client *Client `json:"-"`

	UserID *atomic.Int64 `json:"id"`

	shardGroupCounter *atomic.Int32

	metadata       *sandwich_structs.SandwichMetadata
	eventBlacklist []string

	produceBlacklist []string

	Gateway discord.GatewayBotResponse `json:"gateway" yaml:"gateway"`

	User discord.User `json:"user"`

	configurationMu sync.RWMutex

	gatewayMu sync.RWMutex

	userMu sync.RWMutex

	eventBlacklistMu sync.RWMutex

	produceBlacklistMu sync.RWMutex

	metadataMu sync.RWMutex

	clientMu sync.Mutex

	noShards int32
}

// ManagerConfiguration represents the configuration for the manager.
type ManagerConfiguration struct {
	Sharding struct {
		ShardIDs    string `json:"shard_ids" yaml:"shard_ids"`
		ShardCount  int32  `json:"shard_count" yaml:"shard_count"`
		AutoSharded bool   `json:"auto_sharded" yaml:"auto_sharded"`
	} `json:"sharding" yaml:"sharding"`
	// Unique name that will be referenced internally
	Identifier string `json:"identifier" yaml:"identifier"`
	// Non-unique name that is sent to consumers.
	ProducerIdentifier string `json:"producer_identifier" yaml:"producer_identifier"`

	FriendlyName string `json:"friendly_name" yaml:"friendly_name"`

	Token string `json:"token" yaml:"token"`

	Events struct {
		EventBlacklist   []string `json:"event_blacklist" yaml:"event_blacklist"`
		ProduceBlacklist []string `json:"produce_blacklist" yaml:"produce_blacklist"`
	} `json:"events" yaml:"events"`

	Messaging struct {
		ClientName      string `json:"client_name" yaml:"client_name"`
		ChannelName     string `json:"channel_name" yaml:"channel_name"`
		UseRandomSuffix bool   `json:"use_random_suffix" yaml:"use_random_suffix"`
	} `json:"messaging" yaml:"messaging"`

	// Bot specific configuration
	Bot struct {
		DefaultPresence      discord.UpdateStatus `json:"default_presence" yaml:"default_presence"`
		Intents              int32                `json:"intents" yaml:"intents"`
		ChunkGuildsOnStartup bool                 `json:"chunk_guilds_on_startup" yaml:"chunk_guilds_on_startup"`
	} `json:"bot" yaml:"bot"`

	Caching struct {
		CacheUsers   bool `json:"cache_users" yaml:"cache_users"`
		CacheMembers bool `json:"cache_members" yaml:"cache_members"`
		StoreMutuals bool `json:"store_mutuals" yaml:"store_mutuals"`
		// TODO: Flexible caching
	} `json:"caching" yaml:"caching"`

	AutoStart    bool `json:"auto_start" yaml:"auto_start"`
	DisableTrace bool `json:"disable_trace" yaml:"disable_trace"`

	VirtualShards struct {
		Enabled bool  `json:"enabled" yaml:"enabled"`
		Count   int32 `json:"count" yaml:"count"`       // Number of virtual shards to use
		DmShard int32 `json:"dm_shard" yaml:"dm_shard"` // Shard to use for DMs and non identifiable events
	} `json:"virtual_shards" yaml:"virtual_shards"`

	Rest struct {
		GetGatewayBot struct {
			MaxConcurrency int32 `json:"max_concurrency" yaml:"max_concurrency"` // How many requests to allow in the custom get gateway bot impl
		}
	}
}

// NewManager creates a new manager.
func (sg *Sandwich) NewManager(configuration *ManagerConfiguration) (mg *Manager) {
	logger := sg.Logger.With().Str("manager", configuration.Identifier).Logger()
	logger.Info().Msg("Creating new manager")

	mg = &Manager{
		Sandwich: sg,
		Logger:   logger,

		Error: atomic.NewString(""),

		configurationMu: sync.RWMutex{},
		Configuration:   configuration,

		Identifier: atomic.NewString(configuration.Identifier),

		gatewayMu: sync.RWMutex{},
		Gateway:   discord.GatewayBotResponse{},

		ShardGroups: csmap.Create(
			csmap.WithSize[int32, *ShardGroup](0),
		),

		Client: NewClient(baseURL, configuration.Token),

		UserID: &atomic.Int64{},

		userMu: sync.RWMutex{},
		User:   discord.User{},

		shardGroupCounter: &atomic.Int32{},

		eventBlacklistMu: sync.RWMutex{},
		eventBlacklist:   configuration.Events.EventBlacklist,

		produceBlacklistMu: sync.RWMutex{},
		produceBlacklist:   configuration.Events.ProduceBlacklist,

		metadataMu: sync.RWMutex{},
		metadata: &sandwich_structs.SandwichMetadata{
			Version:     VERSION,
			Identifier:  configuration.Identifier,
			Application: configuration.Identifier, // TODO: Change this
		},
	}

	mg.ctx, mg.cancel = context.WithCancel(sg.ctx)

	return mg
}

// Initialize handles the start up process including connecting the message queue client.
func (mg *Manager) Initialize(forceRestartProducers bool) error {
	gateway, err := mg.GetGateway()
	if err != nil {
		return err
	}

	var producerRestart bool
	if forceRestartProducers {
		producerRestart = true
	} else {
		if mg.ProducerClient == nil {
			producerRestart = true
		} else if mg.ProducerClient.IsClosed() {
			producerRestart = true
		}
	}

	clientName := mg.Configuration.Messaging.ClientName
	if mg.Configuration.Messaging.UseRandomSuffix {
		clientName = clientName + "-" + randomHex(6)
	}

	if producerRestart {
		if mg.ProducerClient != nil {
			// Close
			mg.ProducerClient.Close()
		}

		producerClient, err := NewMQClient(mg.Sandwich.Configuration.Producer.Type)
		if err != nil {
			return err
		}

		err = producerClient.Connect(
			mg.ctx,
			mg,
			clientName,
			mg.Sandwich.Configuration.Producer.Configuration,
		)
		if err != nil {
			mg.Logger.Error().Err(err).Msg("Failed to connect producer client")

			return fmt.Errorf("failed to connect to producer: %w", err)
		}

		mg.ProducerClient = producerClient
	}

	mg.gatewayMu.Lock()
	mg.Gateway = gateway
	mg.gatewayMu.Unlock()

	return nil
}

// Open handles retrieving shard counts and scaling.
func (mg *Manager) Open() error {
	shardIDs, shardCount := mg.getInitialShardCount(
		mg.Configuration.Sharding.ShardCount,
		mg.Configuration.Sharding.ShardIDs,
		mg.Configuration.Sharding.AutoSharded,
	)

	sg := mg.Scale(shardIDs, shardCount)

	ready, err := sg.Open()
	if err != nil {
		go mg.Sandwich.PublishSimpleWebhook(
			"Failed to scale manager",
			"`"+err.Error()+"`",
			"Manager: "+mg.Configuration.Identifier,
			EmbedColourDanger,
		)

		return err
	}

	<-ready

	return nil
}

// GetGateway returns the response from /gateway/bot.
func (mg *Manager) GetGateway() (resp discord.GatewayBotResponse, err error) {
	mg.Sandwich.gatewayLimiter.Lock()

	mg.clientMu.Lock()
	defer mg.clientMu.Unlock()

	_, err = mg.Client.FetchJSON(mg.ctx, "GET", "/gateway/bot", nil, nil, &resp)

	mg.Logger.Info().
		Int32("maxConcurrency", resp.SessionStartLimit.MaxConcurrency).
		Int32("shards", resp.Shards).
		Int32("remaining", resp.SessionStartLimit.Remaining).
		Msg("Received gateway")

	return
}

// Scale handles the creation of new ShardGroups with a specified shard count and IDs.
func (mg *Manager) Scale(shardIDs []int32, shardCount int32) (sg *ShardGroup) {
	shardGroupID := mg.shardGroupCounter.Add(1)
	sg = mg.NewShardGroup(shardGroupID, shardIDs, shardCount)

	mg.ShardGroups.Store(shardGroupID, sg)

	return sg
}

// AllReady, returns whether all shard groups are ready
func (mg *Manager) AllReady() bool {
	var isReady bool = true
	mg.ShardGroups.Range(func(shardGroupID int32, sg *ShardGroup) bool {
		if !sg.allShardsReady.Load() {
			isReady = false
			return true // Stop iteration
		}

		return false
	})

	return isReady
}

// ConsumerShardCount returns the number of shards from a consumer view
//
// If virtual shards is disabled, this will return the actual shard count.
// If virtual shards is enabled, this will return the virtual shard count.
func (mg *Manager) ConsumerShardCount() int32 {
	if mg.Configuration.VirtualShards.Enabled {
		return mg.Configuration.VirtualShards.Count
	}

	return mg.noShards
}

// GetShardIdOfGuild returns the shard id of a guild
func (mg *Manager) GetShardIdOfGuild(guildID discord.Snowflake, shardCount int32) int32 {
	return int32((int64(guildID) >> 22) % int64(shardCount))
}

// RoutePayloadToConsumer routes a SandwichPayload to its corresponding consumer modifying the payload itself
func (mg *Manager) RoutePayloadToConsumer(payload *sandwich_structs.SandwichPayload) error {
	if !mg.Configuration.VirtualShards.Enabled {
		// No need to remap, return
		return nil
	}

	if mg.Configuration.VirtualShards.Count == 0 {
		return fmt.Errorf("virtual shards are enabled but count is 0")
	}

	if payload.EventDispatchIdentifier == nil {
		return fmt.Errorf("eventDispatchIdentifier is nil and cannot be remapped")
	}

	if payload.EventDispatchIdentifier.GloballyRouted {
		// Remap shard to empty
		payload.Metadata.Shard = [3]int32{}
	} else if payload.EventDispatchIdentifier.GuildID != nil && *payload.EventDispatchIdentifier.GuildID != 0 {
		virtualShardId := mg.GetShardIdOfGuild(*payload.EventDispatchIdentifier.GuildID, mg.Configuration.VirtualShards.Count)
		payload.Metadata.Shard = [3]int32{0, virtualShardId, mg.Configuration.VirtualShards.Count}
	} else {
		// Not globally routed + no guild id means it's a DM
		payload.Metadata.Shard = [3]int32{0, mg.Configuration.VirtualShards.DmShard, mg.Configuration.VirtualShards.Count}
	}

	return nil
}

// PublishEvent sends an event to consumers.
func (mg *Manager) PublishEvent(ctx context.Context, eventType string, eventData json.RawMessage) error {
	mg.configurationMu.RLock()
	channelName := mg.Configuration.Messaging.ChannelName
	mg.configurationMu.RUnlock()

	err := mg.ProducerClient.Publish(
		ctx,
		&sandwich_structs.SandwichPayload{
			Type:     eventType,
			Data:     eventData,
			Op:       discord.GatewayOpDispatch,
			Metadata: mg.metadata,
		},
		channelName,
	)

	if err != nil {
		return fmt.Errorf("publishEvent publish: %w", err)
	}

	return nil
}

// WaitForIdentify blocks until a shard can identify.
func (mg *Manager) WaitForIdentify(shardID int32, shardCount int32) error {
	mg.Sandwich.configurationMu.RLock()
	identifyURL := mg.Sandwich.Configuration.Identify.URL
	identifyHeaders := mg.Sandwich.Configuration.Identify.Headers
	token := mg.Configuration.Token
	mg.Sandwich.configurationMu.RUnlock()

	mg.gatewayMu.RLock()
	maxConcurrency := mg.Gateway.SessionStartLimit.MaxConcurrency
	mg.gatewayMu.RUnlock()

	hash, err := quickHash(sha256.New(), token)
	if err != nil {
		return err
	}

	if identifyURL == "" {
		identifyBucketName := fmt.Sprintf(
			"identify:%s:%d",
			hash,
			shardID%mg.Gateway.SessionStartLimit.MaxConcurrency,
		)

		mg.Sandwich.IdentifyBuckets.CreateBucket(
			identifyBucketName, 1, IdentifyRateLimit,
		)

		_ = mg.Sandwich.IdentifyBuckets.WaitForBucket(identifyBucketName)
	} else {
		// Pass arguments to URL.
		sendURL := strings.ReplaceAll(identifyURL, "{shard_id}", strconv.Itoa(int(shardID)))
		sendURL = strings.ReplaceAll(sendURL, "{shard_count}", strconv.Itoa(int(shardCount)))
		sendURL = strings.ReplaceAll(sendURL, "{token}", token)
		sendURL = strings.ReplaceAll(sendURL, "{token_hash}", hash)
		sendURL = strings.ReplaceAll(sendURL, "{max_concurrency}", strconv.Itoa(int(maxConcurrency)))

		_, sendURLErr := url.Parse(sendURL)
		if sendURLErr != nil {
			return fmt.Errorf("failed to create valid identify URL: %w", err)
		}

		identifyPayload := sandwich_structs.IdentifyPayload{
			ShardID:        shardID,
			ShardCount:     shardCount,
			Token:          token,
			TokenHash:      hash,
			MaxConcurrency: maxConcurrency,
		}

		identifyPayloadBytes, err := sandwichjson.Marshal(identifyPayload)
		if err != nil {
			return fmt.Errorf("failed to encode identify payload: %w", err)
		}

		client := http.DefaultClient

		for {
			req, err := http.NewRequestWithContext(mg.ctx, http.MethodPost, sendURL, bytes.NewReader(identifyPayloadBytes))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			for k, v := range identifyHeaders {
				req.Header.Set(k, v)
			}

			req.Header.Set("Content-Type", "application/json")

			res, err := client.Do(req)
			if err != nil {
				mg.Logger.Warn().Err(err).Msg("Encountered error whilst identifying")
				time.Sleep(IdentifyRetry)

				continue
			}

			var identifyResponse sandwich_structs.IdentifyResponse
			err = sandwichjson.UnmarshalReader(res.Body, &identifyResponse)

			if err != nil {
				mg.Logger.Warn().Err(err).Msg("Failed to decode identify response")
				time.Sleep(IdentifyRetry)

				continue
			}

			res.Body.Close()

			if identifyResponse.Success {
				break
			}

			waitDuration := time.Millisecond * time.Duration(identifyResponse.Wait)

			mg.Logger.Info().Dur("wait", waitDuration).Msg("Received wait on identify")

			time.Sleep(waitDuration)
		}
	}

	return nil
}

func (mg *Manager) Close() {
	mg.Logger.Info().Msg("Closing manager shardgroups")

	mg.ShardGroups.Range(func(key int32, sg *ShardGroup) bool {
		sg.Close()
		return false
	})
}

// getInitialShardCount returns the initial shard count and ids to use.
func (mg *Manager) getInitialShardCount(customShardCount int32, customShardIDs string, autoSharded bool) (shardIDs []int32, shardCount int32) {
	if autoSharded {
		shardCount = mg.Gateway.Shards

		if customShardIDs == "" {
			for i := int32(0); i < shardCount; i++ {
				shardIDs = append(shardIDs, i)
			}
		} else {
			shardIDs = returnRangeInt32(customShardIDs, shardCount)
		}
	} else {
		shardCount = customShardCount
		shardIDs = returnRangeInt32(customShardIDs, shardCount)
	}

	mg.noShards = shardCount

	return
}
