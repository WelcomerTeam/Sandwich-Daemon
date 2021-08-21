package discord

import (
	"time"

	"github.com/WelcomerTeam/RealRock/snowflake"
)

// events.go contains the structures of all received events from discord

// Empty structure
type void struct{}

// Hello represents a hello event when connecting.
type Hello struct {
	HeartbeatInterval time.Duration `json:"heartbeat_interal"`
}

// Ready represents when the client has completed the initial handshake.
type Ready struct {
	Version     int            `json:"version"`
	User        User           `json:"user"`
	Guilds      []PartialGuild `json:"guilds"`
	SessionID   string         `json:"session_id"`
	Shard       []int          `json:"shard,omitempty"`
	Application Application    `json:"application"`
}

// Resumed represents the response to a resume event.
type Resumed void

// Reconnect represents the reconnect event.
type Reconnect void

// Invalid Session represents the invalid session event.
type InvalidSession struct {
	Resumable bool `json:"d"`
}

// ApplicationCommandCreate represents the application command create event.
type ApplicationCommandCreate *ApplicationCommand

// ApplicationCommandUpdate represents the application command update event.
type ApplicationCommandUpdate *ApplicationCommand

// ApplicationCommandDelete represents the application command delete event.
type ApplicationCommandDelete *ApplicationCommand

// ChannelCreate represents a channel create event.
type ChannelCreate struct {
	*Channel
}

// ChannelUpdate represents a channel update event.
type ChannelUpdate struct {
	*Channel
}

// ChannelDelete represents a channel delete event.
type ChannelDelete struct {
	*Channel
}

// ChannelPinsUpdate represents a channel pins update event.
type ChannelPisnUpdate struct {
	GuildID          snowflake.ID `json:"guild_id"`
	ChannelID        snowflake.ID `json:"channel_id"`
	LastPinTimestamp *time.Time   `json:"last_pin_timestamp,omitempty"`
}

// ThreadCreate represents a thread create event.
type ThreadCreate struct {
	*Channel
}

// ThreadUpdate represents a thread update event.
type ThreadUpdate struct {
	*Channel
}

// ThreadDelete represents a thread delete event.
type ThreadDelete struct {
	*Channel
}

// ThreadListSync represents a thread list sync event.
type ThreadListSync struct {
	GuildID    snowflake.ID    `json:"guild_id"`
	ChannelIDs []snowflake.ID  `json:"channel_ids,omitempty"`
	Threads    []*Channel      `json:"threads"`
	Members    []*ThreadMember `json:"members"`
}

// ThreadMemberUpdate represents a thread member update event.
type ThreadMemberUpdate *ThreadMember

// ThreadMembersUpdate represents a thread members update event.
type ThreadMembersUpdate struct {
	ID               snowflake.ID    `json:"id"`
	GuildID          snowflake.ID    `json:"guild_id"`
	MemberCount      int             `json:"member_count"`
	AddedMembers     []*ThreadMember `json:"added_members,omitempty"`
	RemovedMemberIDs []snowflake.ID  `json:"removed_member_ids,omitempty"`
}

// GuildCreate represents a guild create event.
type GuildCreate struct {
	*Guild
	Lazy bool `json:"-"` // Internal use.
}

// GuildUpdate represents a guild update event.
type GuildUpdate *Guild

// GuildDelete represents a guild delete event.
type GuildDelete *PartialGuild

// GuildBanAdd represents a guild ban add event.
type GuildBanAdd struct {
	GuildID snowflake.ID `json:"guild_id"`
	User    *User        `json:"user"`
}

// GuildBanRemove represents a guild ban remove event.
type GuildBanRemove struct {
	GuildID snowflake.ID `json:"guild_id"`
	User    *User        `json:"user"`
}

// GuildEmojisUpdate represents a guild emojis update event.
type GuildEmojisUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Emojis  []*Emoji     `json:"emojis"`
}

// GuildStickersUpdate represents a guild stickers update event.
type GuildStickersUpdate struct {
	GuildID  snowflake.ID `json:"guild_id"`
	Stickers []*Sticker   `json:"stickers"`
}

