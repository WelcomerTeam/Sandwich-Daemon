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

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
)

const (
	ShardMaxRetries              = 5
	ShardCompression             = true
	ShardLargeThreshold          = 100
	ShardMaxHeartbeatFailures    = 5
	MessagingMaxClientNameNumber = 9999

	StandardIdentifyLimit = 5
	IdentifyRetry         = (StandardIdentifyLimit * time.Second)
	IdentifyRateLimit     = (StandardIdentifyLimit * time.Second) + (500 * time.Millisecond)
)

// Manager represents a single application.
type Manager struct {
	ctx    context.Context
	cancel func()

	Error *atomic.String `json:"error" yaml:"error"`

	Identifier *atomic.String `json:"-"`

	Sandwich *Sandwich      `json:"-"`
	Logger   zerolog.Logger `json:"-"`

	configurationMu sync.RWMutex
	Configuration   ManagerConfiguration `json:"configuration" yaml:"configuration"`

	gatewayMu sync.RWMutex
	Gateway   discord.GatewayBotResponse `json:"gateway" yaml:"gateway"`

	shardGroupsMu sync.RWMutex
	ShardGroups   map[int32]*ShardGroup `json:"shard_groups" yaml:"shard_groups"`

	ProducerClient MQClient `json:"-"`

	clientMu sync.Mutex
	Client   *Client `json:"-"`

	UserID *atomic.Int64 `json:"id"`

	userMu sync.RWMutex
	User   discord.User `json:"user"`

	shardGroupCounter *atomic.Int32

	eventBlacklistMu sync.RWMutex
	eventBlacklist   []string

	produceBlacklistMu sync.RWMutex
	produceBlacklist   []string
}

// ManagerConfiguration represents the configuration for the manager.
type ManagerConfiguration struct {
	// Unique name that will be referenced internally
	Identifier string `json:"identifier" yaml:"identifier"`
	// Non-unique name that is sent to consumers.
	ProducerIdentifier string `json:"producer_identifier" yaml:"producer_identifier"`

	FriendlyName string `json:"friendly_name" yaml:"friendly_name"`

	Token     string `json:"token" yaml:"token"`
	AutoStart bool   `json:"auto_start" yaml:"auto_start"`

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

	Events struct {
		EventBlacklist   []string `json:"event_blacklist" yaml:"event_blacklist"`
		ProduceBlacklist []string `json:"produce_blacklist" yaml:"produce_blacklist"`
	} `json:"events" yaml:"events"`

	Messaging struct {
		ClientName      string `json:"client_name" yaml:"client_name"`
		ChannelName     string `json:"channel_name" yaml:"channel_name"`
		UseRandomSuffix bool   `json:"use_random_suffix" yaml:"use_random_suffix"`
	} `json:"messaging" yaml:"messaging"`

	Sharding struct {
		AutoSharded bool   `json:"auto_sharded" yaml:"auto_sharded"`
		ShardCount  int32  `json:"shard_count" yaml:"shard_count"`
		ShardIDs    string `json:"shard_ids" yaml:"shard_ids"`
	} `json:"sharding" yaml:"sharding"`
}

// NewManager creates a new manager.
func (sg *Sandwich) NewManager(configuration ManagerConfiguration) (mg *Manager) {
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

		shardGroupsMu: sync.RWMutex{},
		ShardGroups:   make(map[int32]*ShardGroup),

		Client: NewClient(baseURL, configuration.Token),

		UserID: &atomic.Int64{},

		userMu: sync.RWMutex{},
		User:   discord.User{},

		shardGroupCounter: &atomic.Int32{},

		eventBlacklistMu: sync.RWMutex{},
		eventBlacklist:   configuration.Events.EventBlacklist,

		produceBlacklistMu: sync.RWMutex{},
		produceBlacklist:   configuration.Events.ProduceBlacklist,
	}

	mg.ctx, mg.cancel = context.WithCancel(sg.ctx)

	return mg
}

// Initialize handles the start up process including connecting the message queue client.
func (mg *Manager) Initialize() error {
	gateway, err := mg.GetGateway()
	if err != nil {
		return err
	}

	producerClient, err := NewMQClient(mg.Sandwich.Configuration.Producer.Type)
	if err != nil {
		return err
	}

	clientName := mg.Configuration.Messaging.ClientName
	if mg.Configuration.Messaging.UseRandomSuffix {
		clientName = clientName + "-" + randomHex(6)
	}

	err = producerClient.Connect(
		mg.ctx,
		clientName,
		mg.Sandwich.Configuration.Producer.Configuration,
	)
	if err != nil {
		mg.Logger.Error().Err(err).Msg("Failed to connect producer client")

		return fmt.Errorf("failed to connect to producer: %w", err)
	}

	mg.gatewayMu.Lock()
	mg.Gateway = gateway
	mg.gatewayMu.Unlock()

	mg.ProducerClient = producerClient

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

	mg.shardGroupsMu.Lock()
	mg.ShardGroups[shardGroupID] = sg
	mg.shardGroupsMu.Unlock()

	return sg
}

// PublishEvent sends an event to consumers.
func (mg *Manager) PublishEvent(ctx context.Context, eventType string, eventData json.RawMessage) error {
	packet, _ := mg.Sandwich.payloadPool.Get().(*sandwich_structs.SandwichPayload)
	defer mg.Sandwich.payloadPool.Put(packet)

	mg.configurationMu.RLock()
	identifier := mg.Configuration.ProducerIdentifier
	channelName := mg.Configuration.Messaging.ChannelName
	mg.configurationMu.RUnlock()

	packet.Type = eventType
	packet.Op = discord.GatewayOpDispatch
	packet.Data = eventData

	packet.Metadata = sandwich_structs.SandwichMetadata{
		Version:       VERSION,
		Identifier:    identifier,
		Application:   mg.Identifier.Load(),
		ApplicationID: discord.Snowflake(mg.UserID.Load()),
	}

	// Clear currently unused values
	packet.Sequence = 0
	packet.Extra = nil
	packet.Trace = nil

	payload, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = mg.ProducerClient.Publish(
		ctx,
		channelName,
		payload,
	)
	if err != nil {
		return fmt.Errorf("publishEvent publish: %w", err)
	}

	return nil
}

// WaitForIdentify blocks until a shard can identify.
func (mg *Manager) WaitForIdentify(shardID, shardCount int32) error {
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

		err = mg.Sandwich.IdentifyBuckets.WaitForBucket(identifyBucketName)
		if err != nil {
			return fmt.Errorf("failed to wait for identify bucket: %w", err)
		}
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

		var body bytes.Buffer

		var identifyResponse sandwich_structs.IdentifyResponse

		identifyPayload := sandwich_structs.IdentifyPayload{
			ShardID:        shardID,
			ShardCount:     shardCount,
			Token:          token,
			TokenHash:      hash,
			MaxConcurrency: maxConcurrency,
		}

		err = json.NewEncoder(&body).Encode(identifyPayload)
		if err != nil {
			return fmt.Errorf("failed to encode identify payload: %w", err)
		}

		client := http.DefaultClient

		for {
			req, err := http.NewRequestWithContext(mg.ctx, http.MethodPost, sendURL, bytes.NewBuffer(body.Bytes()))
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

			err = json.NewDecoder(res.Body).Decode(&identifyResponse)
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

	mg.shardGroupsMu.RLock()
	for _, sg := range mg.ShardGroups {
		sg.Close()
	}
	mg.shardGroupsMu.RUnlock()
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

	return
}
