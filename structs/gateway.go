package structs

import (
	"encoding/json"
	"time"

	"github.com/bwmarrin/snowflake"
)

// Gateway represents a GET /gateway response
type Gateway struct {
	URL string `json:"url"`
}

// GatewayBot represents a GET /gateway/bot response
type GatewayBot struct {
	URL               string `json:"url"`
	Shards            int    `json:"shards"`
	SessionStartLimit struct {
		Total          int `json:"total"`
		Remaining      int `json:"remaining"`
		ResetAfter     int `json:"reset_after"`
		MaxConcurrency int `json:"max_concurrency"`
	} `json:"session_start_limit"`
}

// GatewayOp represents a packets operation
type GatewayOp uint8

// Operation Codes for gateway messagess
const (
	GatewayOpDispatch GatewayOp = iota
	GatewayOpHeartbeat
	GatewayOpIdentify
	GatewayOpStatusUpdate
	GatewayOpVoiceStateUpdate
	_
	GatewayOpResume
	GatewayOpReconnect
	GatewayOpRequestGuildMembers
	GatewayOpInvalidSession
	GatewayOpHello
	GatewayOpHeartbeatACK
)

// The different gateway intents
const (
	IntentGuilds uint = 1 << iota
	IntentGuildMembers
	IntentGuildBans
	IntentGuildEmojis
	IntentGuildIntegrations
	IntentGuildWebhooks
	IntentGuildInvites
	IntentGuildVoiceStates
	IntentGuildPresences
	IntentGuildMessages
	IntentGuildMessageReactions
	IntentGuildMessageTyping
	IntentDirectMessages
	IntentDirectMessageReactions
	IntentDirectMessageTyping
)

// The gateway's close codes
const (
	CloseUnknownError = 4000 + iota
	CloseUnknownOpCode
	CloseDecodeError
	CloseNotAuthenticated
	CloseAuthenticationFailed
	CloseAlreadyAuthenticated
	_
	CloseInvalidSeq
	CloseRateLimited
	CloseSessionTimeout
	CloseInvalidShard
	CloseShardingRequired
)

// ReceivedPayload is the base of a JSON packet received from discord
type ReceivedPayload struct {
	Op       int             `json:"op"`
	Data     json.RawMessage `json:"d,omitempty"`
	Sequence uint64          `json:"s,omitempty"`
	Type     string          `json:"t,omitempty"`
}

// SentPayload is the base of a JSON packet we sent to discord
type SentPayload struct {
	Op   int         `json:"op"`
	Data interface{} `json:"d"`
}

// Identify represents an identify packet
type Identify struct {
	Token              string              `json:"token"`
	Properties         *IdentifyProperties `json:"properties"`
	Compress           bool                `json:"compress,omitempty"`
	LargeThreshold     int                 `json:"large_threshold,omitempty"`
	Shard              [2]int              `json:"shard,omitempty"`
	Presence           *Activity           `json:"presence,omitempty"`
	GuildSubscriptions bool                `json:"guild_subscriptions,omitempty"`
	Intents            int                 `json:"intents,omitempty"`
}

// IdentifyProperties is the properties sent in the identify packet
type IdentifyProperties struct {
	OS      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

// Resume represents a resume packet
type Resume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       int64  `json:"seq"`
}

// RequestGuildMembers represents a request guild members packet
type RequestGuildMembers struct {
	GuildID snowflake.ID `json:"guild_id"`
	Query   string       `json:"query"`
	Limit   int          `json:"limit"`
}

// UpdateVoiceState represents an update voice state packet
type UpdateVoiceState struct {
	GuildID   snowflake.ID `json:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id"`
	SelfMute  bool         `json:"self_mute"`
	SelfDeaf  bool         `json:"self_deaf"`
}

// Available statuses
const (
	StatusOnline    = "online"
	StatusDND       = "dnd"
	StatusIdle      = "idle"
	StatusInvisible = "invisible"
	StatusOffline   = "offline"
)

// UpdateStatus represents an update status packet
type UpdateStatus struct {
	Since  int       `json:"since,omitempty"`
	Game   *Activity `json:"game,omitempty"`
	Status string    `json:"status"`
	AFK    bool      `json:"afk"`
}

// Hello represents a hello packet
type Hello struct {
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

// Ready represents a ready packet
type Ready struct {
	Version   int      `json:"v"`
	User      *User    `json:"user"`
	Guilds    []*Guild `json:"guilds"`
	SessionID string   `json:"session_id"`
}

// Resumed represents a resumed packet
type Resumed struct {
}
