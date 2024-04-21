package structs

import (
	"sync"

	"github.com/WelcomerTeam/Discord/discord"
	jsoniter "github.com/json-iterator/go"
)

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data             jsoniter.RawMessage
	Extra            map[string]jsoniter.RawMessage
	KeepOriginalData bool // Whether we should keep the original data in the payload.
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

	Emojis map[discord.Snowflake]*StateEmoji `json:"emoji"`
}

type StateGuildChannels struct {
	ChannelsMu sync.RWMutex `json:"-"`

	Channels map[discord.Snowflake]*discord.Channel `json:"channels"`
}

type StateGuildVoiceStates struct {
	VoiceStatesMu sync.RWMutex `json:"-"`

	VoiceStates map[discord.Snowflake]*StateVoiceState `json:"voice_states"`
}

type StateEmoji struct {
	ID            discord.Snowflake     `json:"id"`
	GuildID       discord.Snowflake     `json:"guild_id"`
	Name          string                `json:"name,omitempty"`
	Roles         discord.SnowflakeList `json:"roles,omitempty"`
	UserID        discord.Snowflake     `json:"user"`
	RequireColons bool                  `json:"require_colons"`
	Managed       bool                  `json:"managed"`
	Animated      bool                  `json:"animated"`
	Available     bool                  `json:"available"`
}

type StateUser struct {
	ID            discord.Snowflake       `json:"id"`
	Username      string                  `json:"username,omitempty"`
	Discriminator string                  `json:"discriminator,omitempty"`
	GlobalName    string                  `json:"global_name,omitempty"`
	Avatar        string                  `json:"avatar,omitempty"`
	Bot           bool                    `json:"bot"`
	System        bool                    `json:"system,omitempty"`
	MFAEnabled    bool                    `json:"mfa_enabled,omitempty"`
	Banner        string                  `json:"banner,omitempty"`
	AccentColour  int32                   `json:"accent_color"`
	Locale        string                  `json:"locale,omitempty"`
	Verified      bool                    `json:"verified,omitempty"`
	Email         string                  `json:"email,omitempty"`
	Flags         discord.UserFlags       `json:"flags,omitempty"`
	PremiumType   discord.UserPremiumType `json:"premium_type,omitempty"`
	PublicFlags   discord.UserFlags       `json:"public_flags,omitempty"`
	DMChannelID   *discord.Snowflake      `json:"dm_channel_id,omitempty"`
}

type StateVoiceState struct {
	ChannelID               discord.Snowflake  `json:"channel_id"`
	SessionID               string             `json:"session_id"`
	Deaf                    bool               `json:"deaf"`
	Mute                    bool               `json:"mute"`
	SelfDeaf                bool               `json:"self_deaf"`
	SelfMute                bool               `json:"self_mute"`
	SelfStream              bool               `json:"self_stream"`
	SelfVideo               bool               `json:"self_video"`
	Suppress                bool               `json:"suppress"`
	RequestToSpeakTimestamp *discord.Timestamp `json:"request_to_speak_timestamp,omitempty"`
}
