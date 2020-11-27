package structs

import "github.com/TheRockettek/snowflake"

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

// WebhookUpdate represents a webhook update packet.
type WebhookUpdate struct {
	GuildID   snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
}