// GuildIntegrationsUpdate represents a guild integrations update event.
type GuildIntegrationsUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
}

// GuildMemberAdd represents a guild member add event.
type GuildMemberAdd struct {
	*GuildMember
	GuildID snowflake.ID `json:"guild_id"`
}

// GuildMemberRemove represents a guild member remove event.
type GuildMemberRemove struct {
	GuildID snowflake.ID `json:"guild_id"`
	User    *User        `json:"user"`
}

// GuildMemberUpdate represents a guild member update event.
type GuildMemberUpdate struct {
	GuildID      snowflake.ID   `json:"guild_id"`
	Roles        []snowflake.ID `json:"roles"`
	User         *User          `json:"user"`
	Nick         string         `json:"nick"`
	JoinedAt     time.Time      `json:"joined_at"`
	PremiumSince *time.Time     `json:"premium_since,omitempty"`
	Deaf         *bool          `json:"deaf,omitempty"`
	Mute         *bool          `json:"deaf,omitempty"`
	Pending      *bool          `json:"deaf,omitempty"`
}

// GuildMembersChunk represents a guild members chunk event.
type GuildMembersChunk struct {
	GuildID    snowflake.ID     `json:"guild_id"`
	Members    []*GuildMember   `json:"members"`
	ChunkIndex int              `json:"chunk_index"`
	ChunkCount int              `json:"chunk_count"`
	NotFound   []snowflake.ID   `json:"not_found,omitempty"`
	Presences  []PresenceStatus `json:"presences,omitempty"`
	Nonce      *string          `json:"nonce,omitempty"`
}

// GuildRoleCreate represents a guild role create event.
type GuildRoleCreate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Role    *Role        `json:"role"`
}

// GuildRoleUpdate represents a guild role update event.
type GuildRoleUpdate struct {
	GuildID snowflake.ID `json:"guild_id"`
	Role    *Role        `json:"role"` // TODO: type
}

// GuildRoleDelete represents a guild role delete event.
type GuildRoleDelete struct {
	GuildID snowflake.ID `json:"guild_id"`
	RoleID  snowflake.ID `json:"role_id"`
}

// IntegrationCreate represents the integration create event.
type IntegrationCreate struct {
	*Integration
	GuildID snowflake.ID `json:"guild_id"`
}

// IntegrationUpdate represents the integration update event.
type IntegrationUpdate struct {
	*Integration
	GuildID snowflake.ID `json:"guild_id"`
}

// IntegrationDelete represents the integration delete event.
type IntegrationDelete struct {
	ID            snowflake.ID  `json:"id"`
	GuildID       snowflake.ID  `json:"guild_id"`
	ApplicationID *snowflake.ID `json:"application_id"`
}

// InteractionCreate represents the interaction create event.
type InteractionCreate *Interaction

// InviteCreate represents the invite create event.
type InviteCreate struct {
	ChannelID         snowflake.ID        `json:"channel_id"`
	Code              string              `json:"code"`
	CreatedAt         time.Time           `json:"created_at"`
	GuildID           *snowflake.ID       `json:"guild_id,omitempty"`
	Inviter           *User               `json:"inviter,omitempty"`
	MaxAge            int                 `json:"max_age"`
	MaxUses           int                 `json:"max_uses"`
	TargetType        *InviteTargetType   `json:"target_type,omitempty"`
	TargetUser        *User               `json:"target_user,omitempty"`
	TargetApplication *PartialApplication `json:"target_application"`
	Temporary         bool                `json:"temporary"`
	Uses              int                 `json:"uses"`
}

// InviteDelete represents the invite delete event.
type InviteDelete struct {
	ChannelID snowflake.ID  `json:"channel_id"`
	GuildID   *snowflake.ID `json:"guild_id,omitempty"`
	Code      string        `json:"code"`
}

// MessageCreate represents the message update event.
type MessageUpdate *Message

// MessageCreate represents the message delete event.
type MessageDelete struct {
	ID        snowflake.ID  `json:"id"`
	ChannelID snowflake.ID  `json:"channel_id"`
	GuildID   *snowflake.ID `json:"guild_id,omitempty"`
}

