package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"github.com/google/brotli/go/cbrotli"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"golang.org/x/xerrors"
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
	Configuration   *ManagerConfiguration `json:"configuration" yaml:"configuration"`

	gatewayMu sync.RWMutex
	Gateway   discord.GatewayBot `json:"gateway" yaml:"gateway"`

	shardGroupsMu sync.RWMutex
	ShardGroups   map[int64]*ShardGroup `json:"shard_groups" yaml:"shard_groups"`

	ProducerClient MQClient `json:"-"`

	Client *Client `json:"-"`

	shardGroupCounter *atomic.Int64

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
		DefaultPresence      *discord.UpdateStatus `json:"default_presence" yaml:"default_presence"`
		Intents              int64                 `json:"intents" yaml:"intents"`
		ChunkGuildsOnStartup bool                  `json:"chunk_guilds_on_startup" yaml:"chunk_guilds_on_startup"`
	} `json:"bot" yaml:"bot"`

	Caching struct {
		CacheUsers   bool `json:"cache_users" yaml:"cache_users"`
		CacheMembers bool `json:"cache_members" yaml:"cache_members"`
		StoreMutuals bool `json:"store_mutuals" yaml:"store_mutuals"`
	} `json:"caching" yaml:"caching"`

	Events struct {
		EventBlacklist   []string `json:"event_blacklist" yaml:"event_blacklist"`
		ProduceBlacklist []string `json:"produce_blacklist" yaml:"produce_blacklist"`
	} `json:"events" yaml:"events"`

	Messaging struct {
		ClientName      string `json:"client_name" yaml:"client_name"`
		ChannelName     string `json:"channel_name" yaml:"channel_name"`
		UseRandomSuffix bool   `json:"use_random_prefix" yaml:"use_random_prefix"`
	} `json:"messaging" yaml:"messaging"`

	Sharding struct {
		AutoSharded bool   `json:"auto_sharded" yaml:"auto_sharded"`
		ShardCount  int    `json:"shard_count" yaml:"shard_count"`
		ShardIDs    string `json:"shard_ids" yaml:"shard_ids"`
	} `json:"sharding" yaml:"sharding"`
}

// NewManager creates a new manager.
func (sg *Sandwich) NewManager(configuration *ManagerConfiguration) (mg *Manager, err error) {
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
		Gateway:   discord.GatewayBot{},

		shardGroupsMu: sync.RWMutex{},
		ShardGroups:   make(map[int64]*ShardGroup),

		Client: NewClient(configuration.Token),

		shardGroupCounter: &atomic.Int64{},

		eventBlacklistMu: sync.RWMutex{},
		eventBlacklist:   configuration.Events.EventBlacklist,

		produceBlacklistMu: sync.RWMutex{},
		produceBlacklist:   configuration.Events.ProduceBlacklist,
	}

	mg.ctx, mg.cancel = context.WithCancel(sg.ctx)

	return mg, nil
}

// Initialize handles the start up process including connecting the message queue client.
func (mg *Manager) Initialize() (err error) {
	mg.Gateway, err = mg.GetGateway()
	if err != nil {
		return err
	}

	mg.ProducerClient, err = NewMQClient(mg.Sandwich.Configuration.Producer.Type)
	if err != nil {
		return err
	}

	clientName := mg.Configuration.Messaging.ClientName
	if mg.Configuration.Messaging.UseRandomSuffix {
		clientName = clientName + "-" + strconv.Itoa(rand.Intn(MessagingMaxClientNameNumber))
	}

	err = mg.ProducerClient.Connect(
		mg.ctx,
		clientName,
		mg.Sandwich.Configuration.Producer.Configuration,
	)

	return
}

