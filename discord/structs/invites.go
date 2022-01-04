package discord

// invites.go contains all structures for invites.

// InviteTargetType represents the type of an invites target.
type InviteTargetType uint8

const (
	InviteTargetTypeStream InviteTargetType = 1 + iota
	InviteTargetTypeEmbeddedApplication
)

// Invite represents the structure of Invite data.
type Invite struct {
	ChannelID         Snowflake         `json:"channel_id"`
	Code              string            `json:"code"`
	CreatedAt         string            `json:"created_at"`
	GuildID           *Snowflake        `json:"guild_id,omitempty"`
	Inviter           *User             `json:"inviter,omitempty"`
	MaxAge            int               `json:"max_age"`
	MaxUses           int               `json:"max_uses"`
	TargetType        *InviteTargetType `json:"target_type,omitempty"`
	TargetUser        *User             `json:"target_user,omitempty"`
	TargetApplication *Application      `json:"target_application"`
	Temporary         bool              `json:"temporary"`
	Uses              int               `json:"uses"`
}
