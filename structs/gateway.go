package structs

import (
	"encoding/json"
	"time"
)

// GatewayOp represents a packets operation.
type GatewayOp uint8

// Operation Codes for gateway messages.
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

// The different gateway intents.
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

// The gateway's close codes.
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

// Available statuses.
const (
	StatusOnline    = "online"
	StatusDND       = "dnd"
	StatusIdle      = "idle"
	StatusInvisible = "invisible"
	StatusOffline   = "offline"
)

// Gateway represents a GET /gateway response.
type Gateway struct {
	URL string `json:"url" msgpack:"url"`
}

// GatewayBot represents a GET /gateway/bot response.
type GatewayBot struct {
	URL               string `json:"url" msgpack:"url"`
	Shards            int    `json:"shards" msgpack:"shards"`
	SessionStartLimit struct {
		Total          int `json:"total" msgpack:"total"`
		Remaining      int `json:"remaining" msgpack:"remaining"`
		ResetAfter     int `json:"reset_after" msgpack:"reset_after"`
		MaxConcurrency int `json:"max_concurrency" msgpack:"max_concurrency"`
	} `json:"session_start_limit" msgpack:"session_start_limit"`
}

// ReceivedPayload is the base of a JSON packet received from discord.
type ReceivedPayload struct {
	Op       GatewayOp       `json:"op" msgpack:"op"`
	Data     json.RawMessage `json:"d,omitempty" msgpack:"d,omitempty"`
	Sequence int64           `json:"s,omitempty" msgpack:"s,omitempty"`
	Type     string          `json:"t,omitempty" msgpack:"t,omitempty"`

	// Used for trace tracking
	TraceTime time.Time      `json:"-" msgpack:"-"`
	Trace     map[string]int `json:"-" msgpack:"-"`
}

// ReceivedPayload adds a trace entry and overwrites the current trace time.
func (rp *ReceivedPayload) AddTrace(name string, now time.Time) {
	rp.Trace[name] = int(float64(now.Sub(rp.TraceTime).Milliseconds()))
	rp.TraceTime = now
}

// SentPayload is the base of a JSON packet we sent to discord.
type SentPayload struct {
	Op   int         `json:"op" msgpack:"op"`
	Data interface{} `json:"d" msgpack:"d"`
}
