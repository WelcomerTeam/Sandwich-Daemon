package discord

import (
	jsoniter "github.com/json-iterator/go"
)

// gateway.go contains all structures for interacting with discord's gateway and contains
// all events and structures we send to

// GatewayOp represents the operation codes of a gateway message.
type GatewayOp uint8

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

// IntentFlag represents a bitflag for intents.
type GatewayIntent uint32

const (
	IntentGuilds GatewayIntent = 1 << iota
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
	IntentMessageContent
)

// Gateway close codes.
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
	CloseInvalidAPIVersion
	CloseInvalidIntents
	CloseDisallowedIntents
)

// GatewayPayload represents the base payload received from discord gateway.
type GatewayPayload struct {
	Type     string              `json:"t"`
	Data     jsoniter.RawMessage `json:"d"`
	Sequence int32               `json:"s"`
	Op       GatewayOp           `json:"op"`
}

// SentPayload represents the base payload we send to discords gateway.
type SentPayload struct {
	Data interface{} `json:"d"`
	Op   GatewayOp   `json:"op"`
}

// Gateway Commands

// Identify represents the initial handshake with the gateway.
type Identify struct {
	Properties     *IdentifyProperties `json:"properties"`
	Presence       *UpdateStatus       `json:"presence,omitempty"`
	Token          string              `json:"token"`
	Shard          [2]int32            `json:"shard,omitempty"`
	LargeThreshold int32               `json:"large_threshold"`
	Intents        int32               `json:"intents"`
	Compress       bool                `json:"compress"`
}

// IdentifyProperties are the extra properties sent in the identify packet.
type IdentifyProperties struct {
	OS      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

// Resume resumes a dropped gateway connection.
type Resume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int32  `json:"seq"`
}

// Heartbeat represents the heartbeat packet.
type Heartbeat int

// Request guild members requests members for a guild.
type RequestGuildMembers struct {
	Query     string      `json:"query"`
	Nonce     string      `json:"nonce"`
	UserIDs   []Snowflake `json:"user_ids"`
	GuildID   Snowflake   `json:"guild_id"`
	Limit     int32       `json:"limit"`
	Presences bool        `json:"presences"`
}

// Update Presence updates a client's presence.
type UpdateStatus struct {
	Status     string      `json:"status"`
	Activities []*Activity `json:"activities,omitempty"`
	Since      int32       `json:"since,omitempty"`
	AFK        bool        `json:"afk"`
}
