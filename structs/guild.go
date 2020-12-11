package structs

import (
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/vmihailenco/msgpack"
)

// Guild represents a guild on Discord.
type Guild struct {
	ID                          snowflake.ID               `json:"id" msgpack:"id"`
	OwnerID                     snowflake.ID               `json:"owner_id,omitempty" msgpack:"owner_id,omitempty"`
	AFKChannelID                snowflake.ID               `json:"afk_channel_id,omitempty" msgpack:"afk_channel_id,omitempty"`
	ApplicationID               snowflake.ID               `json:"application_id,omitempty" msgpack:"application_id,omitempty"`
	WidgetChannelID             snowflake.ID               `json:"widget_channel_id,omitempty" msgpack:"widget_channel_id,omitempty"`
	SystemChannelID             snowflake.ID               `json:"system_channel_id,omitempty" msgpack:"system_channel_id,omitempty"`
	Permissions                 int                        `json:"permissions,omitempty" msgpack:"permissions,omitempty"`
	AFKTimeout                  int                        `json:"afk_timeout" msgpack:"afk_timeout"`
	MemberCount                 int                        `json:"member_count,omitempty" msgpack:"member_count,omitempty"`
	VerificationLevel           VerificationLevel          `json:"verification_level" msgpack:"verification_level"`
	DefaultMessageNotifications MessageNotificationLevel   `json:"default_message_notifications" msgpack:"default_message_notifications"`
	ExplicitContentFilter       ExplicitContentFilterLevel `json:"explicit_content_filter" msgpack:"explicit_content_filter"`
	MFALevel                    MFALevel                   `json:"mfa_level" msgpack:"mfa_level"`
	JoinedAt                    string                     `json:"joined_at,omitempty" msgpack:"joined_at,omitempty"`
	Region                      string                     `json:"region" msgpack:"region"`
	Name                        string                     `json:"name" msgpack:"name"`
	Icon                        string                     `json:"icon" msgpack:"icon"`
	Splash                      string                     `json:"splash" msgpack:"splash"`
	Owner                       bool                       `json:"owner,omitempty" msgpack:"owner,omitempty"`
	WidgetEnabled               bool                       `json:"widget_enabled,omitempty" msgpack:"widget_enabled,omitempty"`
	Large                       bool                       `json:"large,omitempty" msgpack:"large,omitempty"`
	Unavailable                 bool                       `json:"unavailable,omitempty" msgpack:"unavailable,omitempty"`
	Features                    []string                   `json:"features" msgpack:"features"`
	Roles                       []*Role                    `json:"roles" msgpack:"roles"`
	Emojis                      []*Emoji                   `json:"emojis" msgpack:"emojis"`
	VoiceStates                 []*VoiceState              `json:"voice_states,omitempty" msgpack:"voice_states,omitempty"`
	Members                     []*GuildMember             `json:"members,omitempty" msgpack:"members,omitempty"`
	Channels                    []*Channel                 `json:"channels,omitempty" msgpack:"channels,omitempty"`
	Presences                   []*Activity                `json:"presences,omitempty" msgpack:"presences,omitempty"`
}

// UnavailableGuild represents an unavailable guild.
type UnavailableGuild struct {
	ID          snowflake.ID `json:"id" msgpack:"id"`
	Unavailable bool         `json:"unavailable" msgpack:"unavailable"`
}

// MessageNotificationLevel represents a guild's message notification level.
type MessageNotificationLevel int

// Message notification levels.
const (
	MessageNotificationsAllMessages MessageNotificationLevel = iota
	MessageNotificationsOnlyMentions
)

// ExplicitContentFilterLevel represents a guild's explicit content filter level.
type ExplicitContentFilterLevel int

// Explicit content filter levels.
const (
	ExplicitContentFilterDisabled ExplicitContentFilterLevel = iota
	ExplicitContentFilterMembersWithoutRoles
	ExplicitContentFilterAllMembers
)

// MFALevel represents a guild's MFA level.
type MFALevel int

// MFA levels.
const (
	MFALevelNone MFALevel = iota
	MFALevelElevated
)

// VerificationLevel represents a guild's verification level.
type VerificationLevel int

// Verification levels.
const (
	VerificationLevelNone VerificationLevel = iota
	VerificationLevelLow
	VerificationLevelMedium
	VerificationLevelHigh
	VerificationLevelVeryHigh
)

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

// GuildMember represents a guild member on Discord.
type GuildMember struct {
	User *User  `json:"user" msgpack:"user"`
	Nick string `json:"nick,omitempty" msgpack:"nick,omitempty"`

	Roles    []snowflake.ID `json:"roles" msgpack:"roles"`
	JoinedAt string         `json:"joined_at" msgpack:"joined_at"`
	Deaf     bool           `json:"deaf" msgpack:"deaf"`
	Mute     bool           `json:"mute" msgpack:"mute"`
}

// MarshalBinary converts the GuildMember into a format usable for redis.
func (gm GuildMember) MarshalBinary() ([]byte, error) {
	return msgpack.Marshal(gm)
}

// UnmarshalBinary converts from the redis format into a GuildMember.
func (gm *GuildMember) UnmarshalBinary(data []byte) error {
	return msgpack.Unmarshal(data, gm)
}
