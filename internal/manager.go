package internal

import (
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
)

const (
	ShardMaxRetries           = 5
	ShardCompression          = true
	ShardLargeThreshold       = 100
	ShardMaxHeartbeatFailures = 5
)

// ManagerConfiguration represents the configuration for the manager.
type ManagerConfiguration struct {
	// Unique name that will be referenced internally
	Identifier string
	// Non-unique name that is sent to consumers.
	ProducerIdentifier string

	FriendlyName string

	Token     string
	AutoStart bool

	// Bot specific configuration
	Bot struct {
		DefaultPresence *discord.Activity
		Intents         int64
	}

	Caching struct {
		CacheUsers           bool
		CacheMembers         bool
		ChunkGuildsOnStartup bool
		StoreMutuals         bool
	}

	Events struct {
		EventBlacklist   []string
		ProduceBlacklist []string
	}

	Messaging struct {
		ClientName      string
		ChannelName     string
		UseRandomSuffix bool
	}

	Sharding struct {
		AutoSharded bool
		ShardCount  int
	}
}
