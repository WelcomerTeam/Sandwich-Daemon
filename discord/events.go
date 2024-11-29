package discord

// events.go contains the structures of all received events from discord

// Empty structure.
type void struct{}

// Hello represents a hello event when connecting.
type Hello struct {
	HeartbeatInterval int32 `json:"heartbeat_interval"`
}

// Ready represents when the client has completed the initial handshake.
type Ready struct {
	SessionID        string               `json:"session_id"`
	ResumeGatewayUrl string               `json:"resume_gateway_url"`
	Guilds           UnavailableGuildList `json:"guilds"`
	Shard            []int32              `json:"shard,omitempty"`
	Application      Application          `json:"application"`
	User             User                 `json:"user"`
	Version          int32                `json:"v"`
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
type ApplicationCommandCreate ApplicationCommand

// ApplicationCommandUpdate represents the application command update event.
type ApplicationCommandUpdate ApplicationCommand

// ApplicationCommandDelete represents the application command delete event.
type ApplicationCommandDelete ApplicationCommand

// ChannelCreate represents a channel create event.
type ChannelCreate Channel

// ChannelUpdate represents a channel update event.
type ChannelUpdate Channel

// ChannelDelete represents a channel delete event.
type ChannelDelete Channel

// ChannelPinsUpdate represents a channel pins update event.
type ChannelPinsUpdate struct {
	LastPinTimestamp Timestamp `json:"last_pin_timestamp"`
	GuildID          GuildID   `json:"guild_id"`
	ChannelID        ChannelID `json:"channel_id"`
}

// ThreadCreate represents a thread create event.
type ThreadCreate Channel

// ThreadUpdate represents a thread update event.
type ThreadUpdate Channel

// ThreadDelete represents a thread delete event.
type ThreadDelete Channel

// ThreadListSync represents a thread list sync event.
type ThreadListSync struct {
	ChannelIDs ChannelIDList    `json:"channel_ids,omitempty"`
	Threads    ChannelList      `json:"threads"`
	Members    ThreadMemberList `json:"members"`
	GuildID    GuildID          `json:"guild_id"`
}

// ThreadMemberUpdate represents a thread member update event.
type ThreadMemberUpdate ThreadMember

// ThreadMembersUpdate represents a thread members update event.
type ThreadMembersUpdate struct {
	AddedMembers     ThreadMemberList `json:"added_members,omitempty"`
	RemovedMemberIDs UserIDList       `json:"removed_member_ids,omitempty"`
	ID               ChannelID        `json:"id"`
	GuildID          GuildID          `json:"guild_id"`
	MemberCount      int32            `json:"member_count"`
}

// GuildAuditLogEntryCreate represents when a guild audit log entry is created.
type GuildAuditLogEntryCreate struct {
	AuditLogEntry
	GuildID GuildID `json:"guild_id"`
}

// GuildCreate represents a guild create event.
type GuildCreate Guild

// GuildUpdate represents a guild update event.
type GuildUpdate Guild

// GuildDelete represents a guild delete event.
type GuildDelete UnavailableGuild

// GuildBanAdd represents a guild ban add event.
type GuildBanAdd GuildBan

// GuildBanRemove represents a guild ban remove event.
type GuildBanRemove GuildBan

// GuildEmojisUpdate represents a guild emojis update event.
type GuildEmojisUpdate struct {
	Emojis  EmojiList `json:"emojis"`
	GuildID GuildID   `json:"guild_id"`
}

// GuildStickersUpdate represents a guild stickers update event.
type GuildStickersUpdate struct {
	Stickers StickerList `json:"stickers"`
	GuildID  GuildID     `json:"guild_id"`
}

// GuildIntegrationsUpdate represents a guild integrations update event.
type GuildIntegrationsUpdate struct {
	GuildID GuildID `json:"guild_id"`
}

// GuildMemberAdd represents a guild member add event.
type GuildMemberAdd GuildMember

// GuildMemberRemove represents a guild member remove event.
type GuildMemberRemove struct {
	User    User    `json:"user"`
	GuildID GuildID `json:"guild_id"`
}

// GuildMemberUpdate represents a guild member update event.
type GuildMemberUpdate GuildMember

// GuildMembersChunk represents a guild members chunk event.
type GuildMembersChunk struct {
	Nonce      string             `json:"nonce"`
	Members    GuildMemberList    `json:"members"`
	NotFound   UserIDList         `json:"not_found,omitempty"`
	Presences  PresenceUpdateList `json:"presences,omitempty"`
	GuildID    GuildID            `json:"guild_id"`
	ChunkIndex int32              `json:"chunk_index"`
	ChunkCount int32              `json:"chunk_count"`
}

// GuildRoleCreate represents a guild role create event.
type GuildRoleCreate Role

// GuildRoleUpdate represents a guild role update event.
type GuildRoleUpdate struct {
	Role    Role    `json:"role"`
	GuildID GuildID `json:"guild_id"`
}

// GuildRoleDelete represents a guild role delete event.
type GuildRoleDelete struct {
	GuildID GuildID `json:"guild_id"`
	RoleID  RoleID  `json:"role_id"`
}

// IntegrationCreate represents the integration create event.
type IntegrationCreate Integration

// IntegrationUpdate represents the integration update event.
type IntegrationUpdate Integration

// IntegrationDelete represents the integration delete event.
type IntegrationDelete struct {
	ApplicationID ApplicationID `json:"application_id"`
	ID            IntegrationID `json:"id"`
	GuildID       GuildID       `json:"guild_id"`
}

// InteractionCreate represents the interaction create event.
type InteractionCreate Interaction

// InviteCreate represents the invite create event.
type InviteCreate Invite

// InviteDelete represents the invite delete event.
type InviteDelete Invite

// MessageCreate represents the message create event.
type MessageCreate Message

// MessageUpdate represents the message update event.
type MessageUpdate Message

// MessageCreate represents the message delete event.
type MessageDelete struct {
	GuildID   *GuildID  `json:"guild_id,omitempty"`
	ID        MessageID `json:"id"`
	ChannelID ChannelID `json:"channel_id"`
}

// MessageCreate represents the message bulk delete event.
type MessageDeleteBulk struct {
	GuildID   *GuildID      `json:"guild_id,omitempty"`
	IDs       MessageIDList `json:"ids"`
	ChannelID ChannelID     `json:"channel_id"`
}

// MessageReactionAdd represents a message reaction add event.
type MessageReactionAdd struct {
	Member          *GuildMember `json:"member,omitempty"`
	MessageAuthorID *UserID      `json:"message_author_id,omitempty"`
	BurstColors     StringList   `json:"burst_colors,omitempty"`
	Emoji           Emoji        `json:"emoji"`
	UserID          UserID       `json:"user_id"`
	ChannelID       ChannelID    `json:"channel_id"`
	MessageID       MessageID    `json:"message_id"`
	GuildID         GuildID      `json:"guild_id,omitempty"`
	Type            ReactionType `json:"type"`
	Burst           bool         `json:"burst"`
}

// MessageReactionRemove represents a message reaction remove event.
type MessageReactionRemove struct {
	GuildID   *GuildID     `json:"guild_id,omitempty"`
	Emoji     Emoji        `json:"emoji"`
	Type      ReactionType `json:"type"`
	UserID    UserID       `json:"user_id"`
	ChannelID ChannelID    `json:"channel_id"`
	MessageID MessageID    `json:"message_id"`
	Burst     bool         `json:"burst"`
}

// MessageReactionRemoveAll represents a message reaction remove all event.
type MessageReactionRemoveAll struct {
	ChannelID ChannelID `json:"channel_id"`
	MessageID MessageID `json:"message_id"`
	GuildID   GuildID   `json:"guild_id,omitempty"`
}

// MessageReactionRemoveEmoji represents a message reaction remove emoji event.
type MessageReactionRemoveEmoji struct {
	GuildID   *GuildID  `json:"guild_id,omitempty"`
	Emoji     Emoji     `json:"emoji"`
	ChannelID ChannelID `json:"channel_id"`
	MessageID MessageID `json:"message_id"`
}

// PresenceUpdate represents a presence update event.
type PresenceUpdate struct {
	ClientStatus ClientStatus   `json:"client_status"`
	Status       PresenceStatus `json:"status"`
	Activities   ActivityList   `json:"activities"`
	User         User           `json:"user"`
	GuildID      GuildID        `json:"guild_id"`
}

// StageInstanceCreate represents a stage instance create event.
type StageInstanceCreate StageInstance

// StageInstanceUpdate represents a stage instance update event.
type StageInstanceUpdate StageInstance

// StageInstanceDelete represents a stage instance delete event.
type StageInstanceDelete StageInstance

// TypingStart represents a typing start event.
type TypingStart struct {
	GuildID   *GuildID     `json:"guild_id,omitempty"`
	Member    *GuildMember `json:"member,omitempty"`
	ChannelID ChannelID    `json:"channel_id"`
	UserID    UserID       `json:"user_id"`
	Timestamp int32        `json:"timestamp"`
}

// UserUpdate represents a user update event.
type UserUpdate User

// VoiceStateUpdate represents the voice state update event.
type VoiceStateUpdate VoiceState

// VoiceServerUpdate represents a voice server update event.
type VoiceServerUpdate struct {
	Token    string  `json:"token"`
	Endpoint string  `json:"endpoint"`
	GuildID  GuildID `json:"guild_id"`
}

// WebhookUpdate represents a webhook update packet.
type WebhookUpdate struct {
	ChannelID ChannelID `json:"channel_id"`
	GuildID   GuildID   `json:"guild_id"`
}

// GuildJoinRequestDelete represents a guild join request delete event.
type GuildJoinRequestDelete struct {
	UserID  UserID  `json:"user_id"`
	GuildID GuildID `json:"guild_id"`
}
