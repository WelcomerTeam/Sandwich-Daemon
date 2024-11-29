package discord

type ReactionType int

const (
	ReactionTypeNormal ReactionType = iota
	ReactionTypeBurst  ReactionType = iota
)

// emoji.go contains all structures for emojis.

// Emoji represents an Emoji on discord.
type Emoji struct {
	GuildID       *GuildID   `json:"guild_id,omitempty"`
	User          *User      `json:"user,omitempty"`
	Name          string     `json:"name"`
	Roles         RoleIDList `json:"roles,omitempty"`
	ID            EmojiID    `json:"id"`
	RequireColons bool       `json:"require_colons"`
	Managed       bool       `json:"managed"`
	Animated      bool       `json:"animated"`
	Available     bool       `json:"available"`
}
