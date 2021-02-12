package structs

import (
	"encoding/json"

	"github.com/TheRockettek/snowflake"
)

// WebhookType is the type of webhook.
type WebhookType int

// Webhook type.
const (
	WebhookTypeIncoming WebhookType = iota + 1
	WebhookTypeChannelFollower
)

// Webhook represents a webhook on Discord.
type Webhook struct {
	ID   snowflake.ID `json:"id" msgpack:"id"`
	Type WebhookType  `json:"type" msgpack:"type"`

	GuildID       snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	ChannelID     snowflake.ID `json:"channel_id,omitempty" msgpack:"channel_id,omitempty"`
	User          User         `json:"user,omitempty" msgpack:"user,omitempty"`
	Name          string       `json:"name" msgpack:"name"`
	Avatar        string       `json:"avatar" msgpack:"avatar"`
	Token         string       `json:"token" msgpack:"token"`
	ApplicationID snowflake.ID `json:"application_id,omitempty" msgpack:"application_id,omitempty"`
}

// WebhookMessage represents a message on Discord for webhooks.
type WebhookMessage struct {
	Content         string                   `json:"content,omitempty" msgpack:"content,omitempty"`
	Username        string                   `json:"username,omitempty" msgpack:"username,omitempty"`
	AvatarURL       string                   `json:"avatar_url,omitempty" msgpack:"avatar_url,omitempty"`
	TTS             bool                     `json:"tts,omitempty" msgpack:"tts,omitempty"`
	Embeds          []Embed                  `json:"embeds,omitempty" msgpack:"embeds,omitempty"`
	PayloadJSON     json.RawMessage          `json:"payload_json,omitempty" msgpack:"payload_json,omitempty"`
	AllowedMentions []MessageAllowedMentions `json:"allowed_mentions,omitempty" msgpack:"allowed_mentions,omitempty"`
	// Todo: file support
}
