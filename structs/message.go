package structs

import (
	"encoding/json"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
)

// Message represents a message on Discord.
type Message struct {
	ID              snowflake.ID       `json:"id" msgpack:"id"`
	ChannelID       snowflake.ID       `json:"channel_id" msgpack:"channel_id"`
	GuildID         snowflake.ID       `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Nonce           snowflake.ID       `json:"nonce,omitempty" msgpack:"nonce,omitempty"`
	WebhookID       snowflake.ID       `json:"webhook_id,omitempty" msgpack:"webhook_id,omitempty"`
	Content         string             `json:"content" msgpack:"content"`
	Timestamp       string             `json:"timestamp" msgpack:"timestamp"`
	EditedTimestamp string             `json:"edited_timestamp" msgpack:"edited_timestamp"`
	Type            int                `json:"type" msgpack:"type"`
	Author          *User              `json:"author" msgpack:"author"`
	Member          *GuildMember       `json:"member,omitempty" msgpack:"member,omitempty"`
	Mentions        []*User            `json:"mentions" msgpack:"mentions"`
	MentionRoles    []snowflake.ID     `json:"mention_roles" msgpack:"mention_roles"`
	Attachments     []Attachment       `json:"attachments" msgpack:"attachments"`
	Embeds          []Embed            `json:"embeds" msgpack:"embeds"`
	Reactions       []Reaction         `json:"reactions" msgpack:"reactions"`
	Activity        MessageActivity    `json:"activity" msgpack:"activity"`
	Application     MessageApplication `json:"application" msgpack:"application"`
	TTS             bool               `json:"tts" msgpack:"tts"`
	MentionEveryone bool               `json:"mention_everyone" msgpack:"mention_everyone"`
	Pinned          bool               `json:"pinned" msgpack:"pinned"`
	// Todo: allowed mentions support
}

// WebhookMessage represents a message on Discord for webhooks.
type WebhookMessage struct {
	Content     string          `json:"content,omitempty" msgpack:"content,omitempty"`
	Username    string          `json:"username,omitempty" msgpack:"username,omitempty"`
	AvatarURL   string          `json:"avatar_url,omitempty" msgpack:"avatar_url,omitempty"`
	TTS         bool            `json:"tts,omitempty" msgpack:"tts,omitempty"`
	Embeds      []Embed         `json:"embeds,omitempty" msgpack:"embeds,omitempty"`
	PayloadJSON json.RawMessage `json:"payload_json,omitempty" msgpack:"payload_json,omitempty"`
	// Todo: allowed mentions and file support
}

// message types.
const (
	MessageTypeDefault = iota
	MessageTypeRecipientAdd
	MessageTypeRecipientRemove
	MessageTypeCall
	MessageTypeChannelNameChange
	MessageTypeChannelIconChange
	MessageTypeChannelPinnedMessage
	MessageTypeGuildMemberJoin
)

// MessageActivity represents a message activity on Discord.
type MessageActivity struct {
	Type    int    `json:"type" msgpack:"type"`
	PartyID string `json:"party_id,omitempty" msgpack:"party_id,omitempty"`
}

// MessageApplication represents a message application on Discord.
type MessageApplication struct {
	ID          snowflake.ID `json:"id" msgpack:"id"`
	CoverImage  string       `json:"cover_image" msgpack:"cover_image"`
	Description string       `json:"description" msgpack:"description"`
	Icon        string       `json:"icon" msgpack:"icon"`
	Name        string       `json:"name" msgpack:"name"`
}

// message activity types.
const (
	MessageActivityTypeJoin = iota
	MessageActivityTypeSpectate
	MessageActivityTypeListen
	MessageActivityTypeJoinRequest
)

// Reaction represents a reaction to a message on Discord.
type Reaction struct {
	Count int    `json:"count" msgpack:"count"`
	Me    bool   `json:"me" msgpack:"me"`
	Emoji *Emoji `json:"emoji" msgpack:"emoji"` // TODO: type
}

// Attachment represents a message attachment on discord.
type Attachment struct {
	ID       snowflake.ID `json:"id" msgpack:"id"`
	Filename string       `json:"filename" msgpack:"filename"`
	Size     int          `json:"size" msgpack:"size"`
	URL      string       `json:"url" msgpack:"url"`
	ProxyURL string       `json:"proxy_url" msgpack:"proxy_url"`
	Height   int          `json:"height" msgpack:"height"`
	Width    int          `json:"width" msgpack:"width"`
}

// Embed represents a message embed on Discord.
type Embed struct {
	Title       string          `json:"title,omitempty" msgpack:"title,omitempty"`
	Type        string          `json:"type,omitempty" msgpack:"type,omitempty"`
	Description string          `json:"description,omitempty" msgpack:"description,omitempty"`
	URL         string          `json:"url,omitempty" msgpack:"url,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty" msgpack:"timestamp,omitempty"`
	Color       int             `json:"color,omitempty" msgpack:"color,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty" msgpack:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty" msgpack:"image,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty" msgpack:"thumbnail,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty" msgpack:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty" msgpack:"provider,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty" msgpack:"author,omitempty"`
	Fields      []*EmbedField   `json:"fields,omitempty" msgpack:"fields,omitempty"`
}

// EmbedFooter represents the footer of an embed.
type EmbedFooter struct {
	Text         string `json:"text" msgpack:"text"`
	IconURL      string `json:"icon_url,omitempty" msgpack:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty" msgpack:"proxy_icon_url,omitempty"`
}

