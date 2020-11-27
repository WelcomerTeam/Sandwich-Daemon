package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// Role represents a role on Discord.
type Role struct {
	ID          snowflake.ID `json:"id" msgpack:"id"`
	Color       int          `json:"color" msgpack:"color"`
	Position    int          `json:"position" msgpack:"position"`
	Permissions int          `json:"permissions" msgpack:"permissions"`
	Name        string       `json:"name" msgpack:"name"`
	Managed     bool         `json:"managed" msgpack:"managed"`
	Mentionable bool         `json:"mentionable" msgpack:"mentionable"`
	Hoist       bool         `json:"hoist" msgpack:"hoist"`
}

// RoleCreate represents a guild role create packet.
type RoleCreate struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Role    *Role        `json:"role" msgpack:"role"`
}

// RoleUpdate represents a guild role update packet.
type RoleUpdate struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Role    *Role        `json:"role" msgpack:"role"`
}

// RoleDelete represents a guild role delete packet.
type RoleDelete struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	RoleID  snowflake.ID `json:"role_id" msgpack:"role_id"`
}
