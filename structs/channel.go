package structs

// ChannelType represents a channel's type
type ChannelType int

// Channel types
const (
	ChannelTypeGuildText ChannelType = iota
	ChannelTypeDM
	ChannelTypeGuildVoice
	ChannelTypeGroupDM
	ChannelTypeGuildCategory
)

// Channel represents a Discord channel
type Channel struct {
	ID                   string      `json:"id"`
	Type                 ChannelType `json:"type"`
	GuildID              string      `json:"guild_id,omitempty"`
	Position             int         `json:"position,omitempty"`
	PermissionOverwrites []Overwrite `json:"permission_overwrites,omitempty"`
	Name                 string      `json:"name,omitempty"`
	Topic                string      `json:"topic,omitempty"`
	NSFW                 bool        `json:"nsfw,omitempty"`
	LastMessageID        string      `json:"last_message_id,omitempty"`
	Bitrate              int         `json:"bitrate,omitempty"`
	UserLimit            int         `json:"user_limit,omitempty"`
	RateLimitPerUser     int         `json:"rate_limit_per_user,omitempty"`
	Recipients           []User      `json:"recipients,omitempty"`
	Icon                 string      `json:"icon,omitempty"`
	OwnerID              string      `json:"owner_id,omitempty"`
	ApplicationID        string      `json:"application_id,omitempty"`
	ParentID             string      `json:"parent_id,omitempty"`
	LastPinTimestamp     string      `json:"last_pin_timestamp"`
}

// Overwrite represents a permission overwrite
type Overwrite struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Allow int    `json:"allow"`
	Deny  int    `json:"deny"`
}

// ChannelCreate represents a channel create packet
type ChannelCreate struct {
	*Channel
}

// ChannelUpdate represents a channel update packet
type ChannelUpdate struct {
	*Channel
}

// ChannelDelete represents a channel delete packet
type ChannelDelete struct {
	*Channel
}

// ChannelPinsUpdate represents a channel pins update packet
type ChannelPinsUpdate struct {
	ChannelID        string `json:"channel_id"`
	LastPinTimestamp string `json:"last_pin_timestamp,omitempty"`
}
