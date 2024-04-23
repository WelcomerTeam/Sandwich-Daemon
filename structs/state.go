package structs

import (
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	jsoniter "github.com/json-iterator/go"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
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
	Guilds *csmap.CsMap[discord.Snowflake, bool] `json:"guilds"`
}

type StateGuildMembers struct {
	Members *csmap.CsMap[discord.Snowflake, *discord.GuildMember] `json:"members"`
}

type StateGuildRoles struct {
	Roles *csmap.CsMap[discord.Snowflake, *discord.Role] `json:"roles"`
}

type StateGuildEmojis struct {
	Emojis *csmap.CsMap[discord.Snowflake, *discord.Emoji] `json:"emoji"`
}

type StateGuildChannels struct {
	Channels *csmap.CsMap[discord.Snowflake, *discord.Channel] `json:"channels"`
}

type StateGuildVoiceStates struct {
	VoiceStates *csmap.CsMap[discord.Snowflake, *discord.VoiceState] `json:"voice_states"`
}

type StateUser struct {
	*discord.User

	LastUpdated time.Time `json:"__sandwich_last_updated,omitempty"`
}
