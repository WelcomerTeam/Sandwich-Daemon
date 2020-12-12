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
