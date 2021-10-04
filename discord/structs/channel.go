package discord

// channel.go contains the information relating to channels

// ChannelType represents a channel's type.
type ChannelType uint8

const (
	ChannelTypeGuildText ChannelType = iota
	ChannelTypeDM
	ChannelTypeGuildVoice
	ChannelTypeGroupDM
	ChannelTypeGuildCategory
	ChannelTypeGuildNews
	ChannelTypeGuildStore
	_
	_
	_
	ChannelTypeGuildNewsThread
	ChannelTypeGuildPublicThread
	ChannelTypeGuildPrivateThread
	ChannelTypeGuildStageVoice
)

// VideoQualityMode represents the quality of the video.
type VideoQualityMode uint8

const (
	VideoQualityModeAuto VideoQualityMode = 1 + iota
	VideoqualityModeFull
)

// StageChannelPrivacyLevel represents the privacy level of a stage channel.
type StageChannelPrivacyLevel uint8

const (
	StageChannelPrivacyLevelPublic StageChannelPrivacyLevel = 1 + iota
	StageChannelPrivacyLevelGuildOnly
)

// Channel represents a Discord channel.
type Channel struct {
	ID                   Snowflake           `json:"id"`
	Type                 ChannelType         `json:"type"`
	GuildID              *Snowflake          `json:"guild_id,omitempty"`
	Position             *int                `json:"position,omitempty"`
	PermissionOverwrites []*ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Name                 string              `json:"name"`
	Topic                *string             `json:"topic,omitempty"`
	NSFW                 *bool               `json:"nsfw,omitempty"`
	LastMessageID        *string             `json:"last_message_id,omitempty"`
	Bitrate              *int                `json:"bitrate,omitempty"`
	UserLimit            *int                `json:"user_limit,omitempty"`
	RateLimitPerUser     *int                `json:"rate_limit_per_user,omitempty"`
	Recipients           []User              `json:"recipients,omitempty"`
	Icon                 *string             `json:"icon,omitempty"`
	OwnerID              *Snowflake          `json:"owner_id,omitempty"`
	ApplicationID        *Snowflake          `json:"application_id,omitempty"`
	ParentID             *Snowflake          `json:"parent_id,omitempty"`
	LastPinTimestamp     *string             `json:"last_pin_timestamp,omitempty"`

	RTCRegion *string `json:"rtc_region,omitempty"`
	// VideoQualityMode *VideoQualityMode `json:"video_quality_mode,omitempty"`

	// Threads
	MessageCount               *int            `json:"message_count,omitempty"`
	MemberCount                *int            `json:"member_count,omitempty"`
	ThreadMetadata             *ThreadMetadata `json:"thread_metadata,omitempty"`
	ThreadMember               *ThreadMember   `json:"member,omitempty"`
	DefaultAutoArchiveDuration int             `json:"default_auto_archive_duration"`

	// Slash Commands
	Permissions *string `json:"permissions,omitempty"`
}

// ChannelOverwrite represents a permission overwrite for a channel.
type ChannelOverwrite struct {
	ID    Snowflake `json:"id"`
	Type  string    `json:"type"`
	Allow jInt64    `json:"allow_new"`
	Deny  jInt64    `json:"deny_new"`
}

// ThreadMetadata contains thread-specific channel fields.
type ThreadMetadata struct {
	Archived            bool   `json:"archived"`
	AutoArchiveDuration int    `json:"auto_archive_duration"`
	ArchiveTimestamp    string `json:"archive_timestamp"`
	Locked              *bool  `json:"locked,omitempty"`
}

// ThreadMember is used to indicate whether a user has joined a thread or not.
type ThreadMember struct {
	ID            *Snowflake `json:"id,omitempty"`
	UserID        *Snowflake `json:"user_id,omitempty"`
	JoinTimestamp string     `json:"join_timestamp"`
	Flags         int        `json:"flags"`
}

// StageInstance represents a stage channel instance.
type StageInstance struct {
	ID                   Snowflake                `json:"id"`
	GuildID              Snowflake                `json:"guild_id"`
	ChannelID            Snowflake                `json:"channel_id"`
	Topic                string                   `json:"topic"`
	PrivacyLabel         StageChannelPrivacyLevel `json:"privacy_level"`
	DiscoverableDisabled bool                     `json:"discoverable_disabled"`
}
