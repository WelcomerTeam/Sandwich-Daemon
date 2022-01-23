package discord

// invites.go contains all structures for invites.

// InviteTargetType represents the type of an invites target.
type InviteTargetType uint8

const (
	InviteTargetTypeStream InviteTargetType = 1 + iota
	InviteTargetTypeEmbeddedApplication
)

// EventStatus represents the status of an event.
type EventStatus uint8

const (
	EventStatusScheduled EventStatus = 1 + iota
	EventStatusActive
	EventStatusCompleted
	EventStatusCanceled
)

// ScheduledEntityType represents the type of event
type ScheduledEntityType uint8

const (
	ScheduledEntityTypeStage ScheduledEntityType = 1 + iota
	ScheduledEntityTypeVoice
	ScheduledEntityTypeExternal
)

// Invite represents the structure of Invite data.
type Invite struct {
	Code    string     `json:"code"`
	Guild   *Guild     `json:"guild,omitempty"`
	GuildID *Snowflake `json:"guild_id,omitempty"`
	Channel *Channel   `json:"channel,omitempty"`

	Inviter                  *User                `json:"inviter,omitempty"`
	TargetType               *InviteTargetType    `json:"target_type,omitempty"`
	TargetUser               *User                `json:"target_user,omitempty"`
	TargetApplication        *Application         `json:"target_application"`
	ApproximatePresenceCount int32                `json:"approximate_presence_count,omitempty"`
	ApproximateMemberCount   int32                `json:"approximate_member_count,omitempty"`
	ExpiresAt                string               `json:"expires_at,omitempty"`
	StageInstance            *InviteStageInstance `json:"stage_instance,omitempty"`
	ScheduledEvent           *ScheduledEvent      `json:"guild_scheduled_event,omitempty"`

	Uses      int32  `json:"uses"`
	MaxUses   int32  `json:"max_uses"`
	MaxAge    int32  `json:"max_age"`
	Temporary bool   `json:"temporary"`
	CreatedAt string `json:"created_at"`
}

// InviteStageInstance represents the structure of an invite stage instance.
type InviteStageInstance struct {
	Members          []*GuildMember `json:"members"`
	ParticipantCount int32          `json:"participant_count"`
	SpeakerCount     int32          `json:"speaker_count"`
	Topic            string         `json:"topic"`
}

// ScheduledEvent represents an scheduled event.
type ScheduledEvent struct {
	ID                 Snowflake                 `json:"id"`
	GuildID            Snowflake                 `json:"guild_id"`
	ChannelID          *Snowflake                `json:"channel_id,omitempty"`
	CreatorID          *Snowflake                `json:"creator_id,omitempty"`
	Name               string                    `json:"name"`
	Description        string                    `json:"description,omitempty"`
	ScheduledStartTime string                    `json:"scheduled_start_time"`
	ScheduledEndTime   string                    `json:"scheduled_end_time"`
	PrivacyLevel       *StageChannelPrivacyLevel `json:"privacy_level"`
	Status             *EventStatus              `json:"status"`
	EntityType         *ScheduledEntityType      `json:"entity_type"`
	EntityID           *Snowflake                `json:"entity_id,omitempty"`
	EntityMetadata     *EventMetadata            `json:"entity_metadata,omitempty"`
	Creator            *User                     `json:"creator,omitempty"`
	UserCount          int32                     `json:"user_count,omitempty"`
}

// EventMetadata contains extra information about a scheduled event.
type EventMetadata struct {
	Location string `json:"location,omitempty"`
}

// ScheduledEventUser represents a user subscribed to an event.
type ScheduledEventUser struct {
	EventID Snowflake    `json:"guild_scheduled_event_id"`
	User    User         `json:"user"`
	Member  *GuildMember `json:"member,omitempty"`
}
