package gateway

import (
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
)

// StateGuild represents a guild in the state
type StateGuild struct {
	ID                          snowflake.ID                       `json:"id"`
	Name                        string                             `json:"name"`
	Icon                        string                             `json:"icon"`
	Splash                      string                             `json:"splash"`
	Owner                       bool                               `json:"owner,omitempty"`
	OwnerID                     snowflake.ID                       `json:"owner_id"`
	Permissions                 int                                `json:"permissions,omitempty"`
	Region                      string                             `json:"region"`
	AFKChannelID                snowflake.ID                       `json:"afk_channel_id"`
	AFKTimeout                  int                                `json:"afk_timeout"`
	EmbedEnabled                bool                               `json:"embed_enabled,omitempty"`
	EmbedChannelID              snowflake.ID                       `json:"embed_channel_id,omitempty"`
	VerificationLevel           structs.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications structs.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       structs.ExplicitContentFilterLevel `json:"explicit_content_filter"`
	Roles                       []snowflake.ID                     `json:"roles"`
	Emojis                      []snowflake.ID                     `json:"emojis"`
	Features                    []string                           `json:"features"`
	MFALevel                    structs.MFALevel                   `json:"mfa_level"`
	ApplicationID               snowflake.ID                       `json:"application_id"`
	WidgetEnabled               bool                               `json:"widget_enabled,omitempty"`
	WidgetChannelID             snowflake.ID                       `json:"widget_channel_id,omitempty"`
	SystemChannelID             snowflake.ID                       `json:"system_channel_id"`
	JoinedAt                    string                             `json:"joined_at,omitempty"`
	Large                       bool                               `json:"large,omitempty"`
	Unavailable                 bool                               `json:"unavailable,omitempty"`
	MemberCount                 int                                `json:"member_count,omitempty"`
	VoiceStates                 []*structs.VoiceState              `json:"voice_states,omitempty"`
	Channels                    []snowflake.ID                     `json:"channels,omitempty"`
	Presences                   []*structs.Activity                `json:"presences,omitempty"`
}

// FromDiscord converts the discord object into the StateGuild form and returns appropriate maps
func (sg *StateGuild) FromDiscord(guild structs.Guild) (
	roles map[snowflake.ID]*structs.Role,
	emojis map[snowflake.ID]*structs.Emoji,
	channels map[snowflake.ID]*structs.Channel) {

	// Im sorry for commiting war crimes
	sg.ID = guild.ID
	sg.Name = guild.Name
	sg.Icon = guild.Icon
	sg.Splash = guild.Splash
	sg.Owner = guild.Owner
	sg.OwnerID = guild.OwnerID
	sg.Permissions = guild.Permissions
	sg.Region = guild.Region
	sg.AFKChannelID = guild.AFKChannelID
	sg.AFKTimeout = guild.AFKTimeout
	sg.EmbedEnabled = guild.EmbedEnabled
	sg.EmbedChannelID = guild.EmbedChannelID
	sg.VerificationLevel = guild.VerificationLevel
	sg.DefaultMessageNotifications = guild.DefaultMessageNotifications
	sg.ExplicitContentFilter = guild.ExplicitContentFilter
	sg.Features = guild.Features
	sg.MFALevel = guild.MFALevel
	sg.ApplicationID = guild.ApplicationID
	sg.WidgetEnabled = guild.WidgetEnabled
	sg.WidgetChannelID = guild.WidgetChannelID
	sg.SystemChannelID = guild.SystemChannelID
	sg.JoinedAt = guild.JoinedAt
	sg.Large = guild.Large
	sg.Unavailable = guild.Unavailable
	sg.MemberCount = guild.MemberCount
	sg.VoiceStates = guild.VoiceStates
	sg.Presences = guild.Presences

	for _, role := range guild.Roles {
		roles[role.ID] = role
		sg.Roles = append(sg.Roles, role.ID)
	}

	for _, emoji := range guild.Emojis {
		emojis[emoji.ID] = emoji
		sg.Emojis = append(sg.Emojis, emoji.ID)
	}

	for _, channel := range guild.Channels {
		channels[channel.ID] = channel
		sg.Channels = append(sg.Channels, channel.ID)
	}

	return
}

// StateGuildMember represents a guild member in the state
type StateGuildMember struct {
	User     snowflake.ID   `json:"user"`
	Nick     string         `json:"nick,omitempty"`
	Roles    []snowflake.ID `json:"roles"`
	JoinedAt string         `json:"joined_at"`
	Deaf     bool           `json:"deaf"`
	Mute     bool           `json:"mute"`
}

// FromDiscord converts from the discord object into the StateGuild form and returns the user object
func (sgm *StateGuildMember) FromDiscord(member structs.GuildMember) (user *structs.User) {
	sgm.User = member.User.ID
	sgm.Nick = member.Nick
	sgm.Roles = member.Roles
	sgm.JoinedAt = member.JoinedAt
	sgm.Deaf = member.Deaf
	sgm.Mute = member.Mute

	return member.User
}

// StateGuildMembersChunk handles the GUILD_MEMBERS_CHUNK event
func (mg *Manager) StateGuildMembersChunk(packet structs.GuildMembersChunk) (err error) {

	println(packet.GuildID, len(packet.Members), packet.ChunkIndex, "/", packet.ChunkCount, packet.NotFound, len(packet.NotFound))

	// STORE_MUTUALS = KEYS[1]
	// if STORE_MUTUALS do
	// 	for i,k in pairs(ARGV) do
	// 		redis.call("HSET", "welcomer:guild:<>:members", k.ID, k)
	// 	end
	// else
	// 	for i,k in pairs(ARGV) do
	// 		redis.call("HSET", "welcomer:guild:<>:members", k.ID, k)
	// 	end
	// end

	if !mg.Configuration.Caching.CacheMembers {
		return
	}

	err = mg.RedisClient.Eval(
		mg.ctx,
		`for i,k in pairs(ARGV) do redis.log(redis.LOG_WARNING, k) end`,
		nil,
		packet.Members,
	).Err()

	return
}

// StateGuildCreate handles the GUILD_CREATE event
func (mg *Manager) StateGuildCreate(packet structs.GuildCreate) (err error) {

	err = mg.RedisClient.Eval(
		mg.ctx,
		``,
		nil,
		packet.Members,
	).Err()

	return
}
