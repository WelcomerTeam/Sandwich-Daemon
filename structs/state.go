package structs

import (
	"sync"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
)

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  interface{}
	Extra map[string]interface{}
}

type StateGuild struct {
	ID              discord.Snowflake `json:"id"`
	Name            string            `json:"name"`
	Icon            string            `json:"icon"`
	IconHash        *string           `json:"icon_hash,omitempty"`
	Splash          string            `json:"splash"`
	DiscoverySplash string            `json:"discovery_splash"`

	Owner       *bool              `json:"owner,omitempty"`
	OwnerID     *discord.Snowflake `json:"owner_id,omitempty"`
	Permissions *int               `json:"permissions,omitempty"`
	Region      string             `json:"region"`

	AFKChannelID *discord.Snowflake `json:"afk_channel_id,omitempty"`
	AFKTimeout   int                `json:"afk_timeout"`

	WidgetEnabled   *bool              `json:"widget_enabled,omitempty"`
	WidgetChannelID *discord.Snowflake `json:"widget_channel_id,omitempty"`

	VerificationLevel           discord.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications discord.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       discord.ExplicitContentFilterLevel `json:"explicit_content_filter"`

	RoleIDs  []*discord.Snowflake `json:"role_idss"`
	EmojiIDs []*discord.Snowflake `json:"emoji_ids"`
	Features []string             `json:"features"`

	MFALevel           discord.MFALevel            `json:"mfa_level"`
	ApplicationID      *discord.Snowflake          `json:"application_id,omitempty"`
	SystemChannelID    *discord.Snowflake          `json:"system_channel_id,omitempty"`
	SystemChannelFlags *discord.SystemChannelFlags `json:"system_channel_flags,omitempty"`
	RulesChannelID     *discord.Snowflake          `json:"rules_channel_id,omitempty"`

	JoinedAt    *string `json:"joined_at,omitempty"`
	Large       *bool   `json:"large,omitempty"`
	MemberCount *int    `json:"member_count,omitempty"`

	ChannelIDs []*discord.Snowflake `json:"channel_ids,omitempty"`

	MaxPresences  *int                 `json:"max_presences,omitempty"`
	MaxMembers    *int                 `json:"max_members,omitempty"`
	VanityURLCode *string              `json:"vanity_url_code,omitempty"`
	Description   *string              `json:"description,omitempty"`
	Banner        *string              `json:"banner,omitempty"`
	PremiumTier   *discord.PremiumTier `json:"premium_tier,omitempty"`

	PremiumSubscriptionCount *int               `json:"premium_subscription_count,omitempty"`
	PreferredLocale          *string            `json:"preferred_locale,omitempty"`
	PublicUpdatesChannelID   *discord.Snowflake `json:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers     *int               `json:"max_video_channel_users,omitempty"`
	ApproximateMemberCount   *int               `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount *int               `json:"approximate_presence_count,omitempty"`

	NSFWLevel      *discord.GuildNSFWLevelType `json:"nsfw_level"`
	StageInstances []*discord.StageInstance    `json:"stage_instances,omitempty"`
	Stickers       []*discord.Sticker          `json:"stickers"`
}

type StateGuildMembers struct {
	MembersMu sync.RWMutex `json:"-"`

	Members map[discord.Snowflake]*StateGuildMember `json:"members"`
}

type StateGuildMember struct {
	UserID *discord.User `json:"user_id"`
	Nick   *string       `json:"nick,omitempty"`

	Roles    []discord.Snowflake `json:"roles"`
	JoinedAt string              `json:"joined_at"`
	Deaf     bool                `json:"deaf"`
	Mute     bool                `json:"mute"`
}

type StateRole struct {
	ID          discord.Snowflake `json:"id"`
	Name        string            `json:"name"`
	Color       int               `json:"color"`
	Hoist       bool              `json:"hoist"`
	Position    int               `json:"position"`
	Permissions int               `json:"permissions"`
	Managed     bool              `json:"managed"`
	Mentionable bool              `json:"mentionable"`
	Tags        *discord.RoleTag  `json:"tags,omitempty"`
}

type StateEmoji struct {
	ID            discord.Snowflake   `json:"id"`
	Name          string              `json:"name"`
	Roles         []discord.Snowflake `json:"roles,omitempty"`
	UserID        *discord.Snowflake  `json:"user,omitempty"`
	RequireColons *bool               `json:"require_colons,omitempty"`
	Managed       *bool               `json:"managed,omitempty"`
	Animated      *bool               `json:"animated,omitempty"`
	Available     *bool               `json:"available,omitempty"`
}

type StateUser struct {
	ID            discord.Snowflake        `json:"id"`
	Username      string                   `json:"username"`
	Discriminator string                   `json:"discriminator"`
	Avatar        *string                  `json:"avatar,omitempty"`
	Bot           *bool                    `json:"bot,omitempty"`
	System        *bool                    `json:"system,omitempty"`
	MFAEnabled    *bool                    `json:"mfa_enabled,omitempty"`
	Banner        *string                  `json:"banner,omitempty"`
	Locale        *string                  `json:"locale,omitempty"`
	Verified      *bool                    `json:"verified,omitempty"`
	Email         *string                  `json:"email,omitempty"`
	Flags         *discord.UserFlags       `json:"flags,omitempty"`
	PremiumType   *discord.UserPremiumType `json:"premium_type,omitempty"`
	PublicFlags   *discord.UserFlags       `json:"public_flags,omitempty"`
}

type StateChannel struct {
	ID                   discord.Snowflake           `json:"id"`
	Type                 discord.ChannelType         `json:"type"`
	GuildID              *discord.Snowflake          `json:"guild_id,omitempty"`
	Position             *int                        `json:"position,omitempty"`
	PermissionOverwrites []*discord.ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Name                 *string                     `json:"name,omitempty"`
	Topic                *string                     `json:"topic,omitempty"`
	NSFW                 *bool                       `json:"nsfw,omitempty"`
	LastMessageID        *string                     `json:"last_message_id,omitempty"`
	Bitrate              *int                        `json:"bitrate,omitempty"`
	UserLimit            *int                        `json:"user_limit,omitempty"`
	RateLimitPerUser     *int                        `json:"rate_limit_per_user,omitempty"`
	RecipientIDs         []*discord.Snowflake        `json:"recipient_ids,omitempty"`
	Icon                 *string                     `json:"icon,omitempty"`
	OwnerID              *discord.Snowflake          `json:"owner_id,omitempty"`
	ApplicationID        *discord.Snowflake          `json:"application_id,omitempty"`
	ParentID             *discord.Snowflake          `json:"parent_id,omitempty"`
	LastPinTimestamp     *string                     `json:"last_pin_timestamp,omitempty"`

	RTCRegion        *string                   `json:"rtc_region,omitempty"`
	VideoQualityMode *discord.VideoQualityMode `json:"video_quality_mode,omitempty"`

	// Threads.
	MessageCount               *int                    `json:"message_count,omitempty"`
	MemberCount                *int                    `json:"member_count,omitempty"`
	ThreadMetadata             *discord.ThreadMetadata `json:"thread_metadata,omitempty"`
	ThreadMember               *discord.ThreadMember   `json:"member,omitempty"`
	DefaultAutoArchiveDuration int                     `json:"default_auto_archive_duration"`

	// Slash Commands.
	Permissions *string `json:"permissions,omitempty"`
}
