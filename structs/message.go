package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// Message represents a message on Discord
type Message struct {
	ID              snowflake.ID       `json:"id"`
	ChannelID       snowflake.ID       `json:"channel_id"`
	GuildID         snowflake.ID       `json:"guild_id,omitempty"`
	Author          *User              `json:"author"`
	Member          *GuildMember       `json:"member,omitempty"`
	Content         string             `json:"content"`
	Timestamp       string             `json:"timestamp"`
	EditedTimestamp string             `json:"edited_timestamp"`
	TTS             bool               `json:"tts"`
	MentionEveryone bool               `json:"mention_everyone"`
	Mentions        []*User            `json:"mentions"`
	MentionRoles    []snowflake.ID     `json:"mention_roles"`
	Attachments     []Attachment       `json:"attachments"`
	Embeds          []Embed            `json:"embeds"`
	Reactions       []Reaction         `json:"reactions"`
	Nonce           snowflake.ID       `json:"nonce,omitempty"`
	Pinned          bool               `json:"pinned"`
	WebhookID       snowflake.ID       `json:"webhook_id,omitempty"`
	Type            int                `json:"type"`
	Activity        MessageActivity    `json:"activity"`
	Application     MessageApplication `json:"application"`
}

// message types
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

// MessageActivity represents a message activity on Discord
type MessageActivity struct {
	Type    int    `json:"type"`
	PartyID string `json:"party_id,omitempty"`
}

// MessageApplication represents a message application on Discord
type MessageApplication struct {
	ID          snowflake.ID `json:"id"`
	CoverImage  string       `json:"cover_image"`
	Description string       `json:"description"`
	Icon        string       `json:"icon"`
	Name        string       `json:"name"`
}

// message activity types
const (
	MessageActivityTypeJoin = iota
	MessageActivityTypeSpectate
	MessageActivityTypeListen
	MessageActivityTypeJoinRequest
)

// Reaction represents a reaction to a message on Discord
type Reaction struct {
	Count int    `json:"count"`
	Me    bool   `json:"me"`
	Emoji *Emoji `json:"emoji"` // TODO: type
}

// Attachment represents a message attachment on discord
type Attachment struct {
	ID       snowflake.ID `json:"id"`
	Filename string       `json:"filename"`
	Size     int          `json:"size"`
	URL      string       `json:"url"`
	ProxyURL string       `json:"proxy_url"`
	Height   int          `json:"height"`
	Width    int          `json:"width"`
}

// Embed represents a message embed on Discord
type Embed struct {
	Title       string         `json:"title,omitempty"`
	Type        string         `json:"type,omitempty"`
	Description string         `json:"description,omitempty"`
	URL         string         `json:"url,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Color       int            `json:"color,omitempty"`
	Footer      EmbedFooter    `json:"footer,omitempty"`
	Image       EmbedImage     `json:"image,omitempty"`
	Thumbnail   EmbedThumbnail `json:"thumbnail,omitempty"`
	Video       EmbedVideo     `json:"video,omitempty"`
	Provider    EmbedProvider  `json:"provider,omitempty"`
	Author      EmbedAuthor    `json:"author,omitempty"`
	Fields      []EmbedField   `json:"fields,omitempty"`
}

// EmbedFooter represents the footer of an embed
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// EmbedImage represents an image in an embed
type EmbedImage struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

// EmbedThumbnail represents the thumbnail of an embed
type EmbedThumbnail struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omiempty"`
	Width    int    `json:"width,omitempty"`
}

// EmbedVideo represents the video of an embed
type EmbedVideo struct {
	URL    string `json:"url,omitempty"`
	Height int    `json:"height,omitempty"`
	Width  int    `json:"width,omitempty"`
}

// EmbedProvider represents the provider of an embed
type EmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// EmbedAuthor represents the author of an embed
type EmbedAuthor struct {
	Name         string `json:"name,omitempty"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// EmbedField represents a field in an embed
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// MessageCreate represents a message create packet
type MessageCreate struct {
	*Message
}

// MessageUpdate represents a message update packet
type MessageUpdate struct {
	*Message
}

// MessageDelete represents a message delete packet
type MessageDelete struct {
	ID        snowflake.ID `json:"id"`
	ChannelID snowflake.ID `json:"channel_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
}

// MessageDeleteBulk represents a message delete bulk packet
type MessageDeleteBulk struct {
	IDs       []snowflake.ID `json:"ids"`
	ChannelID snowflake.ID   `json:"channel_id"`
	GuildID   snowflake.ID   `json:"guild_id,omitempty"`
}

// MessageReactionAdd represents a message reaction add packet
type MessageReactionAdd struct {
	UserID    snowflake.ID `json:"user_id"`
	ChannelID snowflake.ID `json:"channel_id"`
	MessageID snowflake.ID `json:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	Emoji     interface{}  `json:"emoji"` // TODO: type
}

// MessageReactionRemove represents a message reaction remove packet
type MessageReactionRemove struct {
	UserID    snowflake.ID `json:"user_id"`
	ChannelID snowflake.ID `json:"channel_id"`
	MessageID snowflake.ID `json:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	Emoji     *Emoji       `json:"emoji"` // TODO: type
}

// MessageReactionRemoveAll represents a message reaction remove all packet
type MessageReactionRemoveAll struct {
	ChannelID snowflake.ID `json:"channel_id"`
	MessageID snowflake.ID `json:"message_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
}
