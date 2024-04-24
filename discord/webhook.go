package discord

import "encoding/json"

// webhook.go represents all structures to create a webhook and interact with it.

// WebhookType is the type of webhook.
type WebhookType uint8

// Webhook type.
const (
	WebhookTypeIncoming WebhookType = iota + 1
	WebhookTypeChannelFollower
)

// Webhook represents a webhook on discord.
type Webhook struct {
	GuildID       *Snowflake  `json:"guild_id,omitempty"`
	ChannelID     *Snowflake  `json:"channel_id,omitempty"`
	User          *User       `json:"user,omitempty"`
	ApplicationID *Snowflake  `json:"application_id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Avatar        string      `json:"avatar,omitempty"`
	Token         string      `json:"token"`
	ID            Snowflake   `json:"id"`
	Type          WebhookType `json:"type"`
}

// WebhookMessage represents the structure for sending a webhook message.
type WebhookMessageParams struct {
	PayloadJSON     *json.RawMessage          `json:"payload_json,omitempty"`
	Content         string                    `json:"content,omitempty"`
	Username        string                    `json:"username,omitempty"`
	AvatarURL       string                    `json:"avatar_url,omitempty"`
	Embeds          []*Embed                  `json:"embeds,omitempty"`
	AllowedMentions []*MessageAllowedMentions `json:"allowed_mentions,omitempty"`
	Components      []*InteractionComponent   `json:"components,omitempty"`
	Files           []*File                   `json:"-"`
	Attachments     []*MessageAttachment      `json:"attachments,omitempty"`
	TTS             bool                      `json:"tts,omitempty"`
}

// WebhookParam represents the data sent to discord to create a webhook.
type WebhookParam struct {
	Name   *string `json:"name,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
}
