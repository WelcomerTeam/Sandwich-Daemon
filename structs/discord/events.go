package structs

import (
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
)

// Hello represents a hello packet.
type Hello struct {
	HeartbeatInterval time.Duration `json:"heartbeat_interval" msgpack:"heartbeat_interval"`
}

// Ready represents a ready packet.
type Ready struct {
	Version   int      `json:"v" msgpack:"v"`
	User      *User    `json:"user" msgpack:"user"`
	Guilds    []*Guild `json:"guilds" msgpack:"guilds"`
	SessionID string   `json:"session_id" msgpack:"session_id"`
}

// Resume represents a resume packet.
type Resume struct {
	Token     string `json:"token" msgpack:"token"`
	SessionID string `json:"session_id" msgpack:"session_id"`
	Sequence  int64  `json:"seq" msgpack:"seq"`
}

// Identify represents an identify packet.
type Identify struct {
	Intents            int                 `json:"intents,omitempty" msgpack:"intents,omitempty"`
	LargeThreshold     int                 `json:"large_threshold,omitempty" msgpack:"large_threshold,omitempty"`
	Shard              [2]int              `json:"shard,omitempty" msgpack:"shard,omitempty"`
	Token              string              `json:"token" msgpack:"token"`
	Properties         *IdentifyProperties `json:"properties" msgpack:"properties"`
	Presence           *UpdateStatus       `json:"presence,omitempty" msgpack:"presence,omitempty"`
	Compress           bool                `json:"compress,omitempty" msgpack:"compress,omitempty"`
	GuildSubscriptions bool                `json:"guild_subscriptions,omitempty" msgpack:"guild_subscriptions,omitempty"`
}

// IdentifyProperties is the properties sent in the identify packet.
type IdentifyProperties struct {
	OS      string `json:"$os" msgpack:"$os"`
	Browser string `json:"$browser" msgpack:"$browser"`
	Device  string `json:"$device" msgpack:"$device"`
}

// RequestGuildMembers represents a request guild members packet.
type RequestGuildMembers struct {
	GuildID   snowflake.ID   `json:"guild_id" msgpack:"guild_id"`
	Query     string         `json:"query" msgpack:"query"`
	Limit     int            `json:"limit" msgpack:"limit"`
	Presences bool           `json:"presences" msgpack:"presences"`
	Nonce     string         `json:"nonce" msgpack:"nonce"`
	UserIDs   []snowflake.ID `json:"user_ids" msgpack:"user_ids"`
}

// UpdateVoiceState represents an update voice state packet.
type UpdateVoiceState struct {
	GuildID   snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	SelfMute  bool         `json:"self_mute" msgpack:"self_mute"`
	SelfDeaf  bool         `json:"self_deaf" msgpack:"self_deaf"`
}

// UpdateStatus represents an update status packet.
type UpdateStatus struct {
	Since  int       `json:"since,omitempty" msgpack:"since,omitempty"`
	Game   *Activity `json:"game,omitempty" msgpack:"game,omitempty"`
	Status string    `json:"status" msgpack:"status"`
	AFK    bool      `json:"afk" msgpack:"afk"`
}

// ChannelCreate represents a channel create packet.
type ChannelCreate struct {
	*Channel
}

// ChannelUpdate represents a channel update packet.
type ChannelUpdate struct {
	*Channel
}

// ChannelDelete represents a channel delete packet.
type ChannelDelete struct {
	*Channel
}

// ChannelPinsUpdate represents a channel pins update packet.
type ChannelPinsUpdate struct {
	ChannelID        snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	LastPinTimestamp string       `json:"last_pin_timestamp,omitempty" msgpack:"last_pin_timestamp,omitempty"`
}

// MessageCreate represents a message create packet.
type MessageCreate struct {
	*Message
}

// MessageUpdate represents a message update packet.
type MessageUpdate struct {
	*Message
}

// MessageDelete represents a message delete packet.
type MessageDelete struct {
	ID        snowflake.ID `json:"id" msgpack:"id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}

// MessageDeleteBulk represents a message delete bulk packet.
type MessageDeleteBulk struct {
	IDs       []snowflake.ID `json:"ids" msgpack:"ids"`
	ChannelID snowflake.ID   `json:"channel_id" msgpack:"channel_id"`
	GuildID   snowflake.ID   `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}

