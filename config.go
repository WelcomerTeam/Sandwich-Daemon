package sandwich

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/WelcomerTeam/Discord/discord"
)

type Configuration struct {
	Sandwich     *DaemonConfiguration        `json:"sandwich"`
	Applications []*ApplicationConfiguration `json:"applications"`
}

type DaemonConfiguration struct {
	// This is used to segment automatically sharded applications.
	NodeCount int32 `json:"node_count"`
	NodeID    int32 `json:"node_id"`
}

type ApplicationConfiguration struct {
	// ApplicationIdentifier used in internal APIs to identify the application.
	ApplicationIdentifier string `json:"application_identifier"`

	// ProducerIdentifier is a reusable identifier that can be used by consumers for routing.
	// This can allow Bot A, Bot B, Bot C to all use the same producer identifier and be handled
	// by the same consumer. The consumer will use the Identifier to determine what token to use.
	ProducerIdentifier string `json:"producer_identifier"`

	// This is the display name of the application. This is included in status APIs.
	DisplayName string `json:"display_name"`

	// This is the client name that is passed to producers.
	ClientName          string `json:"client_name"`
	IncludeRandomSuffix bool   `json:"client_name_uses_random_suffix"`

	BotToken  string `json:"bot_token"`
	AutoStart bool   `json:"auto_start"`

	DefaultPresence    discord.UpdateStatus `json:"default_presence"`
	Intents            int32                `json:"intents"`
	ChunkGuildsOnStart bool                 `json:"chunk_guilds_on_start"`

	// Events that the application should not handle.
	EventBlacklist []string `json:"event_blacklist"`
	// Events that the application should handle, but will not be produced.
	ProduceBlacklist []string `json:"produce_blacklist"`

	AutoSharded bool   `json:"auto_sharded"`
	ShardCount  int32  `json:"shard_count"`
	ShardIDs    string `json:"shard_ids"`

	Values map[string]any `json:"values"`
}

type ConfigProvider interface {
	GetConfig(ctx context.Context) (*Configuration, error)
	SaveConfig(ctx context.Context, config *Configuration) error
}

// ConfigProviderFromPath is a basic config provider that reads and writes to a file.

type ConfigProviderFromPath struct {
	path string
}

func NewConfigProviderFromPath(path string) ConfigProviderFromPath {
	return ConfigProviderFromPath{path}
}

func (c ConfigProviderFromPath) GetConfig(_ context.Context) (*Configuration, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	slog.Debug("Loaded config", "config", config)

	return &config, nil
}

func (c ConfigProviderFromPath) SaveConfig(_ context.Context, config *Configuration) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	slog.Debug("Saving config", "config", config)

	return os.WriteFile(c.path, data, 0o600)
}
