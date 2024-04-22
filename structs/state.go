package structs

import (
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	jsoniter "github.com/json-iterator/go"
)

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  jsoniter.RawMessage
	Extra map[string]jsoniter.RawMessage
}

type StateDMChannel struct {
	*discord.Channel

	ExpiresAt discord.Int64 `json:"expires_at"`
}

type StateMutualGuilds struct {
	GuildsMu sync.RWMutex `json:"-"`

	Guilds map[discord.Snowflake]bool `json:"guilds"`
}

type StateGuildMembers struct {
	MembersMu sync.RWMutex `json:"-"`

	Members map[discord.Snowflake]*discord.GuildMember `json:"members"`
}

type StateGuildRoles struct {
	RolesMu sync.RWMutex `json:"-"`

	Roles map[discord.Snowflake]*discord.Role `json:"roles"`
}

type StateGuildEmojis struct {
	EmojisMu sync.RWMutex `json:"-"`

	Emojis map[discord.Snowflake]*discord.Emoji `json:"emoji"`
}

type StateGuildChannels struct {
	ChannelsMu sync.RWMutex `json:"-"`

	Channels map[discord.Snowflake]*discord.Channel `json:"channels"`
}

type StateGuildVoiceStates struct {
	VoiceStatesMu sync.RWMutex `json:"-"`

	VoiceStates map[discord.Snowflake]*discord.VoiceState `json:"voice_states"`
}

type StateUser struct {
	*discord.User

	LastUpdated time.Time `json:"__sandwich_last_updated,omitempty"`
}
