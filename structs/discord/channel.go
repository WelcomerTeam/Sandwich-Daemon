package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// ChannelType represents a channel's type.
type ChannelType int

// Channel types.
const (
	ChannelTypeGuildText ChannelType = iota
	ChannelTypeDM
	ChannelTypeGuildVoice
	ChannelTypeGroupDM
	ChannelTypeGuildCategory
	ChannelTypeGuildNews
	ChannelTypeGuildStore
)

// Channel represents a Discord channel.
type Channel struct {
	ID                   snowflake.ID       `json:"id" msgpack:"id"`
	Type                 ChannelType        `json:"type" msgpack:"type"`
	GuildID              snowflake.ID       `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Position             int                `json:"position,omitempty" msgpack:"position,omitempty"`
	PermissionOverwrites []ChannelOverwrite `json:"permission_overwrites,omitempty" msgpack:"permission_overwrites,omitempty"`
	Name                 string             `json:"name,omitempty" msgpack:"name,omitempty"`
	Topic                string             `json:"topic,omitempty" msgpack:"topic,omitempty"`
	NSFW                 bool               `json:"nsfw,omitempty" msgpack:"nsfw,omitempty"`
	LastMessageID        string             `json:"last_message_id,omitempty" msgpack:"last_message_id,omitempty"`
	Bitrate              int                `json:"bitrate,omitempty" msgpack:"bitrate,omitempty"`
	UserLimit            int                `json:"user_limit,omitempty" msgpack:"user_limit,omitempty"`
	RateLimitPerUser     int                `json:"rate_limit_per_user,omitempty" msgpack:"rate_limit_per_user,omitempty"`
	Recipients           []User             `json:"recipients,omitempty" msgpack:"recipients,omitempty"`
	Icon                 string             `json:"icon,omitempty" msgpack:"icon,omitempty"`
	OwnerID              snowflake.ID       `json:"owner_id,omitempty" msgpack:"owner_id,omitempty"`
	ApplicationID        snowflake.ID       `json:"application_id,omitempty" msgpack:"application_id,omitempty"`
	ParentID             snowflake.ID       `json:"parent_id,omitempty" msgpack:"parent_id,omitempty"`
	LastPinTimestamp     string             `json:"last_pin_timestamp,omitempty" msgpack:"last_pin_timestamp,omitempty"`
}

// ChannelOverwrite represents a permission overwrite for a channel.
type ChannelOverwrite struct {
	ID    string `json:"id" msgpack:"id"`
	Type  string `json:"type" msgpack:"type"`
	Allow int    `json:"allow" msgpack:"allow"`
	Deny  int    `json:"deny" msgpack:"deny"`
}
