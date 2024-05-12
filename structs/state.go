package structs

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
)

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  json.RawMessage
	Extra map[string]json.RawMessage
}

type StateDMChannel struct {
	*discord.Channel

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

type StateGuildVoiceStates struct {
	VoiceStatesMu sync.RWMutex `json:"-"`

	VoiceStates map[discord.Snowflake]*StateVoiceState `json:"voice_states"`
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

	VerificationLevel           discord.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications discord.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       discord.ExplicitContentFilterLevel `json:"explicit_content_filter"`

	MFALevel           discord.MFALevel            `json:"mfa_level"`
	ApplicationID      *discord.Snowflake          `json:"application_id,omitempty"`
	SystemChannelID    *discord.Snowflake          `json:"system_channel_id,omitempty"`
	SystemChannelFlags *discord.SystemChannelFlags `json:"system_channel_flags,omitempty"`
	RulesChannelID     *discord.Snowflake          `json:"rules_channel_id,omitempty"`

	JoinedAt    time.Time `json:"joined_at"`
	Large       bool      `json:"large"`
	Unavailable bool      `json:"unavailable"`
	MemberCount int32     `json:"member_count"`

	MaxPresences  int32  `json:"max_presences"`
	MaxMembers    int32  `json:"max_members"`
	VanityURLCode string `json:"vanity_url_code"`
	Description   string `json:"description"`
	Banner        string `json:"banner"`

	PremiumTier               *discord.PremiumTier        `json:"premium_tier,omitempty"`
	PremiumSubscriptionCount  int32                       `json:"premium_subscription_count"`
	PreferredLocale           string                      `json:"preferred_locale"`
	PublicUpdatesChannelID    *discord.Snowflake          `json:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers      int32                       `json:"max_video_channel_users"`
	ApproximateMemberCount    int32                       `json:"approximate_member_count"`
	ApproximatePresenceCount  int32                       `json:"approximate_presence_count"`
	NSFWLevel                 *discord.GuildNSFWLevelType `json:"nsfw_level"`
	PremiumProgressBarEnabled bool                        `json:"premium_progress_bar_enabled"`

	RoleIDs    []discord.Snowflake `json:"role_ids"`
	EmojiIDs   []discord.Snowflake `json:"emoji_ids"`
	ChannelIDs []discord.Snowflake `json:"channel_ids"`

	Features             []string                 `json:"features"`
	StageInstances       []discord.StageInstance  `json:"stage_instances"`
	Stickers             []discord.Sticker        `json:"stickers"`
	GuildScheduledEvents []discord.ScheduledEvent `json:"guild_scheduled_events"`
}

type StateGuildMember struct {
	UserID                     discord.Snowflake   `json:"user_id"`
	Nick                       string              `json:"nick"`
	Avatar                     string              `json:"avatar,omitempty"`
	Roles                      []discord.Snowflake `json:"roles"`
	JoinedAt                   time.Time           `json:"joined_at"`
	PremiumSince               string              `json:"premium_since"`
	Deaf                       bool                `json:"deaf"`
	Mute                       bool                `json:"mute"`
	Pending                    bool                `json:"pending"`
	Permissions                *discord.Int64      `json:"permissions"`
	CommunicationDisabledUntil string              `json:"communication_disabled_until,omitempty"`
}

type StateRole struct {
	ID           discord.Snowflake `json:"id"`
	Name         string            `json:"name"`
	Color        int32             `json:"color"`
	Hoist        bool              `json:"hoist"`
	Icon         string            `json:"icon"`
	UnicodeEmoji string            `json:"unicode_emoji"`
	Position     int32             `json:"position"`
	Permissions  discord.Int64     `json:"permissions"`
	Managed      bool              `json:"managed"`
	Mentionable  bool              `json:"mentionable"`
	Tags         *discord.RoleTag  `json:"tags"`
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
	ID                         discord.Snowflake          `json:"id"`
	Type                       discord.ChannelType        `json:"type"`
	GuildID                    *discord.Snowflake         `json:"guild_id,omitempty"`
	Position                   int32                      `json:"position,omitempty"`
	PermissionOverwrites       []discord.ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Name                       string                     `json:"name,omitempty"`
	Topic                      string                     `json:"topic,omitempty"`
	NSFW                       bool                       `json:"nsfw"`
	LastMessageID              string                     `json:"last_message_id,omitempty"`
	Bitrate                    int32                      `json:"bitrate,omitempty"`
	UserLimit                  int32                      `json:"user_limit,omitempty"`
	RateLimitPerUser           int32                      `json:"rate_limit_per_user,omitempty"`
	Recipients                 []discord.Snowflake        `json:"recipients,omitempty"`
	Icon                       string                     `json:"icon,omitempty"`
	OwnerID                    *discord.Snowflake         `json:"owner_id,omitempty"`
	ApplicationID              *discord.Snowflake         `json:"application_id,omitempty"`
	ParentID                   *discord.Snowflake         `json:"parent_id,omitempty"`
	LastPinTimestamp           *time.Time                 `json:"last_pin_timestamp,omitempty"`
	RTCRegion                  string                     `json:"rtc_region,omitempty"`
	VideoQualityMode           *discord.VideoQualityMode  `json:"video_quality_mode,omitempty"`
	MessageCount               int32                      `json:"message_count,omitempty"`
	MemberCount                int32                      `json:"member_count,omitempty"`
	ThreadMetadata             *discord.ThreadMetadata    `json:"thread_metadata,omitempty"`
	ThreadMember               *discord.ThreadMember      `json:"member,omitempty"`
	DefaultAutoArchiveDuration int32                      `json:"default_auto_archive_duration,omitempty"`
	Permissions                *discord.Int64             `json:"permissions,omitempty"`
}

type StateUser struct {
	ID            discord.Snowflake       `json:"id"`
	Username      string                  `json:"username"`
	Discriminator string                  `json:"discriminator"`
	GlobalName    string                  `json:"global_name"`
	Avatar        string                  `json:"avatar"`
	Bot           bool                    `json:"bot"`
	System        bool                    `json:"system,omitempty"`
	MFAEnabled    bool                    `json:"mfa_enabled,omitempty"`
	Banner        string                  `json:"banner,omitempty"`
	AccentColour  int32                   `json:"accent_color"`
	Locale        string                  `json:"locale,omitempty"`
	Verified      bool                    `json:"verified,omitempty"`
	Email         string                  `json:"email,omitempty"`
	Flags         discord.UserFlags       `json:"flags,omitempty"`
	PremiumType   discord.UserPremiumType `json:"premium_type,omitempty"`
	PublicFlags   discord.UserFlags       `json:"public_flags,omitempty"`
	DMChannelID   *discord.Snowflake      `json:"dm_channel_id,omitempty"`
}

type StateVoiceState struct {
	ChannelID               discord.Snowflake `json:"channel_id"`
	SessionID               string            `json:"session_id"`
	Deaf                    bool              `json:"deaf"`
	Mute                    bool              `json:"mute"`
	SelfDeaf                bool              `json:"self_deaf"`
	SelfMute                bool              `json:"self_mute"`
	SelfStream              bool              `json:"self_stream"`
	SelfVideo               bool              `json:"self_video"`
	Suppress                bool              `json:"suppress"`
	RequestToSpeakTimestamp time.Time         `json:"request_to_speak_timestamp"`
}
