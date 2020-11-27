package gateway

import (
	"fmt"
	"strconv"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/xerrors"
)

// StateGuild represents a guild in the state.
type StateGuild struct {
	ID                          snowflake.ID                       `json:"id" msgpack:"id"`
	OwnerID                     snowflake.ID                       `json:"owner_id" msgpack:"owner_id"`
	AFKChannelID                snowflake.ID                       `json:"afk_channel_id" msgpack:"afk_channel_id"`
	EmbedChannelID              snowflake.ID                       `json:"embed_channel_id,omitempty" msgpack:"embed_channel_id,omitempty"`
	Name                        string                             `json:"name" msgpack:"name"`
	Icon                        string                             `json:"icon" msgpack:"icon"`
	Splash                      string                             `json:"splash" msgpack:"splash"`
	Region                      string                             `json:"region" msgpack:"region"`
	Permissions                 int                                `json:"permissions,omitempty" msgpack:"permissions,omitempty"`
	AFKTimeout                  int                                `json:"afk_timeout" msgpack:"afk_timeout"`
	VerificationLevel           structs.VerificationLevel          `json:"verification_level" msgpack:"verification_level"`
	DefaultMessageNotifications structs.MessageNotificationLevel   `json:"default_message_notifications" msgpack:"default_message_notifications"`
	ExplicitContentFilter       structs.ExplicitContentFilterLevel `json:"explicit_content_filter" msgpack:"explicit_content_filter"`
	MFALevel                    structs.MFALevel                   `json:"mfa_level" msgpack:"mfa_level"`
	ApplicationID               snowflake.ID                       `json:"application_id" msgpack:"application_id"`
	WidgetChannelID             snowflake.ID                       `json:"widget_channel_id,omitempty" msgpack:"widget_channel_id,omitempty"`
	SystemChannelID             snowflake.ID                       `json:"system_channel_id" msgpack:"system_channel_id"`
	JoinedAt                    string                             `json:"joined_at,omitempty" msgpack:"joined_at,omitempty"`
	Owner                       bool                               `json:"owner,omitempty" msgpack:"owner,omitempty"`
	WidgetEnabled               bool                               `json:"widget_enabled,omitempty" msgpack:"widget_enabled,omitempty"`
	EmbedEnabled                bool                               `json:"embed_enabled,omitempty" msgpack:"embed_enabled,omitempty"`
	Large                       bool                               `json:"large,omitempty" msgpack:"large,omitempty"`
	Unavailable                 bool                               `json:"unavailable,omitempty" msgpack:"unavailable,omitempty"`
	MemberCount                 int                                `json:"member_count,omitempty" msgpack:"member_count,omitempty"`
	Roles                       []snowflake.ID                     `json:"roles" msgpack:"roles"`
	Emojis                      []snowflake.ID                     `json:"emojis" msgpack:"emojis"`
	Features                    []string                           `json:"features" msgpack:"features"`
	VoiceStates                 []*structs.VoiceState              `json:"voice_states,omitempty" msgpack:"voice_states,omitempty"`
	Channels                    []snowflake.ID                     `json:"channels,omitempty" msgpack:"channels,omitempty"`
	Presences                   []*structs.Activity                `json:"presences,omitempty" msgpack:"presences,omitempty"`
}

// FromDiscord converts the discord object into the StateGuild form and returns appropriate maps.
func (sg *StateGuild) FromDiscord(guild structs.Guild) (
	roles map[string]interface{},
	emojis map[string]interface{},
	channels map[string]interface{}) {
	// (
	// 	roles map[snowflake.ID]*structs.Role,
	// 	emojis map[snowflake.ID]*structs.Emoji,
	// 	channels map[snowflake.ID]*structs.Channel)
	// Im sorry for committing war crimes
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

	// roles = make(map[snowflake.ID]*structs.Role, 0)
	// emojis = make(map[snowflake.ID]*structs.Emoji, 0)
	// channels = make(map[snowflake.ID]*structs.Channel, 0)

	// for _, role := range guild.Roles {
	// 	roles[role.ID] = role
	// 	sg.Roles = append(sg.Roles, role.ID)
	// }

	// for _, emoji := range guild.Emojis {
	// 	emojis[emoji.ID] = emoji
	// 	sg.Emojis = append(sg.Emojis, emoji.ID)
	// }

	// for _, channel := range guild.Channels {
	// 	channels[channel.ID] = channel
	// 	sg.Channels = append(sg.Channels, channel.ID)
	// }

	var ma interface{}

	var err error

	roles = make(map[string]interface{})
	emojis = make(map[string]interface{})
	channels = make(map[string]interface{})

	for _, role := range guild.Roles {
		if ma, err = msgpack.Marshal(role); err == nil {
			roles[role.ID.String()] = ma

			sg.Roles = append(sg.Roles, role.ID)
		}
	}

	for _, emoji := range guild.Emojis {
		if ma, err = msgpack.Marshal(emoji); err == nil {
			emojis[emoji.ID.String()] = ma

			sg.Emojis = append(sg.Emojis, emoji.ID)
		}
	}

	for _, channel := range guild.Channels {
		if ma, err = msgpack.Marshal(channel); err == nil {
			channels[channel.ID.String()] = ma

			sg.Channels = append(sg.Channels, channel.ID)
		}
	}

	return roles, emojis, channels
}

