package discord

import "github.com/WelcomerTeam/RealRock/snowflake"

// guild.go contains the structures to represent a guild.

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
type MFALevel uint8

// MFA levels.
const (
	MFALevelNone MFALevel = iota
	MFALevelElevated
)

// VerificationLevel represents a guild's verification level.
type VerificationLevel uint8

const (
	VerificationLevelNone VerificationLevel = iota
	VerificationLevelLow
	VerificationLevelMedium
	VerificationLevelHigh
	VerificationLevelVeryHigh
)

// SystemChannelFlags represents the flags of a system channel.
type SystemChannelFlags uint8

const (
	SystemChannelFlagsSuppressJoin SystemChannelFlags = 1 << iota
	SystemChannelFlagsPremiumSubscriptions
)

// PremiumTier represents the current boosting tier of a guild.
type PremiumTier uint8

const (
	PremiumTierNone PremiumTier = iota
	PremiumTier1
	PremiumTier2
	PremiumTier3
)

// GuildNSFWLevelType represents the level of the guild.
type GuildNSFWLevelType uint8

const (
	GuildNSFWLevelTypDefault GuildNSFWLevelType = iota
	GuildNSFWLevelTypeExplicit
	GuildNSFWLevelTypeSafe
	GuildNSFWLevelTypeAgeRestricted
)

// Guild represents a guild on Discord.
type Guild struct {
	ID              snowflake.ID `json:"id"`
	Name            string       `json:"name"`
	Icon            string       `json:"icon"`
	IconHash        *string      `json:"icon_hash,omitempty"`
	Splash          string       `json:"splash"`
	DiscoverySplash string       `json:"discovery_splash"`

	Owner       *bool         `json:"owner,omitempty"`
	OwnerID     *snowflake.ID `json:"owner_id,omitempty"`
	Permissions *int          `json:"permissions,omitempty"`
	Region      string        `json:"region"`

	AFKChannelID *snowflake.ID `json:"afk_channel_id,omitempty"`
	AFKTimeout   int           `json:"afk_timeout"`

	WidgetEnabled   *bool         `json:"widget_enabled,omitempty"`
	WidgetChannelID *snowflake.ID `json:"widget_channel_id,omitempty"`

	VerificationLevel           VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       ExplicitContentFilterLevel `json:"explicit_content_filter"`

	Roles    []*Role  `json:"roles"`
	Emojis   []*Emoji `json:"emojis"`
	Features []string `json:"features"`

	MFALevel           MFALevel            `json:"mfa_level"`
	ApplicationID      *snowflake.ID       `json:"application_id,omitempty"`
	SystemChannelID    *snowflake.ID       `json:"system_channel_id,omitempty"`
	SystemChannelFlags *SystemChannelFlags `json:"system_channel_flags,omitempty"`
	RulesChannelID     *snowflake.ID       `json:"rules_channel_id,omitempty"`

	JoinedAt    *string `json:"joined_at,omitempty"`
	Large       *bool   `json:"large,omitempty"`
	Unavailable *bool   `json:"unavailable,omitempty"`
	MemberCount *int    `json:"member_count,omitempty"`

	VoiceStates []*VoiceState  `json:"voice_states,omitempty"`
	Members     []*GuildMember `json:"members,omitempty"`
	Channels    []*Channel     `json:"channels,omitempty"`
	Presences   []*Activity    `json:"presences,omitempty"`

	MaxPresences  *int         `json:"max_presences,omitempty"`
	MaxMembers    *int         `json:"max_members,omitempty"`
	VanityURLCode *string      `json:"vanity_url_code,omitempty"`
	Description   *string      `json:"description,omitempty"`
	Banner        *string      `json:"banner,omitempty"`
	PremiumTier   *PremiumTier `json:"premium_tier,omitempty"`

	PremiumSubscriptionCount *int          `json:"premium_subscription_count,omitempty"`
	PreferredLocale          *string       `json:"preferred_locale,omitempty"`
	PublicUpdatesChannelID   *snowflake.ID `json:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers     *int          `json:"max_video_channel_users,omitempty"`
	ApproximateMemberCount   *int          `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount *int          `json:"approximate_presence_count,omitempty"`

	NSFWLevel      *GuildNSFWLevelType `json:"nsfw_level"`
	StageInstances []*StageInstance    `json:"stage_instances,omitempty"`
	Stickers       []*Sticker          `json:"stickers"`
}

// UnavailableGuild represents an unavailable guild.
type UnavailableGuild struct {
	ID          snowflake.ID `json:"id"`
	Unavailable bool         `json:"unavailable"`
}

// GuildMember represents a guild member on Discord.
type GuildMember struct {
	User *User   `json:"user"`
	Nick *string `json:"nick,omitempty"`

	Roles    []snowflake.ID `json:"roles"`
	JoinedAt string         `json:"joined_at"`
	Deaf     bool           `json:"deaf"`
	Mute     bool           `json:"mute"`
}

// VoiceState represents the voice state on Discord.
type VoiceState struct {
	GuildID   *snowflake.ID `json:"guild_id,omitempty"`
	ChannelID snowflake.ID  `json:"channel_id"`
	UserID    snowflake.ID  `json:"user_id"`
	Member    *GuildMember  `json:"member,omitempty"`
	SessionID string        `json:"session_id"`
	Deaf      bool          `json:"deaf"`
	Mute      bool          `json:"mute"`
	SelfDeaf  bool          `json:"self_deaf"`
	SelfMute  bool          `json:"self_mute"`
	Suppress  bool          `json:"suppress"`
}
