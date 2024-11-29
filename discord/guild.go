package discord

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
type MFALevel uint16

// MFA levels.
const (
	MFALevelNone MFALevel = iota
	MFALevelElevated
)

// VerificationLevel represents a guild's verification level.
type VerificationLevel uint16

const (
	VerificationLevelNone VerificationLevel = iota
	VerificationLevelLow
	VerificationLevelMedium
	VerificationLevelHigh
	VerificationLevelVeryHigh
)

// SystemChannelFlags represents the flags of a system channel.
type SystemChannelFlags uint16

const (
	SystemChannelFlagsSuppressJoin SystemChannelFlags = 1 << iota
	SystemChannelFlagsPremiumSubscriptions
	SystemChannelFlagsSuppressSetupTips
	SystemChannelFlagsHideMemberJoinStickerReplyButtons
	SystemChannelFlagsSuppressSubscriptionNotifications
	SystemChannelFlagsHideRoleSubscriptionReplyButtons
	_
	_
)

// PremiumTier represents the current boosting tier of a guild.
type PremiumTier uint16

const (
	PremiumTierNone PremiumTier = iota
	PremiumTier1
	PremiumTier2
	PremiumTier3
)

// GuildNSFWLevelType represents the level of the guild.
type GuildNSFWLevelType uint16

const (
	GuildNSFWLevelTypDefault GuildNSFWLevelType = iota
	GuildNSFWLevelTypeExplicit
	GuildNSFWLevelTypeSafe
	GuildNSFWLevelTypeAgeRestricted
)

// Guild represents a guild on discord.
type Guild struct {
	WidgetChannelID             *ChannelID                 `json:"widget_channel_id,omitempty"`
	PublicUpdatesChannelID      *ChannelID                 `json:"public_updates_channel_id,omitempty"`
	PremiumTier                 *PremiumTier               `json:"premium_tier,omitempty"`
	RulesChannelID              *ChannelID                 `json:"rules_channel_id,omitempty"`
	SystemChannelFlags          *SystemChannelFlags        `json:"system_channel_flags,omitempty"`
	Permissions                 *Int64                     `json:"permissions,omitempty"`
	SystemChannelID             *ChannelID                 `json:"system_channel_id,omitempty"`
	AFKChannelID                *ChannelID                 `json:"afk_channel_id,omitempty"`
	ApplicationID               *ApplicationID             `json:"application_id,omitempty"`
	Icon                        *string                    `json:"icon"`
	WidgetEnabled               *bool                      `json:"widget_enabled,omitempty"`
	JoinedAt                    Timestamp                  `json:"joined_at"`
	Description                 string                     `json:"description"`
	PreferredLocale             string                     `json:"preferred_locale"`
	Name                        string                     `json:"name"`
	IconHash                    string                     `json:"icon_hash,omitempty"`
	Banner                      string                     `json:"banner,omitempty"`
	VanityURLCode               string                     `json:"vanity_url_code"`
	Splash                      string                     `json:"splash,omitempty"`
	DiscoverySplash             string                     `json:"discovery_splash,omitempty"`
	Region                      string                     `json:"region"`
	Presences                   PresenceUpdateList         `json:"presences"`
	GuildScheduledEvents        ScheduledEventList         `json:"guild_scheduled_events"`
	Stickers                    StickerList                `json:"stickers"`
	Features                    StringList                 `json:"features"`
	StageInstances              StageInstanceList          `json:"stage_instances"`
	Roles                       RoleList                   `json:"roles"`
	Emojis                      EmojiList                  `json:"emojis"`
	VoiceStates                 VoiceStateList             `json:"voice_states"`
	Members                     GuildMemberList            `json:"members"`
	Channels                    ChannelList                `json:"channels"`
	Threads                     ChannelList                `json:"threads"`
	OwnerID                     UserID                     `json:"owner_id"`
	ID                          GuildID                    `json:"id"`
	ExplicitContentFilter       ExplicitContentFilterLevel `json:"explicit_content_filter"`
	DefaultMessageNotifications MessageNotificationLevel   `json:"default_message_notifications"`
	ApproximateMemberCount      int32                      `json:"approximate_member_count"`
	MaxMembers                  int32                      `json:"max_members"`
	MemberCount                 int32                      `json:"member_count"`
	AFKTimeout                  int32                      `json:"afk_timeout"`
	MaxPresences                int32                      `json:"max_presences"`
	PremiumSubscriptionCount    int32                      `json:"premium_subscription_count"`
	ApproximatePresenceCount    int32                      `json:"approximate_presence_count"`
	MaxVideoChannelUsers        int32                      `json:"max_video_channel_users"`
	NSFWLevel                   GuildNSFWLevelType         `json:"nsfw_level"`
	VerificationLevel           VerificationLevel          `json:"verification_level"`
	MFALevel                    MFALevel                   `json:"mfa_level"`
	Unavailable                 bool                       `json:"unavailable"`
	Large                       bool                       `json:"large"`
	Owner                       bool                       `json:"owner"`
	PremiumProgressBarEnabled   bool                       `json:"premium_progress_bar_enabled"`
}

// UnavailableGuild represents an unavailable guild.
type UnavailableGuild struct {
	ID          GuildID `json:"id"`
	Unavailable bool    `json:"unavailable"`
}

// GuildMember represents a guild member on discord.
type GuildMember struct {
	User                       *User      `json:"user,omitempty"`
	GuildID                    *GuildID   `json:"guild_id,omitempty"`
	CommunicationDisabledUntil *string    `json:"communication_disabled_until,omitempty"`
	Nick                       string     `json:"nick,omitempty"`
	Avatar                     string     `json:"avatar,omitempty"`
	PremiumSince               string     `json:"premium_since,omitempty"`
	JoinedAt                   Timestamp  `json:"joined_at,omitempty"`
	Roles                      RoleIDList `json:"roles"`
	Permissions                Int64      `json:"permissions"`
	Flags                      int        `json:"flags"`
	Deaf                       bool       `json:"deaf"`
	Mute                       bool       `json:"mute"`
	Pending                    bool       `json:"pending"`
}

// VoiceState represents the voice state on discord.
type VoiceState struct {
	RequestToSpeakTimestamp *Timestamp   `json:"request_to_speak_timestamp"`
	GuildID                 *GuildID     `json:"guild_id,omitempty"`
	Member                  *GuildMember `json:"member,omitempty"`
	SessionID               string       `json:"session_id"`
	UserID                  UserID       `json:"user_id"`
	ChannelID               ChannelID    `json:"channel_id"`
	Mute                    bool         `json:"mute"`
	SelfDeaf                bool         `json:"self_deaf"`
	SelfMute                bool         `json:"self_mute"`
	SelfStream              bool         `json:"self_stream"`
	SelfVideo               bool         `json:"self_video"`
	Suppress                bool         `json:"suppress"`
	Deaf                    bool         `json:"deaf"`
}

// GuildBan represents a ban entry.
type GuildBan struct {
	GuildID *GuildID `json:"guild_id,omitempty"`
	Reason  string
	User    User `json:"user"`
}

// GuildPruneParam represents the arguments for a guild prune.
type GuildPruneParam struct {
	Days              *int32   `json:"days,omitempty"`
	IncludeRoles      []RoleID `json:"include_roles"`
	ComputePruneCount bool     `json:"compute_prune_count"`
}
