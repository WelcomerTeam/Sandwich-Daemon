package structs

import (
	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
	jsoniter "github.com/json-iterator/go"
	"sync"
)

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  jsoniter.RawMessage
	Extra map[string]jsoniter.RawMessage
}

type StateGuild struct {
	WidgetEnabled *bool `json:"widget_enabled"`
	Large         *bool `json:"large"`
	Unavailable   *bool `json:"unavailable"`

	AFKTimeout  int     `json:"afk_timeout"`
	Permissions *string `json:"permissions,omitempty"`

	NSFWLevel *discord.GuildNSFWLevelType `json:"nsfw_level"`
	ID        discord.Snowflake           `json:"id"`

	Name            string `json:"name"`
	Icon            string `json:"icon"`
	Splash          string `json:"splash"`
	DiscoverySplash string `json:"discovery_splash"`
	Region          string `json:"region"`
	Description     string `json:"description"`
	Banner          string `json:"banner"`

	MFALevel                    discord.MFALevel                   `json:"mfa_level"`
	VerificationLevel           discord.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications discord.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       discord.ExplicitContentFilterLevel `json:"explicit_content_filter"`

	MemberCount              *int `json:"member_count"`
	PremiumSubscriptionCount *int `json:"premium_subscription_count"`
	MaxVideoChannelUsers     *int `json:"max_video_channel_users"`
	ApproximateMemberCount   *int `json:"approximate_member_count"`
	ApproximatePresenceCount *int `json:"approximate_presence_count"`
	MaxPresences             *int `json:"max_presences"`
	MaxMembers               *int `json:"max_members"`

	IconHash        *string `json:"icon_hash"`
	JoinedAt        *string `json:"joined_at"`
	VanityURLCode   *string `json:"vanity_url_code"`
	PreferredLocale *string `json:"preferred_locale"`

	PremiumTier        *discord.PremiumTier        `json:"premium_tier"`
	SystemChannelFlags *discord.SystemChannelFlags `json:"system_channel_flags"`

	PublicUpdatesChannelID *discord.Snowflake `json:"public_updates_channel_id"`
	OwnerID                *discord.Snowflake `json:"owner_id"`
	AFKChannelID           *discord.Snowflake `json:"afk_channel_id"`
	WidgetChannelID        *discord.Snowflake `json:"widget_channel_id"`
	ApplicationID          *discord.Snowflake `json:"application_id"`
	SystemChannelID        *discord.Snowflake `json:"system_channel_id"`
	RulesChannelID         *discord.Snowflake `json:"rules_channel_id"`

	RoleIDs        []discord.Snowflake     `json:"role_ids"`
	EmojiIDs       []discord.Snowflake     `json:"emoji_ids"`
	Features       []string                `json:"features"`
	ChannelIDs     []discord.Snowflake     `json:"channel_ids"`
	StageInstances []discord.StageInstance `json:"stage_instances"`
	Stickers       []discord.Sticker       `json:"stickers"`
}

type StateDMChannel struct {
	*discord.Channel

	ExpiresAt int64 `json:"expires_at"`
}

type StateMutualGuilds struct {
	GuildsMu sync.RWMutex `json:"-"`

	Guilds map[discord.Snowflake]bool `json:"guilds"`
}

type StateGuildMembers struct {
	MembersMu sync.RWMutex `json:"-"`

	Members map[discord.Snowflake]*StateGuildMember `json:"members"`
}

type StateGuildMember struct {
	Deaf         bool                `json:"deaf"`
	Mute         bool                `json:"mute"`
	JoinedAt     string              `json:"joined_at"`
	Pending      *bool               `json:"pending"`
	PremiumSince *string             `json:"premium_since"`
	Permissions  *string             `json:"permissions"`
	Nick         *string             `json:"nick"`
	UserID       discord.Snowflake   `json:"user_id"`
	Roles        []discord.Snowflake `json:"roles"`
}