// MessageReactionAdd represents a message reaction add packet.
type MessageReactionAdd struct {
	UserID    snowflake.ID `json:"user_id" msgpack:"user_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	MessageID snowflake.ID `json:"message_id" msgpack:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Emoji     *Emoji       `json:"emoji" msgpack:"emoji"`
}

// MessageReactionRemove represents a message reaction remove packet.
type MessageReactionRemove struct {
	UserID    snowflake.ID `json:"user_id" msgpack:"user_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	MessageID snowflake.ID `json:"message_id" msgpack:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Emoji     *Emoji       `json:"emoji" msgpack:"emoji"`
}

// MessageReactionRemoveAll represents a message reaction remove all packet.
type MessageReactionRemoveAll struct {
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	MessageID snowflake.ID `json:"message_id" msgpack:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}

// GuildCreate represents a guild create packet.
type GuildCreate struct {
	Guild
	Lazy bool `json:"-" msgpack:"-"` // Internal use only.
}

// GuildUpdate represents a guild update packet.
type GuildUpdate Guild

// GuildDelete represents a guild delete packet.
type GuildDelete UnavailableGuild

// GuildBanAdd represents a guild ban add packet.
type GuildBanAdd struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	User    *User        `json:"user" msgpack:"user"`
}

// GuildBanRemove represents a guild ban remove packet.
type GuildBanRemove struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	User    *User        `json:"user" msgpack:"user"`
}

// GuildEmojisUpdate represents a guild emojis update packet.
type GuildEmojisUpdate struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Emojis  []*Emoji     `json:"emojis" msgpack:"emojis"` // TODO: type
}

// GuildIntegrationsUpdate represents a guild integrations update packet.
type GuildIntegrationsUpdate struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
}

// GuildMemberAdd represents a guild member add packet.
type GuildMemberAdd struct {
	*GuildMember
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
}

// GuildMemberRemove represents a guild member remove packet.
type GuildMemberRemove struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	User    *User        `json:"user" msgpack:"user"`
}

// GuildMemberUpdate represents a guild member update packet.
type GuildMemberUpdate struct {
	GuildID snowflake.ID   `json:"guild_id" msgpack:"guild_id"`
	Roles   []snowflake.ID `json:"roles" msgpack:"roles"`
	User    *User          `json:"user" msgpack:"user"`
	Nick    string         `json:"nick" msgpack:"nick"`
}

// GuildMembersChunk represents a guild members chunk packet.
type GuildMembersChunk struct {
	GuildID    snowflake.ID     `json:"guild_id" msgpack:"guild_id"`
	Members    []*GuildMember   `json:"members" msgpack:"members"`
	ChunkIndex int              `json:"chunk_index" msgpack:"chunk_index"`
	ChunkCount int              `json:"chunk_count" msgpack:"chunk_count"`
	NotFound   []snowflake.ID   `json:"not_found" msgpack:"not_found"`
	Presences  []PresenceStatus `json:"presences" msgpack:"presences"`
	Nonce      string           `json:"nonce" msgpack:"nonce"`
}

// GuildRoleCreate represents a guild role create packet.
type GuildRoleCreate struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Role    interface{}  `json:"role" msgpack:"role"` // TODO: type
}

// GuildRoleUpdate represents a guild role update packet.
type GuildRoleUpdate struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Role    *Role        `json:"role" msgpack:"role"` // TODO: type
}

// GuildRoleDelete represents a guild role delete packet.
type GuildRoleDelete struct {
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	RoleID  snowflake.ID `json:"role_id" msgpack:"role_id"`
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

// VoiceStateUpdate represents the VOICE_STATE_UPDATE packet.
type VoiceStateUpdate VoiceState

// VoiceServerUpdate represents a VOICE_SERVER_UPDATE packet.
type VoiceServerUpdate struct {
	Token    string       `json:"token" msgpack:"token"`
	GuildID  snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Endpoint string       `json:"endpoint" msgpack:"endpoint"`
}

// WebhookUpdate represents a webhook update packet.
type WebhookUpdate struct {
	GuildID   snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
}

// TypingStart represents a typing start packet.
type TypingStart struct {
	ChannelID snowflake.ID `json:"channel_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	UserID    snowflake.ID `json:"user_id"`
	Timestamp int          `json:"timestamp"`
}

// UserUpdate represents a user update packet.
type UserUpdate struct {
	*User
}
