package structs

import "github.com/TheRockettek/snowflake"

// VoiceOp is the voice packet operation codes
type VoiceOp uint8

// Voice op codes
const (
	VoiceOpIdentify VoiceOp = iota
	VoiceOpSelectProtocol
	VoiceOpReady
	VoiceOpHeartbeat
	VoiceOpSessionDescription
	VoiceOpSpeaking
	VoiceOpHeartbeatAck
	VoiceOpResume
	VoiceOpHello
	VoiceOpResumed
	_
	_
	_
	VoiceOpClientDisconnect
)

// Voice close codes
const (
	_ = 4000 + iota
	VoiceCloseUnknownOpCode
	_
	VoiceCloseNotAuthenticate
	VoiceCloseAuthenticationFailed
	VoiceCloseAlreadyAuthenticated
	VoiceCloseSessionNoLongerValid
	_
	_
	VoiceCloseSessionTimeout
	_
	VoiceCloseServerNotFound
	VoiceCloseUnknownProtocol
	_
	VoiceCloseDisconnected
	VoiceCloseVoiceServerCrashed
	VoiceCloseUnknownEncryptionMode
)

// VoiceState represents the voice state on Discord
type VoiceState struct {
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	UserID    snowflake.ID `json:"user_id" msgpack:"user_id"`
	Member    GuildMember  `json:"member,omitempty" msgpack:"member,omitempty"`
	SessionID string       `json:"session_id" msgpack:"session_id"`
	Deaf      bool         `json:"deaf" msgpack:"deaf"`
	Mute      bool         `json:"mute" msgpack:"mute"`
	SelfDeaf  bool         `json:"self_deaf" msgpack:"self_deaf"`
	SelfMute  bool         `json:"self_mute" msgpack:"self_mute"`
	Suppress  bool         `json:"suppress" msgpack:"suppress"`
}

// VoiceStateUpdate represents the VOICE_STATE_UPDATE packet
type VoiceStateUpdate VoiceState

// VoiceServerUpdate represents a VOICE_SERVER_UPDATE packet
type VoiceServerUpdate struct {
	Token    string       `json:"token" msgpack:"token"`
	GuildID  snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Endpoint string       `json:"endpoint" msgpack:"endpoint"`
}
