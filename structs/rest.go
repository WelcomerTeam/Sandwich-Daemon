package structs

import discord "github.com/WelcomerTeam/Discord/structs"

type BaseRestResponse struct {
	Ok    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type StatusEndpointResponse struct {
	Managers []*StatusEndpointManager `json:"managers"`
}

type StatusEndpointManager struct {
	DisplayName string                      `json:"display_name"`
	ShardGroups []*StatusEndpointShardGroup `json:"shard_groups"`
}

type StatusEndpointShardGroup struct {
	ShardGroupID int32 `json:"id"`

	// ShardID, Status, Latency (in milliseconds), Guilds, Uptime (in milliseconds)
	Shards [][5]int `json:"shards"`

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
	ShardCount  int32  `json:"shard_count"`
	AutoSharded bool   `json:"auto_sharded"`
	Identifier  string `json:"identifier"`
}

type SandwichConsumerConfiguration struct {
	Version     string                                  `json:"v"`
	Identifiers map[string]ManagerConsumerConfiguration `json:"identifiers"`
}

type ManagerConsumerConfiguration struct {
	Token string            `json:"token"`
	ID    discord.Snowflake `json:"id"`
	User  discord.User      `json:"user"`
}

type CreateManagerArguments struct {
	Identifier         string `json:"identifier"`
	ProducerIdentifier string `json:"producer_identifier"`
	FriendlyName       string `json:"friendly_name"`
	Token              string `json:"token"`
	ClientName         string `json:"client_name"`
	ChannelName        string `json:"channel_name"`
}
