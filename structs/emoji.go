package structs

import "github.com/bwmarrin/snowflake"

// Emoji represents an Emoji on discord
type Emoji struct {
	ID            snowflake.ID   `json:"id"`
	Name          string         `json:"name"`
	Roles         []snowflake.ID `json:"roles,omitempty"`
	User          *User          `json:"user,omitempty"`
	RequireColons bool           `json:"require_colons,omitempty"`
	Managed       bool           `json:"managed,omitempty"`
	Animated      bool           `json:"animated,omitempty"`
	Available     bool           `json:"available,omitempty"`
}
