package structs

import (
	discord "github.com/WelcomerTeam/Discord/discord"
	discord_structs "github.com/WelcomerTeam/Discord/structs"
	jsoniter "github.com/json-iterator/go"
	"sync"
)

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  jsoniter.RawMessage
	Extra map[string]jsoniter.RawMessage
}

type StateDMChannel struct {
	*discord_structs.Channel

	ExpiresAt discord.Int64 `json:"expires_at"`
}

type StateMutualGuilds struct {
	GuildsMu sync.RWMutex `json:"-"`

	Guilds map[discord.Snowflake]bool `json:"guilds"`
}

type StateGuildMembers struct {
	MembersMu sync.RWMutex `json:"-"`

	Members map[discord.Snowflake]*StateGuildMember `json:"members"`
}

type StateGuildRoles struct {
	RolesMu sync.RWMutex `json:"-"`

	Roles map[discord.Snowflake]*StateRole `json:"roles"`
}

type StateGuildEmojis struct {
	EmojisMu sync.RWMutex `json:"-"`

	Emojis map[discord.Snowflake]*StateEmoji `json:"emoji"`
}

type StateGuildChannels struct {
	ChannelsMu sync.RWMutex `json:"-"`

	Channels map[discord.Snowflake]*StateChannel `json:"channels"`
}

