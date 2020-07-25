package structs

import (
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/vmihailenco/msgpack"
)

// Guild represents a guild on Discord
type Guild struct {
	ID                          snowflake.ID               `json:"id"`
	Name                        string                     `json:"name"`
	Icon                        string                     `json:"icon"`
	Splash                      string                     `json:"splash"`
	Owner                       bool                       `json:"owner,omitempty"`
	OwnerID                     snowflake.ID               `json:"owner_id,omitempty"`
	Permissions                 int                        `json:"permissions,omitempty"`
	Region                      string                     `json:"region"`
	AFKChannelID                snowflake.ID               `json:"afk_channel_id,omitempty"`
	AFKTimeout                  int                        `json:"afk_timeout"`
	EmbedEnabled                bool                       `json:"embed_enabled,omitempty"`
	EmbedChannelID              snowflake.ID               `json:"embed_channel_id,omitempty"`
	VerificationLevel           VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       ExplicitContentFilterLevel `json:"explicit_content_filter"`
	Roles                       []*Role                    `json:"roles"`
	Emojis                      []*Emoji                   `json:"emojis"`
	Features                    []string                   `json:"features"`
	MFALevel                    MFALevel                   `json:"mfa_level"`
	ApplicationID               snowflake.ID               `json:"application_id,omitempty"`
	WidgetEnabled               bool                       `json:"widget_enabled,omitempty"`
	WidgetChannelID             snowflake.ID               `json:"widget_channel_id,omitempty"`
	SystemChannelID             snowflake.ID               `json:"system_channel_id,omitempty"`
	JoinedAt                    string                     `json:"joined_at,omitempty"`
	Large                       bool                       `json:"large,omitempty"`
	Unavailable                 bool                       `json:"unavailable,omitempty"`
	MemberCount                 int                        `json:"member_count,omitempty"`
	VoiceStates                 []*VoiceState              `json:"voice_states,omitempty"`
	Members                     []*GuildMember             `json:"members,omitempty"`
	Channels                    []*Channel                 `json:"channels,omitempty"`
	Presences                   []*Activity                `json:"presences,omitempty"`
}

// UnavailableGuild represents an unavailable guild
type UnavailableGuild struct {
	ID          snowflake.ID `json:"id"`
	Unavailable bool         `json:"unavailable"`
}

// MessageNotificationLevel represents a guild's message notification level
type MessageNotificationLevel int

// Message notification levels
const (
	MessageNotificationsAllMessages MessageNotificationLevel = iota
	MessageNotificationsOnlyMentions
)

// ExplicitContentFilterLevel represents a guild's explicit content filter level
type ExplicitContentFilterLevel int

// Explicit content filter levels
const (
	ExplicitContentFilterDisabled ExplicitContentFilterLevel = iota
	ExplicitContentFilterMembersWithoutRoles
	ExplicitContentFilterAllMembers
)

// MFALevel represents a guild's MFA level
type MFALevel int

// MFA levels
const (
	MFALevelNone MFALevel = iota
	MFALevelElevated
)

// VerificationLevel represents a guild's verification level
type VerificationLevel int

// Verification levels
const (
	VerificationLevelNone VerificationLevel = iota
	VerificationLevelLow
	VerificationLevelMedium
	VerificationLevelHigh
	VerificationLevelVeryHigh
)

// GuildCreate represents a guild create packet
type GuildCreate struct {
	Guild
	Lazy bool `json:"-"` // Internal use only
}

// GuildUpdate represents a guild update packet
type GuildUpdate Guild

// GuildDelete represents a guild delete packet
type GuildDelete UnavailableGuild

// GuildBanAdd represents a guild ban add packet
type GuildBanAdd struct {
	GuildID snowflake.ID `json:"guild_id"`
	User    *User        `json:"user"`
}

// GuildBanRemove represents a guild ban remove packet
type GuildBanRemove struct {
	GuildID snowflake.ID `json:"guild_id"`
	User    *User        `json:"user"`
}

// GuildEmojisUpdate represents a guild emojis update packet
type GuildEmojisUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Emojis  []*Emoji     `json:"emojis"` // TODO: type
}

// GuildIntegrationsUpdate represents a guild integrations update packet
type GuildIntegrationsUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
}

// GuildMemberAdd represents a guild member add packet
type GuildMemberAdd struct {
	*GuildMember
	GuildID snowflake.ID `json:"guild_id"`
}

// GuildMemberRemove represents a guild member remove packet
type GuildMemberRemove struct {
	GuildID snowflake.ID `json:"guild_id"`
	User    *User        `json:"user"`
}

// GuildMemberUpdate represents a guild member update packet
type GuildMemberUpdate struct {
	GuildID snowflake.ID   `json:"guild_id"`
	Roles   []snowflake.ID `json:"roles"`
	User    *User          `json:"user"`
	Nick    string         `json:"nick"`
}

// GuildMembersChunk represents a guild members chunk packet
type GuildMembersChunk struct {
	GuildID    snowflake.ID     `json:"guild_id"`
	Members    []*GuildMember   `json:"members"`
	ChunkIndex int              `json:"chunk_index"`
	ChunkCount int              `json:"chunk_count"`
	NotFound   []snowflake.ID   `json:"not_found"`
	Presences  []PresenceStatus `json:"presences"`
	Nonce      string           `json:"nonce"`
}

// GuildRoleCreate represents a guild role create packet
type GuildRoleCreate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Role    interface{}  `json:"role"` // TODO: type
}

// GuildRoleUpdate represents a guild role update packet
type GuildRoleUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Role    *Role        `json:"role"` // TODO: type
}

// GuildRoleDelete represents a guild role delete packet
type GuildRoleDelete struct {
	GuildID snowflake.ID `json:"guild_id"`
	RoleID  snowflake.ID `json:"role_id"`
}

// GuildMember represents a guild member on Discord
type GuildMember struct {
	User     *User          `json:"user"`
	Nick     string         `json:"nick,omitempty"`
	Roles    []snowflake.ID `json:"roles"`
	JoinedAt string         `json:"joined_at"`
	Deaf     bool           `json:"deaf"`
	Mute     bool           `json:"mute"`
}

// MarshalBinary converts the GuildMember into a format usable for redis
func (gm GuildMember) MarshalBinary() ([]byte, error) {
	return msgpack.Marshal(gm)
}

// UnmarshalBinary converts from the redis format into a GuildMember
func (gm *GuildMember) UnmarshalBinary(data []byte) error {
	return msgpack.Unmarshal(data, gm)
}
