package structs

import (
	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
)

type BaseRestResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
	Ok    bool        `json:"ok"`
}

type StatusEndpointResponse struct {
	Managers []StatusEndpointManager `json:"managers"`
	Uptime   int                     `json:"uptime"`
}

type StatusEndpointManager struct {
	DisplayName string                     `json:"display_name"`
	ShardGroups []StatusEndpointShardGroup `json:"shard_groups"`
}

type StatusEndpointShardGroup struct {

	// ShardID, Status, Latency (in milliseconds), Guilds, Uptime (in seconds), Total Uptime (in seconds)
	Shards [][6]int `json:"shards"`

	Uptime       int   `json:"uptime"`
	ShardGroupID int32 `json:"id"`

	Status ShardGroupStatus `json:"status"`
}

type UserResponse struct {
	User            discord.User `json:"user"`
	IsLoggedIn      bool         `json:"logged_in"`
	IsAuthenticated bool         `json:"authenticated"`
}

type DashboardGetResponse struct {
	Configuration interface{} `json:"configuration"` // Avoids circular references
}

type CreateManagerShardGroupArguments struct {
	ShardIDs    string `json:"shard_ids"`
	Identifier  string `json:"identifier"`
	ShardCount  int32  `json:"shard_count"`
	AutoSharded bool   `json:"auto_sharded"`
}

type SandwichConsumerConfiguration struct {
	Identifiers map[string]ManagerConsumerConfiguration `json:"identifiers"`
	Version     string                                  `json:"v"`
}

type ManagerConsumerConfiguration struct {
	Token string            `json:"token"`
	User  discord.User      `json:"user"`
	ID    discord.Snowflake `json:"id"`
}

type CreateManagerArguments struct {
	Identifier         string `json:"identifier"`
	ProducerIdentifier string `json:"producer_identifier"`
	FriendlyName       string `json:"friendly_name"`
	Token              string `json:"token"`
	ClientName         string `json:"client_name"`
	ChannelName        string `json:"channel_name"`
}
