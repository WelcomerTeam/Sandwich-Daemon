package structs

import (
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/vmihailenco/msgpack"
)

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

// SystemChannelFlags represents the flags of a system channel.
type SystemChannelFlags int

// System channel flags.
const (
	SystemChannelFlagsSuppressJoin SystemChannelFlags = 1 << iota
	SystemChannelFlagsPremiumSubscriptions
)

// PremiumTier represents the current boosting tier of a guild.
type PremiumTier int

// Premium tier.
const (
	PremiumTierNone PremiumTier = iota
	PremiumTier1
	PremiumTier2
	PremiumTier3
)

// Guild represents a guild on Discord.
type Guild struct {
	ID              snowflake.ID `json:"id" msgpack:"id"`
	Name            string       `json:"name" msgpack:"name"`
	Icon            string       `json:"icon" msgpack:"icon"`
	IconHash        string       `json:"icon_hash,omitempty" msgpack:"icon_hash,omitempty"`
	Splash          string       `json:"splash" msgpack:"splash"`
	DiscoverySplash string       `json:"discovery_splash" msgpack:"discovery_splash"`

	Owner       bool         `json:"owner,omitempty" msgpack:"owner,omitempty"`
	OwnerID     snowflake.ID `json:"owner_id,omitempty" msgpack:"owner_id,omitempty"`
	Permissions int          `json:"permissions,omitempty" msgpack:"permissions,omitempty"`
	Region      string       `json:"region" msgpack:"region"`

	AFKChannelID snowflake.ID `json:"afk_channel_id,omitempty" msgpack:"afk_channel_id,omitempty"`
	AFKTimeout   int          `json:"afk_timeout" msgpack:"afk_timeout"`

	WidgetEnabled   bool         `json:"widget_enabled,omitempty" msgpack:"widget_enabled,omitempty"`
	WidgetChannelID snowflake.ID `json:"widget_channel_id,omitempty" msgpack:"widget_channel_id,omitempty"`

	VerificationLevel           VerificationLevel          `json:"verification_level" msgpack:"verification_level"`
	DefaultMessageNotifications MessageNotificationLevel   `json:"default_message_notifications" msgpack:"default_message_notifications"`
	ExplicitContentFilter       ExplicitContentFilterLevel `json:"explicit_content_filter" msgpack:"explicit_content_filter"`

	Roles    []*Role  `json:"roles" msgpack:"roles"`
	Emojis   []*Emoji `json:"emojis" msgpack:"emojis"`
	Features []string `json:"features" msgpack:"features"`

	MFALevel           MFALevel           `json:"mfa_level" msgpack:"mfa_level"`
	ApplicationID      snowflake.ID       `json:"application_id,omitempty" msgpack:"application_id,omitempty"`
	SystemChannelID    snowflake.ID       `json:"system_channel_id,omitempty" msgpack:"system_channel_id,omitempty"`
	SystemChannelFlags SystemChannelFlags `json:"system_channel_flags,omitempty" msgpack:"system_channel_flags,omitempty"`
	RulesChannelID     snowflake.ID       `json:"rules_channel_id,omitempty" msgpack:"rules_channel_id,omitempty"`

	JoinedAt    string `json:"joined_at,omitempty" msgpack:"joined_at,omitempty"`
	Large       bool   `json:"large,omitempty" msgpack:"large,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty" msgpack:"unavailable,omitempty"`
	MemberCount int    `json:"member_count,omitempty" msgpack:"member_count,omitempty"`

	VoiceStates []*VoiceState  `json:"voice_states,omitempty" msgpack:"voice_states,omitempty"`
	Members     []*GuildMember `json:"members,omitempty" msgpack:"members,omitempty"`
	Channels    []*Channel     `json:"channels,omitempty" msgpack:"channels,omitempty"`
	Presences   []*Activity    `json:"presences,omitempty" msgpack:"presences,omitempty"`

	MaxPresences  int         `json:"max_presences,omitempty" msgpack:"max_presences,omitempty"`
	MaxMembers    int         `json:"max_members,omitempty" msgpack:"max_members,omitempty"`
	VanityURLCode string      `json:"vanity_url_code,omitempty" msgpack:"vanity_url_code,omitempty"`
	Description   string      `json:"description,omitempty" msgpack:"description,omitempty"`
	Banner        string      `json:"banner,omitempty" msgpack:"banner,omitempty"`
	PremiumTier   PremiumTier `json:"premium_tier,omitempty" msgpack:"premium_tier,omitempty"`

	PremiumSubscriptionCount int          `json:"premium_subscription_count,omitempty" msgpack:"premium_subscription_count,omitempty"`
	PreferredLocale          string       `json:"preferred_locale,omitempty" msgpack:"preferred_locale,omitempty"`
	PublicUpdatesChannelID   snowflake.ID `json:"public_updates_channel_id,omitempty" msgpack:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers     int          `json:"max_video_channel_users,omitempty" msgpack:"max_video_channel_users,omitempty"`
	ApproximateMemberCount   int          `json:"approximate_member_count,omitempty" msgpack:"approximate_member_count,omitempty"`
	ApproximatePresenceCount int          `json:"approximate_presence_count,omitempty" msgpack:"approximate_presence_count,omitempty"`
}

// UnavailableGuild represents an unavailable guild.
type UnavailableGuild struct {
	ID          snowflake.ID `json:"id" msgpack:"id"`
	Unavailable bool         `json:"unavailable" msgpack:"unavailable"`
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

// VoiceState represents the voice state on Discord.
type VoiceState struct {
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	UserID    snowflake.ID `json:"user_id" msgpack:"user_id"`
	Member    GuildMember  `json:"member,omitempty" msgpack:"member,omitempty"`
	SessionID string       `json:"session_id" msgpack:"session_id"`
	Deaf      bool         `json:"deaf" msgpack:"deaf"`
	Mute      bool         `json:"mute" msgpack:"mute"`
	SelfDeaf  bool         `json:"self_deaf" msgpack:"self_deaf"`
	SelfMute  bool         `json:"self_mute" msgpack:"self_mute"`
	Suppress  bool         `json:"suppress" msgpack:"suppress"`
}

// MarshalBinary converts the GuildMember into a format usable for redis.
func (gm GuildMember) MarshalBinary() ([]byte, error) {
	return msgpack.Marshal(gm)
}

// UnmarshalBinary converts from the redis format into a GuildMember.
func (gm *GuildMember) UnmarshalBinary(data []byte) error {
	return msgpack.Unmarshal(data, gm)
}
