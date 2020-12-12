package structs

import (
	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
)

// Message represents a message on Discord.
type Message struct {
	ID        snowflake.ID `json:"id" msgpack:"id"`
	ChannelID snowflake.ID `json:"channel_id" msgpack:"channel_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
	Author    *User        `json:"author" msgpack:"author"`
	Member    *GuildMember `json:"member,omitempty" msgpack:"member,omitempty"`

	Content         string `json:"content" msgpack:"content"`
	Timestamp       string `json:"timestamp" msgpack:"timestamp"`
	EditedTimestamp string `json:"edited_timestamp" msgpack:"edited_timestamp"`
	TTS             bool   `json:"tts" msgpack:"tts"`

	MentionEveryone bool                    `json:"mention_everyone" msgpack:"mention_everyone"`
	Mentions        []*User                 `json:"mentions" msgpack:"mentions"`
	MentionRoles    []snowflake.ID          `json:"mention_roles" msgpack:"mention_roles"`
	MentionChannels []MessageChannelMention `json:"mention_channels,omitempty" msgpack:"mention_channels,omitempty"`

	Attachments       []Attachment       `json:"attachments" msgpack:"attachments"`
	Embeds            []Embed            `json:"embeds" msgpack:"embeds"`
	Reactions         []MessageReaction  `json:"reactions" msgpack:"reactions"`
	Nonce             snowflake.ID       `json:"nonce,omitempty" msgpack:"nonce,omitempty"`
	Pinned            bool               `json:"pinned" msgpack:"pinned"`
	WebhookID         snowflake.ID       `json:"webhook_id,omitempty" msgpack:"webhook_id,omitempty"`
	Type              MessageType        `json:"type" msgpack:"type"`
	Activity          MessageActivity    `json:"activity" msgpack:"activity"`
	Application       MessageApplication `json:"application" msgpack:"application"`
	MessageReference  []MessageReference `json:"message_referenced,omitempty" msgpack:"message_referenced,omitempty"`
	Flags             MessageFlag        `json:"flags,omitempty" msgpack:"flags,omitempty"`
	Stickers          []Sticker          `json:"stickers,omitempty" msgpack:"stickers,omitempty"`
	ReferencedMessage *Message           `json:"referenced_message,omitempty" msgpack:"referenced_message,omitempty"`
}

type MessageType int64

// message types.
const (
	MessageTypeDefault MessageType = iota
	MessageTypeRecipientAdd
	MessageTypeRecipientRemove
	MessageTypeCall
	MessageTypeChannelNameChange
	MessageTypeChannelIconChange
	MessageTypeChannelPinnedMessage
	MessageTypeGuildMemberJoin
	MessageTypeUserPremiumGuildSubscription
	MessageTypeUserPremiumGuildSubscriptionTier1
	MessageTypeUserPremiumGuildSubscriptionTier2
	MessageTypeUserPremiumGuildSubscriptionTier3
	MessageTypeChannelFollowAdd
	MessageTypeGuildDiscoveryDisqualified
	MessageTypeGuildDiscoveryRequalified
	MessageTypeReply
)

type MessageFlag int64

// message flags.
const (
	MessageFlagCrossposted MessageFlag = 1 << iota
	MessageFlagIsCrosspost
	MessageFlagSuppressEmbeds
	MessageFlagSourceMessageDeleted
	MessageFlagUrgent
)

// MessageChannelMention represents a mentioned channel.
type MessageChannelMention struct {
	ID      snowflake.ID `json:"id" msgpack:"id"`
	GuildID snowflake.ID `json:"guild_id" msgpack:"guild_id"`
	Type    ChannelType  `json:"type" msgpack:"type"`
	Name    string       `json:"name" msgpack:"name"`
}

// MessageReference represents crossposted messages or replys.
type MessageReference struct {
	ID        int64 `json:"message_id,omitempty" msgpack:"message_id,omitempty"`
	ChannelID int64 `json:"channel_id,omitempty" msgpack:"channel_id,omitempty"`
	GuildID   int64 `json:"guild_id,omitempty" msgpack:"guild_id,omitempty"`
}

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

// MessageReaction represents a reaction to a message on Discord.
type MessageReaction struct {
	Count int    `json:"count" msgpack:"count"`
	Me    bool   `json:"me" msgpack:"me"`
	Emoji *Emoji `json:"emoji" msgpack:"emoji"`
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

// MessageAllowedMentions is the structure of the allowed mentions entry.
type MessageAllowedMentions struct {
	// Allowed values are: roles, users, everyone.
	Parse       []string       `json:"parse" msgpack:"parse"`
	Roles       []snowflake.ID `json:"roles" msgpack:"roles"`
	Users       []snowflake.ID `json:"users" msgpack:"users"`
	RepliedUser bool           `json:"replied_user" msgpack:"replied_user"`
}
