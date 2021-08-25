package discord

import (
	"github.com/WelcomerTeam/RealRock/snowflake"
)

// webhook.go represents all structures to create a webhook and interact with it.

// WebhookType is the type of webhook.
type WebhookType uint8

// Webhook type.
const (
	WebhookTypeIncoming WebhookType = iota + 1
	WebhookTypeChannelFollower
)

// Webhook represents a webhook on Discord.
type Webhook struct {
	ID   snowflake.ID `json:"id"`
	Type WebhookType  `json:"type"`

	GuildID       *snowflake.ID `json:"guild_id,omitempty"`
	ChannelID     *snowflake.ID `json:"channel_id,omitempty"`
	User          *User         `json:"user,omitempty"`
	Name          string        `json:"name"`
	Avatar        string        `json:"avatar"`
	Token         string        `json:"token"`
	ApplicationID *snowflake.ID `json:"application_id,omitempty"`
}

// WebhookMessage represents a message on Discord for webhooks.
type WebhookMessage struct {
	Content         *string                   `json:"content,omitempty"`
	Username        *string                   `json:"username,omitempty"`
	AvatarURL       *string                   `json:"avatar_url,omitempty"`
	TTS             *bool                     `json:"tts,omitempty"`
	Embeds          []*Embed                  `json:"embeds,omitempty"`
	AllowedMentions []*MessageAllowedMentions `json:"allowed_mentions,omitempty"`
	Components      []*InteractionComponent   `json:"components,omitempty"`
	// PayloadJSON     *json.RawMessage          `json:"payload_json,omitempty"`
}
