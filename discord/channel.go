package discord

import (
	"bytes"
	"fmt"
	"strconv"
)

// channel.go contains the information relating to channels

// ChannelType represents a channel's type.
type ChannelType uint16

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
type VideoQualityMode uint16

const (
	VideoQualityModeAuto VideoQualityMode = 1 + iota
	VideoqualityModeFull
)

// StageChannelPrivacyLevel represents the privacy level of a stage channel.
type StageChannelPrivacyLevel uint16

const (
	StageChannelPrivacyLevelPublic StageChannelPrivacyLevel = 1 + iota
	StageChannelPrivacyLevelGuildOnly
)

// Channel represents a Discord channel.
type Channel struct {
	OwnerID                    *UserID              `json:"owner_id,omitempty"`
	GuildID                    *GuildID             `json:"guild_id,omitempty"`
	ThreadMember               *ThreadMember        `json:"member,omitempty"`
	ThreadMetadata             *ThreadMetadata      `json:"thread_metadata,omitempty"`
	LastPinTimestamp           *Timestamp           `json:"last_pin_timestamp"`
	ParentID                   *ChannelID           `json:"parent_id,omitempty"`
	ApplicationID              *ApplicationID       `json:"application_id,omitempty"`
	LastMessageID              *string              `json:"last_message_id"`
	RTCRegion                  string               `json:"rtc_region"`
	Topic                      string               `json:"topic"`
	Icon                       string               `json:"icon"`
	Name                       string               `json:"name"`
	PermissionOverwrites       ChannelOverwriteList `json:"permission_overwrites"`
	Recipients                 UserList             `json:"recipients"`
	Permissions                Int64                `json:"permissions"`
	ID                         ChannelID            `json:"id"`
	UserLimit                  int32                `json:"user_limit"`
	Bitrate                    int32                `json:"bitrate"`
	MessageCount               int32                `json:"message_count"`
	MemberCount                int32                `json:"member_count"`
	RateLimitPerUser           int32                `json:"rate_limit_per_user"`
	Position                   int32                `json:"position"`
	DefaultAutoArchiveDuration int32                `json:"default_auto_archive_duration"`
	VideoQualityMode           VideoQualityMode     `json:"video_quality_mode"`
	Type                       ChannelType          `json:"type"`
	NSFW                       bool                 `json:"nsfw"`
}

// ChannelOverwrite represents a permission overwrite for a channel.
type ChannelOverwrite struct {
	Type  ChannelOverrideType `json:"type"`
	ID    Snowflake           `json:"id"`
	Allow Int64               `json:"allow"`
	Deny  Int64               `json:"deny"`
}

// ChannelOverrideType represents the target of a channel override.
type ChannelOverrideType uint16

func (in *ChannelOverrideType) IsNil() bool {
	return *in == 0
}

func (in *ChannelOverrideType) UnmarshalJSON(b []byte) error {
	if !bytes.Equal(b, null) {
		// Discord will pass ChannelOverrideType as a string if it is in an audit log.
		if b[0] == '"' {
			i, err := strconv.ParseInt(string(b[1:len(b)-1]), 10, 64)
			if err != nil {
				return fmt.Errorf("failed to unmarshal json: %v", err)
			}

			*in = ChannelOverrideType(i)
		} else {
			i, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				return fmt.Errorf("failed to unmarshal json: %v", err)
			}

			*in = ChannelOverrideType(i)
		}
	}

	return nil
}

func (in ChannelOverrideType) MarshalJSON() ([]byte, error) {
	return uint16Bytes(uint16(in)), nil
}

func (in ChannelOverrideType) String() string {
	return strconv.FormatInt(int64(in), 10)
}

const (
	ChannelOverrideTypeRole ChannelOverrideType = iota
	ChannelOverrideTypeMember
)

// ThreadMetadata contains thread-specific channel fields.
type ThreadMetadata struct {
	ArchiveTimestamp    Timestamp `json:"archive_timestamp"`
	AutoArchiveDuration int32     `json:"auto_archive_duration"`
	Archived            bool      `json:"archived"`
	Locked              bool      `json:"locked"`
}

// ThreadMember is used to indicate whether a user has joined a thread or not.
type ThreadMember struct {
	ID            *ChannelID `json:"id,omitempty"`
	UserID        *UserID    `json:"user_id,omitempty"`
	GuildID       *GuildID   `json:"guild_id,omitempty"`
	JoinTimestamp Timestamp  `json:"join_timestamp"`
	Flags         int32      `json:"flags"`
}

// StageInstance represents a stage channel instance.
type StageInstance struct {
	Topic                string                   `json:"topic"`
	ID                   StageInstanceID          `json:"id"`
	GuildID              GuildID                  `json:"guild_id"`
	ChannelID            ChannelID                `json:"channel_id"`
	PrivacyLabel         StageChannelPrivacyLevel `json:"privacy_level"`
	DiscoverableDisabled bool                     `json:"discoverable_disabled"`
}

// FollowedChannel represents a followed channel.
type FollowedChannel struct {
	ChannelID ChannelID `json:"channel_id"`
	WebhookID WebhookID `json:"webhook_id"`
}

// ChannelPermissionsParams represents the arguments to modify guild channel permissions.
type ChannelPermissionsParams struct {
	ID              ChannelID `json:"id"`
	Position        int32     `json:"position"`
	LockPermissions bool      `json:"lock_permissions"`
	ParentID        ChannelID `json:"parent_id"`
}