// Open handles retrieving shard counts and scaling.
func (mg *Manager) Open() (err error) {
	shardIDs, shardCount := mg.getInitialShardCount()

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
func (mg *Manager) GetGateway() (resp discord.GatewayBot, err error) {
	mg.Sandwich.gatewayLimiter.Lock()
	_, err = mg.Client.FetchJSON(mg.ctx, "GET", "/gateway/bot", nil, nil, &resp)

	mg.Logger.Info().
		Int("maxConcurrency", resp.SessionStartLimit.MaxConcurrency).
		Int("shards", resp.Shards).
		Int("remaining", resp.SessionStartLimit.Remaining).
		Msg("Received Gateway")

	return
}

// Scale handles the creation of new ShardGroups with a specified shard count and IDs.
func (mg *Manager) Scale(shardIDs []int, shardCount int) (sg *ShardGroup) {
	shardGroupID := mg.shardGroupCounter.Add(1)
	sg = mg.NewShardGroup(shardGroupID, shardIDs, shardCount)

	mg.shardGroupsMu.Lock()
	mg.ShardGroups[shardGroupID] = sg
	mg.shardGroupsMu.Unlock()

	return sg
}

// PublishEvent sends an event to consumers.
func (mg *Manager) PublishEvent(ctx context.Context, eventType string, eventData interface{}) (err error) {
	packet := mg.Sandwich.payloadPool.Get().(*structs.SandwichPayload)
	defer mg.Sandwich.payloadPool.Put(packet)

	mg.configurationMu.RLock()
	identifier := mg.Configuration.ProducerIdentifier
	channelName := mg.Configuration.Messaging.ChannelName
	mg.configurationMu.RUnlock()

	packet.Type = eventType
	packet.Op = discord.GatewayOpDispatch
	packet.Data = eventData

	packet.Metadata = structs.SandwichMetadata{
		Version:    VERSION,
		Identifier: identifier,
	}

	// Clear currently unused values
	packet.Sequence = 0
	packet.Extra = nil
	packet.Trace = nil

	payload, err := json.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("failed to marshal payload: %w", err)
	}

	var compressionOptions cbrotli.WriterOptions

	if len(payload) > minPayloadCompressionSize {
		compressionOptions = mg.Sandwich.DefaultCompressionOptions
	} else {
		compressionOptions = mg.Sandwich.FastCompressionOptions
	}

	result, err := cbrotli.Encode(payload, compressionOptions)
	if err != nil {
		return
	}

	err = mg.ProducerClient.Publish(
		ctx,
		channelName,
		result,
	)

	if err != nil {
		return xerrors.Errorf("publishEvent publish: %w", err)
	}

	return nil
}

// WaitForIdentify blocks until a shard can identify.
func (mg *Manager) WaitForIdentify(shardID int, shardCount int) (err error) {
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

		mg.Sandwich.IdentifyBuckets.WaitForBucket(identifyBucketName)
	} else {
		// Pass arguments to URL
		sendURL := strings.Replace(identifyURL, "{shard_id}", strconv.Itoa(shardID), 0)
		sendURL = strings.Replace(sendURL, "{shard_count}", strconv.Itoa(shardCount), 0)
		sendURL = strings.Replace(sendURL, "{token}", token, 0)
		sendURL = strings.Replace(sendURL, "{token_hash}", hash, 0)
		sendURL = strings.Replace(sendURL, "{max_concurrency}", strconv.Itoa(maxConcurrency), 0)

		_, sendURLErr := url.Parse(sendURL)
		if sendURLErr != nil {
			return nil
		}

		var body bytes.Buffer

		var identifyResponse structs.IdentifyResponse

		identifyPayload := structs.IdentifyPayload{
			ShardID:        shardID,
			ShardCount:     shardCount,
			Token:          token,
			MaxConcurrency: maxConcurrency,
		}

		err = json.NewEncoder(&body).Encode(identifyPayload)
		if err != nil {
			return err
		}

		client := http.DefaultClient

		for {
			req, err := http.NewRequestWithContext(mg.ctx, "POST", sendURL, &body)
			if err != nil {
				return err
			}

			for k, v := range identifyHeaders {
				req.Header.Set(k, v)
			}

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

			time.Sleep(time.Millisecond * time.Duration(identifyResponse.Wait))
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

	if mg.cancel != nil {
		mg.cancel()
	}
}

// getInitialShardCount returns the initial shard count and ids to use.
func (mg *Manager) getInitialShardCount() (shardIDs []int, shardCount int) {
	mg.configurationMu.RLock()
	defer mg.configurationMu.RUnlock()

	if mg.Configuration.Sharding.AutoSharded {
		shardCount = mg.Gateway.Shards

		for i := 0; i < shardCount; i++ {
			shardIDs = append(shardIDs, i)
		}
	} else {
		shardCount = mg.Configuration.Sharding.ShardCount
		shardIDs = returnRange(mg.Configuration.Sharding.ShardIDs, shardCount)
	}

	return
}