type StateGuild struct {
	ID              discord.Snowflake `json:"id"`
	Name            string            `json:"name"`
	Icon            string            `json:"icon"`
	IconHash        string            `json:"icon_hash"`
	Splash          string            `json:"splash"`
	DiscoverySplash string            `json:"discovery_splash"`

	Owner       bool               `json:"owner"`
	OwnerID     *discord.Snowflake `json:"owner_id,omitempty"`
	Permissions *discord.Int64     `json:"permissions,omitempty"`
	Region      string             `json:"region"`

	AFKChannelID *discord.Snowflake `json:"afk_channel_id,omitempty"`
	AFKTimeout   int32              `json:"afk_timeout"`

	WidgetEnabled   bool               `json:"widget_enabled"`
	WidgetChannelID *discord.Snowflake `json:"widget_channel_id,omitempty"`

	VerificationLevel           discord_structs.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications discord_structs.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       discord_structs.ExplicitContentFilterLevel `json:"explicit_content_filter"`

	MFALevel           discord_structs.MFALevel            `json:"mfa_level"`
	ApplicationID      *discord.Snowflake                  `json:"application_id,omitempty"`
	SystemChannelID    *discord.Snowflake                  `json:"system_channel_id,omitempty"`
	SystemChannelFlags *discord_structs.SystemChannelFlags `json:"system_channel_flags,omitempty"`
	RulesChannelID     *discord.Snowflake                  `json:"rules_channel_id,omitempty"`

	JoinedAt    string `json:"joined_at"`
	Large       bool   `json:"large"`
	Unavailable bool   `json:"unavailable"`
	MemberCount int32  `json:"member_count"`

	MaxPresences  int    `json:"max_presences"`
	MaxMembers    int    `json:"max_members"`
	VanityURLCode string `json:"vanity_url_code"`
	Description   string `json:"description"`
	Banner        string `json:"banner"`

	PremiumTier               *discord_structs.PremiumTier        `json:"premium_tier,omitempty"`
	PremiumSubscriptionCount  int                                 `json:"premium_subscription_count"`
	PreferredLocale           string                              `json:"preferred_locale"`
	PublicUpdatesChannelID    *discord.Snowflake                  `json:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers      int                                 `json:"max_video_channel_users"`
	ApproximateMemberCount    int                                 `json:"approximate_member_count"`
	ApproximatePresenceCount  int                                 `json:"approximate_presence_count"`
	NSFWLevel                 *discord_structs.GuildNSFWLevelType `json:"nsfw_level"`
	PremiumProgressBarEnabled bool                                `json:"premium_progress_bar_enabled"`

	RoleIDs    []discord.Snowflake `json:"role_ids"`
	EmojiIDs   []discord.Snowflake `json:"emoji_ids"`
	ChannelIDs []discord.Snowflake `json:"channel_ids"`

	Features             []string                         `json:"features"`
	StageInstances       []discord_structs.StageInstance  `json:"stage_instances"`
	Stickers             []discord_structs.Sticker        `json:"stickers"`
	GuildScheduledEvents []discord_structs.ScheduledEvent `json:"guild_scheduled_events"`
}

type StateGuildMember struct {
	UserID                     discord.Snowflake   `json:"user_id"`
	Nick                       string              `json:"nick"`
	Avatar                     string              `json:"avatar,omitempty"`
	Roles                      []discord.Snowflake `json:"roles"`
	JoinedAt                   string              `json:"joined_at"`
	PremiumSince               string              `json:"premium_since"`
	Deaf                       bool                `json:"deaf"`
	Mute                       bool                `json:"mute"`
	Pending                    bool                `json:"pending"`
	Permissions                *discord.Int64      `json:"permissions"`
	CommunicationDisabledUntil string              `json:"communication_disabled_until,omitempty"`
}

type StateRole struct {
	ID           discord.Snowflake        `json:"id"`
	Name         string                   `json:"name"`
	Color        int32                    `json:"color"`
	Hoist        bool                     `json:"hoist"`
	Icon         string                   `json:"icon"`
	UnicodeEmoji string                   `json:"unicode_emoji"`
	Position     int32                    `json:"position"`
	Permissions  discord.Int64            `json:"permissions"`
	Managed      bool                     `json:"managed"`
	Mentionable  bool                     `json:"mentionable"`
	Tags         *discord_structs.RoleTag `json:"tags"`
}

type StateEmoji struct {
	ID            discord.Snowflake   `json:"id"`
	GuildID       discord.Snowflake   `json:"guild_id"`
	Name          string              `json:"name"`
	Roles         []discord.Snowflake `json:"roles,omitempty"`
	UserID        discord.Snowflake   `json:"user"`
	RequireColons bool                `json:"require_colons"`
	Managed       bool                `json:"managed"`
	Animated      bool                `json:"animated"`
	Available     bool                `json:"available"`
}

type StateChannel struct {
	ID                         discord.Snowflake                  `json:"id"`
	Type                       discord_structs.ChannelType        `json:"type"`
	GuildID                    *discord.Snowflake                 `json:"guild_id,omitempty"`
	Position                   int32                              `json:"position,omitempty"`
	PermissionOverwrites       []discord_structs.ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Name                       string                             `json:"name,omitempty"`
	Topic                      string                             `json:"topic,omitempty"`
	NSFW                       bool                               `json:"nsfw"`
	LastMessageID              string                             `json:"last_message_id,omitempty"`
	Bitrate                    int32                              `json:"bitrate,omitempty"`
	UserLimit                  int32                              `json:"user_limit,omitempty"`
	RateLimitPerUser           int32                              `json:"rate_limit_per_user,omitempty"`
	Recipients                 []discord.Snowflake                `json:"recipients,omitempty"`
	Icon                       string                             `json:"icon,omitempty"`
	OwnerID                    *discord.Snowflake                 `json:"owner_id,omitempty"`
	ApplicationID              *discord.Snowflake                 `json:"application_id,omitempty"`
	ParentID                   *discord.Snowflake                 `json:"parent_id,omitempty"`
	LastPinTimestamp           string                             `json:"last_pin_timestamp,omitempty"`
	RTCRegion                  string                             `json:"rtc_region,omitempty"`
	VideoQualityMode           *discord_structs.VideoQualityMode  `json:"video_quality_mode,omitempty"`
	MessageCount               int32                              `json:"message_count,omitempty"`
	MemberCount                int32                              `json:"member_count,omitempty"`
	ThreadMetadata             *discord_structs.ThreadMetadata    `json:"thread_metadata,omitempty"`
	ThreadMember               *discord_structs.ThreadMember      `json:"member,omitempty"`
	DefaultAutoArchiveDuration int32                              `json:"default_auto_archive_duration,omitempty"`
	Permissions                *discord.Int64                     `json:"permissions,omitempty"`
}

type StateUser struct {
	ID            discord.Snowflake                `json:"id"`
	Username      string                           `json:"username"`
	Discriminator string                           `json:"discriminator"`
	Avatar        string                           `json:"avatar"`
	Bot           bool                             `json:"bot"`
	System        bool                             `json:"system,omitempty"`
	MFAEnabled    bool                             `json:"mfa_enabled,omitempty"`
	Banner        string                           `json:"banner,omitempty"`
	AccentColour  int32                            `json:"accent_color"`
	Locale        string                           `json:"locale,omitempty"`
	Verified      bool                             `json:"verified,omitempty"`
	Email         string                           `json:"email,omitempty"`
	Flags         *discord_structs.UserFlags       `json:"flags,omitempty"`
	PremiumType   *discord_structs.UserPremiumType `json:"premium_type,omitempty"`
	PublicFlags   *discord_structs.UserFlags       `json:"public_flags,omitempty"`
	DMChannelID   *discord.Snowflake               `json:"dm_channel_id,omitempty"`
}