type StateGuildRoles struct {
	RolesMu sync.RWMutex `json:"-"`

	Roles map[discord.Snowflake]*StateRole `json:"roles"`
}

type StateRole struct {
	Hoist       bool              `json:"hoist"`
	Managed     bool              `json:"managed"`
	Mentionable bool              `json:"mentionable"`
	Color       int               `json:"color"`
	Position    int               `json:"position"`
	Permissions int               `json:"permissions"`
	ID          discord.Snowflake `json:"id"`
	Tags        *discord.RoleTag  `json:"tags"`
	Name        string            `json:"name"`
}

type StateGuildEmojis struct {
	EmojisMu sync.RWMutex `json:"-"`

	Emojis map[discord.Snowflake]*StateEmoji `json:"emoji"`
}

type StateEmoji struct {
	ID            discord.Snowflake   `json:"id"`
	Name          string              `json:"name"`
	Roles         []discord.Snowflake `json:"roles"`
	UserID        discord.Snowflake   `json:"user"`
	RequireColons *bool               `json:"require_colons"`
	Managed       bool                `json:"managed"`
	Animated      bool                `json:"animated"`
	Available     bool                `json:"available"`
}

type StateGuildChannels struct {
	ChannelsMu sync.RWMutex `json:"-"`

	Channels map[discord.Snowflake]*StateChannel `json:"channels"`
}

type StateChannel struct {
	ID                   discord.Snowflake          `json:"id"`
	Type                 discord.ChannelType        `json:"type"`
	GuildID              *discord.Snowflake         `json:"guild_id"`
	Position             *int                       `json:"position"`
	PermissionOverwrites []discord.ChannelOverwrite `json:"permission_overwrites"`
	Name                 string                     `json:"name"`
	Topic                *string                    `json:"topic"`
	NSFW                 *bool                      `json:"nsfw"`
	// LastMessageID        *string                     `json:"last_message_id"`
	Bitrate          *int `json:"bitrate"`
	UserLimit        *int `json:"user_limit"`
	RateLimitPerUser *int `json:"rate_limit_per_user"`
	// RecipientIDs         []discord.Snowflake        `json:"recipient_ids"`
	Icon    *string            `json:"icon"`
	OwnerID *discord.Snowflake `json:"owner_id"`
	// ApplicationID        *discord.Snowflake          `json:"application_id"`
	ParentID *discord.Snowflake `json:"parent_id"`
	// LastPinTimestamp     *string                     `json:"last_pin_timestamp"`

	// RTCRegion *string `json:"rtc_region"`
	// VideoQualityMode *discord.VideoQualityMode `json:"video_quality_mode"`

	// Threads.
	// MessageCount               *int                    `json:"message_count"`
	// MemberCount                *int                    `json:"member_count"`
	ThreadMetadata *discord.ThreadMetadata `json:"thread_metadata"`
	// ThreadMember               *discord.ThreadMember   `json:"member"`
	// DefaultAutoArchiveDuration int                     `json:"default_auto_archive_duration"`

	// Slash Commands.
	Permissions *string `json:"permissions"`
}

type StateUser struct {
	ID            discord.Snowflake        `json:"id"`
	Bot           bool                     `json:"bot"`
	System        *bool                    `json:"system"`
	Verified      *bool                    `json:"verified"`
	MFAEnabled    *bool                    `json:"mfa_enabled"`
	Flags         *discord.UserFlags       `json:"flags"`
	PremiumType   *discord.UserPremiumType `json:"premium_type"`
	PublicFlags   *discord.UserFlags       `json:"public_flags"`
	Username      string                   `json:"username"`
	Discriminator string                   `json:"discriminator"`
	Avatar        string                   `json:"avatar"`
	Banner        *string                  `json:"banner"`
	Locale        *string                  `json:"locale"`
	Email         *string                  `json:"email"`
	DMChannelID   *discord.Snowflake       `json:"dm_channel_id"`
}
