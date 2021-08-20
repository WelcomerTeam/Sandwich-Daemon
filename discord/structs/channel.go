package discord

import (
	"time"

	"github.com/WelcomerTeam/RealRock/snowflake"
)

// channel.go contains the information relating to channels

// ChannelType represents a channel's type.
type ChannelType int8

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

// VideoQualityMode represents the quality of the video
type VideoQualityMode int8

const (
	VideoQualityModeAuto VideoQualityMode = 1 + iota
	VideoqualityModeFull
)

// Channel represents a Discord channel.
type Channel struct {
	ID                   snowflake.ID        `json:"id"`
	Type                 ChannelType         `json:"type"`
	GuildID              *snowflake.ID       `json:"guild_id,omitempty"`
	Position             *int                `json:"position,omitempty"`
	PermissionOverwrites []*ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Name                 *string             `json:"name,omitempty"`
	Topic                *string             `json:"topic,omitempty"`
	NSFW                 *bool               `json:"nsfw,omitempty"`
	LastMessageID        *string             `json:"last_message_id,omitempty"`
	Bitrate              *int                `json:"bitrate,omitempty"`
	UserLimit            *int                `json:"user_limit,omitempty"`
	RateLimitPerUser     *int                `json:"rate_limit_per_user,omitempty"`
	Recipients           []*User             `json:"recipients,omitempty"`
	Icon                 *string             `json:"icon,omitempty"`
	OwnerID              *snowflake.ID       `json:"owner_id,omitempty"`
	ApplicationID        *snowflake.ID       `json:"application_id,omitempty"`
	ParentID             *snowflake.ID       `json:"parent_id,omitempty"`
	LastPinTimestamp     *time.Time          `json:"last_pin_timestamp,omitempty"`

	RTCRegion        *string           `json:"rtc_region,omitempty"`
	VideoQualityMode *VideoQualityMode `json:"video_quality_mode,omitempty"`

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
	ID    snowflake.ID `json:"id"`
	Type  int          `json:"type"`
	Allow int64        `json:"allow"`
	Deny  int64        `json:"deny"`
}

// ThreadMetadata contains thread-specific channel fields.
type ThreadMetadata struct {
	Archived            bool      `json:"archived"`
	AutoArchiveDuration int       `json:"auto_archive_duration"`
	ArchiveTimestamp    time.Time `json:"archive_timestamp"`
	Locked              *bool     `json:"locked,omitempty"`
}

// ThreadMember is used to indicate whether a user has joined a thread or not.
type ThreadMember struct {
	ID            *snowflake.ID `json:"id,omitempty"`
	UserID        *snowflake.ID `json:"user_id,omitempty"`
	JoinTimestamp time.Time     `json:"join_timestamp`
	Flags         int           `json:"flags"`
}
