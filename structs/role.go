package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// Role represents a role on Discord
type Role struct {
	ID          snowflake.ID `json:"id"`
	Name        string       `json:"name"`
	Color       int          `json:"color"`
	Hoist       bool         `json:"hoist"`
	Position    int          `json:"position"`
	Permissions int          `json:"permissions"`
	Managed     bool         `json:"managed"`
	Mentionable bool         `json:"mentionable"`
}

// RoleCreate represents a guild role create packet
type RoleCreate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Role    *Role        `json:"role"`
}

// RoleUpdate represents a guild role update packet
type RoleUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Role    *Role        `json:"role"`
}

// RoleDelete represents a guild role delete packet
type RoleDelete struct {
	GuildID snowflake.ID `json:"guild_id"`
	RoleID  snowflake.ID `json:"role_id"`
}
