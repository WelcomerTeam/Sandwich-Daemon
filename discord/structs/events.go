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
type InteractionCreate Interaction

// TODO:  InteractionCreate

// TODO:  InviteCreate
// TODO:  InviteDelete

// TODO:  MessageCreate
// TODO:  MessageUpdate
// TODO:  MessageDelete
// TODO:  MessageDeleteBulk

// TODO:  MessageReactionAdd
// TODO:  MessageReactionRemove
// TODO:  MessageReactionRemoveAll
// TODO:  MessageReactionRemoveEmoji

// TODO:  PresenceUpdate

// TODO:  StageInstanceCreate
// TODO:  StageInstanceUpdate
// TODO:  StageInstanceDelete

// TODO:  TypingStart
// TODO:  UserUpdate

// TODO:  VoiceStateUpdate
// TODO:  VoiceServerUpdate
// TODO:  WebhooksUpdate
