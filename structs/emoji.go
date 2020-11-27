package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// Emoji represents an Emoji on discord.
type Emoji struct {
	ID            snowflake.ID   `json:"id" msgpack:"id"`
	Name          string         `json:"name" msgpack:"name"`
	Roles         []snowflake.ID `json:"roles,omitempty" msgpack:"roles,omitempty"`
	User          *User          `json:"user,omitempty" msgpack:"user,omitempty"`
	RequireColons bool           `json:"require_colons,omitempty" msgpack:"require_colons,omitempty"`
	Managed       bool           `json:"managed,omitempty" msgpack:"managed,omitempty"`
	Animated      bool           `json:"animated,omitempty" msgpack:"animated,omitempty"`
	Available     bool           `json:"available,omitempty" msgpack:"available,omitempty"`
}
