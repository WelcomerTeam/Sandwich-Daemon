package structs

import (
	"sync"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
)

// StateGuild represents a guild in the state.
type StateGuild struct {
	*Guild

	Roles   []*Role        `json:"-" msgpack:"-"`
	RoleIDs []snowflake.ID `json:"roles" msgpack:"roles"`

	Emojis   []*Emoji       `json:"-" msgpack:"-"`
	EmojiIDs []snowflake.ID `json:"emojis" msgpack:"emojis"`

	Channels   []*Channel     `json:"-" msgpack:"-"`
	ChannelIDs []snowflake.ID `json:"channels" msgpack:"channels"`
}

// StateGuildMembers stores the Members for a Guild along with
// the appropriate lock.
type StateGuildMembers struct {
	GuildID snowflake.ID `json:"id"`

	MembersMu sync.RWMutex                       `json:"-"`
	Members   map[snowflake.ID]*StateGuildMember `json:"-"`
}

// StateGuildMember represents a guild member in the state.
type StateGuildMember struct {
	GuildMember

	User snowflake.ID `json:"user" msgpack:"user"`
}

// FromDiscord converts a guild member to a state guild member.
func FromGuildMember(member *GuildMember) (sgm *StateGuildMember) {
	sgm.User = member.User.ID
	sgm.Nick = member.Nick
	sgm.Roles = member.Roles
	sgm.JoinedAt = member.JoinedAt
	sgm.Deaf = member.Deaf
	sgm.Mute = member.Mute

	return sgm
}

func (sgm *StateGuildMember) ToGuildMember(u *User) (member *GuildMember) {
	member.User = u
	member.Nick = sgm.Nick
	member.Roles = sgm.Roles
	member.JoinedAt = sgm.JoinedAt
	member.Deaf = sgm.Deaf
	member.Mute = sgm.Mute

	return member
}