// StateGuildMember represents a guild member in the state.
type StateGuildMember struct {
	User     snowflake.ID   `json:"user" msgpack:"user"`
	Nick     string         `json:"nick,omitempty" msgpack:"nick,omitempty"`
	Roles    []snowflake.ID `json:"roles" msgpack:"roles"`
	JoinedAt string         `json:"joined_at" msgpack:"joined_at"`
	Deaf     bool           `json:"deaf" msgpack:"deaf"`
	Mute     bool           `json:"mute" msgpack:"mute"`
}

// FromDiscord converts from the discord object into the StateGuild form and returns the user object.
func (sgm *StateGuildMember) FromDiscord(member structs.GuildMember) (user *structs.User) {
	sgm.User = member.User.ID
	sgm.Nick = member.Nick
	sgm.Roles = member.Roles
	sgm.JoinedAt = member.JoinedAt
	sgm.Deaf = member.Deaf
	sgm.Mute = member.Mute

	return member.User
}

// CreateKey creates a redis key from a format and values.
func (mg *Manager) CreateKey(key string, values ...interface{}) string {
	return mg.Configuration.Caching.RedisPrefix + ":" + fmt.Sprintf(key, values...)
}

// StateGuildMembersChunk handles the GUILD_MEMBERS_CHUNK event.
func (mg *Manager) StateGuildMembersChunk(packet structs.GuildMembersChunk) (err error) {
	if !mg.Configuration.Caching.CacheMembers {
		return
	}

	members := make([]interface{}, 0, len(packet.Members))

	for _, member := range packet.Members {
		if ma, err := msgpack.Marshal(member); err == nil {
			members = append(members, ma)
		}
	}

	err = mg.RedisClient.Eval(
		mg.ctx,
		`
		local redisPrefix = KEYS[1]
		local guildID = KEYS[2]
		local storeMutuals = KEYS[3] == true
		local cacheUsers = KEYS[4] == true

		local member
		local user

		local call = redis.call

		redis.log(3, "Received " .. #ARGV .. " member(s) in GuildMembersChunk")

		for i,k in pairs(ARGV) do
				member = cmsgpack.unpack(k)

				-- We do not want the user object stored in the member
				local user = member['user']
				user['id'] = string.format("%.0f",user['id'])

				member['user'] = nil
				member['id'] = user['id']

				redis.log(3, user['id'], type(user['id']), )

				if cacheUsers then
						redis.call("HSET", redisPrefix .. ":user", user['id'], cmsgpack.pack(user))
				end

				call("HSET", redisPrefix .. ":guild:" .. guildID .. ":members", user['id'], cmsgpack.pack(member))

				if storeMutuals then
						call("SADD", redisPrefix .. ":mutual:" .. user['id'], guildID)
				end

		end
		`,
		[]string{
			mg.Configuration.Caching.RedisPrefix,
			packet.GuildID.String(),
			strconv.FormatBool(mg.Configuration.Caching.StoreMutuals),
			strconv.FormatBool(mg.Configuration.Caching.CacheUsers),
		},
		members,
	).Err()

	if err != nil {
		return xerrors.Errorf("Failed to process guild member chunks: %w", err)
	}

	return nil
}

// StateGuildCreate handles the GUILD_CREATE event.
func (mg *Manager) StateGuildCreate(packet structs.GuildCreate) (ok bool, err error) {
	var k []byte

	sg := StateGuild{}

	roles, emojis, channels := sg.FromDiscord(packet.Guild)

	if k, err = msgpack.Marshal(sg); err == nil {
		err = mg.RedisClient.HSet(mg.ctx, mg.CreateKey("guilds"), sg.ID, k).Err()
		if err != nil {
			mg.Logger.Error().Err(err).Msg("Failed to push guild to redis")
		}
	}

	if len(roles) > 0 {
		err = mg.RedisClient.HSet(mg.ctx, mg.CreateKey("guild:%s:roles", sg.ID), roles).Err()
		if err != nil {
			mg.Logger.Error().Err(err).Msg("Failed to push guild roles to redis")
		}
	}

	if len(emojis) > 0 {
		err = mg.RedisClient.HSet(mg.ctx, mg.CreateKey("emojis"), emojis).Err()
		if err != nil {
			mg.Logger.Error().Err(err).Msg("Failed to push guild emojis to redis")
		}
	}

	if len(channels) > 0 {
		err = mg.RedisClient.HSet(mg.ctx, mg.CreateKey("channels"), channels).Err()
		if err != nil {
			mg.Logger.Error().Err(err).Msg("Failed to push guild channels to redis")
		}
	}

	// ok, err = mg.StoreInterfaceKey(sg, "guilds", sg.ID)
	// println("guilds", ok, err.Error())

	// ok, err = mg.StoreInterface(roles, "guilds:%s:roles", sg.ID)
	// println("roles", len(roles), ok, err.Error())

	// ok, err = mg.StoreInterface(emojis, "emojis")
	// println("emojis", len(emojis), ok, err.Error())

	// ok, err = mg.StoreInterface(channels, "channels")
	// println("channels", len(channels), ok, err.Error())

	return ok, err
}
