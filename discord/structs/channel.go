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
	ID                         Snowflake           `json:"id"`
	GuildID                    *Snowflake          `json:"guild_id,omitempty"`
	Type                       ChannelType         `json:"type"`
	Position                   int32               `json:"position,omitempty"`
	PermissionOverwrites       []*ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Name                       string              `json:"name,omitempty"`
	Topic                      string              `json:"topic,omitempty"`
	NSFW                       bool                `json:"nsfw"`
	LastMessageID              string              `json:"last_message_id,omitempty"`
	Bitrate                    int32               `json:"bitrate,omitempty"`
	UserLimit                  int32               `json:"user_limit,omitempty"`
	RateLimitPerUser           int32               `json:"rate_limit_per_user,omitempty"`
	Recipients                 []*User             `json:"recipients,omitempty"`
	Icon                       string              `json:"icon,omitempty"`
	OwnerID                    *Snowflake          `json:"owner_id,omitempty"`
	ApplicationID              *Snowflake          `json:"application_id,omitempty"`
	ParentID                   *Snowflake          `json:"parent_id,omitempty"`
	LastPinTimestamp           string              `json:"last_pin_timestamp,omitempty"`
	RTCRegion                  string              `json:"rtc_region,omitempty"`
	VideoQualityMode           *VideoQualityMode   `json:"video_quality_mode,omitempty"`
	MessageCount               int32               `json:"message_count,omitempty"`
	MemberCount                int32               `json:"member_count,omitempty"`
	ThreadMetadata             *ThreadMetadata     `json:"thread_metadata,omitempty"`
	ThreadMember               *ThreadMember       `json:"member,omitempty"`
	DefaultAutoArchiveDuration int32               `json:"default_auto_archive_duration,omitempty"`
	Permissions                *Int64              `json:"permissions,omitempty"`
}

// ChannelOverwrite represents a permission overwrite for a channel.
type ChannelOverwrite struct {
	ID    Snowflake            `json:"id"`
	Type  *ChannelOverrideType `json:"type"`
	Allow Int64                `json:"allow"`
	Deny  Int64                `json:"deny"`
}

// ChannelOverrideType represents the target of a channel override.
type ChannelOverrideType uint8

const (
	ChannelOverrideTypeRole ChannelOverrideType = iota
	ChannelOverrideTypeMember
)

// ThreadMetadata contains thread-specific channel fields.
type ThreadMetadata struct {
	Archived            bool   `json:"archived"`
	AutoArchiveDuration int32  `json:"auto_archive_duration"`
	ArchiveTimestamp    string `json:"archive_timestamp"`
	Locked              bool   `json:"locked"`
}

// ThreadMember is used to indicate whether a user has joined a thread or not.
type ThreadMember struct {
	ID            *Snowflake `json:"id,omitempty"`
	UserID        *Snowflake `json:"user_id,omitempty"`
	GuildID       *Snowflake `json:"guild_id,omitempty"`
	JoinTimestamp string     `json:"join_timestamp"`
	Flags         int32      `json:"flags"`
}

// StageInstance represents a stage channel instance.
type StageInstance struct {
	ID                   Snowflake                 `json:"id"`
	GuildID              Snowflake                 `json:"guild_id"`
	ChannelID            Snowflake                 `json:"channel_id"`
	Topic                string                    `json:"topic"`
	PrivacyLabel         *StageChannelPrivacyLevel `json:"privacy_level"`
	DiscoverableDisabled bool                      `json:"discoverable_disabled"`
}
