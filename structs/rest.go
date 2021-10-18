package structs

import discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"

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
	ShardGroupID int `json:"id"`

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
