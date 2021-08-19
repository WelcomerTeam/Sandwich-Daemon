package discord

import (
	"time"

	"github.com/WelcomerTeam/RealRock/snowflake"
)

// events.go contains the structures of all received events from discord

// Hello represents a hello packet.
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

// ChannelCreate represents a channel create packet.
type ChannelCreate struct {
	*Channel
}

// ChannelUpdate represents a channel update packet.
type ChannelUpdate struct {
	*Channel
}

// ChannelDelete represents a channel delete packet.
type ChannelDelete struct {
	*Channel
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

// ThreadMembersUpdate represents a thread member update event.
type ThreadMembersUpdate struct {
	ID               snowflake.ID    `json:"id"`
	GuildID          snowflake.ID    `json:"guild_id"`
	MemberCount      int             `json:"member_count"`
	AddedMembers     []*ThreadMember `json:"added_members,omitempty"`
	RemovedMemberIDs []snowflake.ID  `json:"removed_member_ids,omitempty"`
}

// ChannelPinsUpdate represents a channel pins update event.
type ChannelPisnUpdate struct {
	GuildID          snowflake.ID `json:"guild_id"`
	ChannelID        snowflake.ID `json:"channel_id"`
	LastPinTimestamp time.Time    `json:"last_pin_timestamp,omitempty"`
}

// TODO:  GuildCreate
// TODO:  GuildUpdate
// TODO:  GuildDelete
// TODO:  GuildBanAdd
// TODO:  GuildBanRemove
// TODO:  GuildEmojisUpdate
// TODO:  GuildStickersUpdate
// TODO:  GuildIntegrationsUpdate
// TODO:  GuildMemberAdd
// TODO:  GuildMemberRemove
// TODO:  GuildMemberUpdate
// TODO:  GuildMembersChunk
// TODO:  GuildRoleCreate
// TODO:  GuildRoleUpdate
// TODO:  GuildRoleDelete
// TODO:  IntegrationCreate
// TODO:  IntegrationUpdate
// TODO:  IntegrationDelete
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
// TODO:  TypingStart
// TODO:  UserUpdate
// TODO:  VoiceStateUpdate
// TODO:  VoiceServerUpdate
// TODO:  WebhooksUpdate
// TODO:  ApplicationCommandCreate
// TODO:  ApplicationCommandUpdate
// TODO:  ApplicationCommandDelete
// TODO:  InteractionCreate
// TODO:  StageInstanceCreate
// TODO:  StageInstanceUpdate
// TODO:  StageInstanceDelete