// EmbedImage represents an image in an embed.
type EmbedImage struct {
	URL      string `json:"url,omitempty" msgpack:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty" msgpack:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty" msgpack:"height,omitempty"`
	Width    int    `json:"width,omitempty" msgpack:"width,omitempty"`
}

// EmbedThumbnail represents the thumbnail of an embed.
type EmbedThumbnail struct {
	URL      string `json:"url,omitempty" msgpack:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty" msgpack:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty" msgpack:"height,omitempty"`
	Width    int    `json:"width,omitempty" msgpack:"width,omitempty"`
}

// EmbedVideo represents the video of an embed.
type EmbedVideo struct {
	URL    string `json:"url,omitempty" msgpack:"url,omitempty"`
	Height int    `json:"height,omitempty" msgpack:"height,omitempty"`
	Width  int    `json:"width,omitempty" msgpack:"width,omitempty"`
}

// EmbedProvider represents the provider of an embed.
type EmbedProvider struct {
	Name string `json:"name,omitempty" msgpack:"name,omitempty"`
	URL  string `json:"url,omitempty" msgpack:"url,omitempty"`
}

// EmbedAuthor represents the author of an embed.
type EmbedAuthor struct {
	Name         string `json:"name,omitempty" msgpack:"name,omitempty"`
	URL          string `json:"url,omitempty" msgpack:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty" msgpack:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty" msgpack:"proxy_icon_url,omitempty"`
}

// EmbedField represents a field in an embed.
type EmbedField struct {
	Name   string `json:"name" msgpack:"name"`
	Value  string `json:"value" msgpack:"value"`
	Inline bool   `json:"inline,omitempty" msgpack:"inline,omitempty"`
}

// MessageCreate represents a message create packet.
type MessageCreate struct {
	*Message
}

// MessageUpdate represents a message update packet.
type MessageUpdate struct {
	*Message
}

// MessageDelete represents a message delete packet.
type MessageDelete struct {
	ID        snowflake.ID `json:"id" msgpack:"id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}

// MessageDeleteBulk represents a message delete bulk packet.
type MessageDeleteBulk struct {
	IDs       []snowflake.ID `json:"ids" msgpack:"ids"`
	ChannelID snowflake.ID   `json:"channel_id" msgpack:"channel_id"`
	GuildID   snowflake.ID   `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}

// MessageReactionAdd represents a message reaction add packet.
type MessageReactionAdd struct {
	UserID    snowflake.ID `json:"user_id" msgpack:"user_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	MessageID snowflake.ID `json:"message_id" msgpack:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Emoji     interface{}  `json:"emoji" msgpack:"emoji"` // TODO: type
}

// MessageReactionRemove represents a message reaction remove packet.
type MessageReactionRemove struct {
	UserID    snowflake.ID `json:"user_id" msgpack:"user_id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	MessageID snowflake.ID `json:"message_id" msgpack:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Emoji     *Emoji       `json:"emoji" msgpack:"emoji"` // TODO: type
}

// MessageReactionRemoveAll represents a message reaction remove all packet.
type MessageReactionRemoveAll struct {
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	MessageID snowflake.ID `json:"message_id" msgpack:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}
