package structs

import "github.com/TheRockettek/snowflake"

// Webhook represents a webhook on Discord
type Webhook struct {
	ID        snowflake.ID `json:"id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	User      User         `json:"user,omitempty"`
	Name      string       `json:"name"`
	Avatar    string       `json:"avatar"`
	Token     string       `json:"token"`
}

// WebhookUpdate represents a webhook update packet
type WebhookUpdate struct {
	GuildID   snowflake.ID `json:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id"`
}
