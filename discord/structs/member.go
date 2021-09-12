package discord

// member.go contains all structures that represent a guild member.

// GuildMember represents a guild member on Discord.
type Member struct {
	User         *User       `json:"user"`
	Nick         *string     `json:"nick,omitempty"`
	Roles        []Snowflake `json:"roles"`
	JoinedAt     string      `json:"joined_at"`
	PremiumSince *string     `json:"premium_since,omitempty"`
	Deaf         bool        `json:"deaf"`
	Mute         bool        `json:"mute"`
	Pending      *bool       `json:"pending,omitempty"`
	Permissions  *string     `json:"permissions,omitempty"`
}
