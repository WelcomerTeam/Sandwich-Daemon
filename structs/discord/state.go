package structs

import (
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/vmihailenco/msgpack"
)

// StateGuild represents a guild in the state.
type StateGuild struct {
	Guild

	Roles   []*Role        `json:"-" msgpack:"-"`
	RoleIDs []snowflake.ID `json:"roles" msgpack:"roles"`

	Emojis   []*Emoji       `json:"-" msgpack:"-"`
	EmojiIDs []snowflake.ID `json:"emojis" msgpack:"emojis"`

	Channels   []*Channel     `json:"-" msgpack:"-"`
	ChannelIDs []snowflake.ID `json:"channels" msgpack:"channels"`
}

// FromGuild converts from a guild to a state guild.
func (sg *StateGuild) FromGuild(guild Guild) (
	roles map[string]interface{},
	emojis map[string]interface{},
	channels map[string]interface{}) {
	sg.Guild = guild

	var ma interface{}

	var err error

	roles = make(map[string]interface{})
	emojis = make(map[string]interface{})
	channels = make(map[string]interface{})

	for _, role := range guild.Roles {
		if ma, err = msgpack.Marshal(role); err == nil {
			roles[role.ID.String()] = ma

			sg.RoleIDs = append(sg.RoleIDs, role.ID)
		}
	}

	for _, emoji := range guild.Emojis {
		if ma, err = msgpack.Marshal(emoji); err == nil {
			emojis[emoji.ID.String()] = ma

			sg.EmojiIDs = append(sg.EmojiIDs, emoji.ID)
		}
	}

	for _, channel := range guild.Channels {
		if ma, err = msgpack.Marshal(channel); err == nil {
			channels[channel.ID.String()] = ma

			sg.ChannelIDs = append(sg.ChannelIDs, channel.ID)
		}
	}

	return roles, emojis, channels
}

// StateGuildMember represents a guild member in the state.
type StateGuildMember struct {
	GuildMember

	User snowflake.ID `json:"user" msgpack:"user"`
}

// FromDiscord converts a guild member to a state guild member.
func (sgm *StateGuildMember) FromGuildMember(member GuildMember) {
	sgm.User = member.User.ID
	sgm.Nick = member.Nick

	sgm.Roles = member.Roles
	sgm.JoinedAt = member.JoinedAt
	sgm.Deaf = member.Deaf
	sgm.Mute = member.Mute
}
