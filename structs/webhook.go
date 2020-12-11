package structs

import (
	"encoding/json"

	"github.com/TheRockettek/snowflake"
)

// Webhook represents a webhook on Discord.
type Webhook struct {
	ID        snowflake.ID `json:"id" msgpack:"id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	ChannelID snowflake.ID `json:"channel_id,omitempty" msgpack:"channel_id,omitempty"`
	User      User         `json:"user,omitempty" msgpack:"user,omitempty"`
	Name      string       `json:"name" msgpack:"name"`
	Avatar    string       `json:"avatar" msgpack:"avatar"`
	Token     string       `json:"token" msgpack:"token"`
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

// WebhookUpdate represents a webhook update packet.
type WebhookUpdate struct {
	GuildID   snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
}