// MessageCreate represents the message bulk delete event.
type MessageDeleteBulk struct {
	IDs       []snowflake.ID `json:"ids"`
	ChannelID snowflake.ID   `json:"channel_id"`
	GuildID   *snowflake.ID  `json:"guild_id,omitempty"`
}

// MessageReactionAdd represents a message reaction add event.
type MessageReactionAdd struct {
	UserID    snowflake.ID `json:"user_id"`
	ChannelID snowflake.ID `json:"channel_id"`
	MessageID snowflake.ID `json:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	Member    *GuildMember `json:"member,omitempty"`
	Emoji     *Emoji       `json:"emoji"`
}

// MessageReactionRemove represents a message reaction remove event.
type MessageReactionRemove struct {
	UserID    snowflake.ID  `json:"user_id"`
	ChannelID snowflake.ID  `json:"channel_id"`
	MessageID snowflake.ID  `json:"message_id"`
	GuildID   *snowflake.ID `json:"guild_id,omitempty"`
	Emoji     *Emoji        `json:"emoji"`
}

// MessageReactionRemoveAll represents a message reaction remove all event.
type MessageReactionRemoveAll struct {
	ChannelID snowflake.ID `json:"channel_id"`
	MessageID snowflake.ID `json:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
}

// MessageReactionRemoveEmoji represents a message reaction remove emoji event.
type MessageReactionRemoveEmoji struct {
	ChannelID snowflake.ID  `json:"channel_id"`
	GuildID   *snowflake.ID `json:"guild_id,omitempty"`
	MessageID snowflake.ID  `json:"message_id"`
	Emoji     *PartialEmoji `json:"emoji"`
}

// PresenceUpdate represents a presence update event.
type PresenceUpdate struct {
	User         *User          `json:"user"`
	GuildID      snowflake.ID   `json:"guild_id"`
	Status       PresenceStatus `json:"status"`
	Activities   []*Activity    `json:"activities"`
	ClientStatus *ClientStatus  `json:"clienbt_status"`
}

// StageInstanceCreate represents a stage instance create event.
type StageInstanceCreate *StageInstance

// StageInstanceUpdate represents a stage instance update event.
type StageInstanceUpdate *StageInstance

// StageInstanceDelete represents a stage instance delete event.
type StageInstanceDelete *StageInstance

// TypingStart represents a typing start event.
type TypingStart struct {
	ChannelID snowflake.ID  `json:"channel_id"`
	GuildID   *snowflake.ID `json:"guild_id,omitempty"`
	UserID    snowflake.ID  `json:"user_id"`
	Timestamp int           `json:"timestamp"`
	Member    *GuildMember  `json:"member,omitempty"`
}

// UserUpdate represents a user update event.
type UserUpdate *User

// VoiceStateUpdate represents the voice state update event.
type VoiceStateUpdate struct {
	GuildID                 snowflake.ID  `json:"guild_id"`
	ChannelID               *snowflake.ID `json:"channel_id,omitempty"`
	UserID                  snowflake.ID  `json:"user_id"`
	Member                  *GuildMember  `json:"member,omitempty"`
	SessionID               string        `json:"session_id"`
	Deaf                    bool          `json:"deaf"`
	Mute                    bool          `json:"mute"`
	SelfDeaf                bool          `json:"self_deaf"`
	SelfMute                bool          `json:"self_mute"`
	SelfStream              *bool         `json:"self_stream,omitempty"`
	SelfVideo               bool          `json:"self_video"`
	Suppress                bool          `json:"suppress"`
	RequestToSpeakTimestamp time.Time     `json:"request_to_speak_timestamp"`
}

// VoiceServerUpdate represents a voice server update event.
type VoiceServerUpdate struct {
	Token    string       `json:"token" msgpack:"token"`
	GuildID  snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Endpoint string       `json:"endpoint" msgpack:"endpoint"`
}

// WebhookUpdate represents a webhook update packet.
type WebhookUpdate struct {
	GuildID   snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
}
