package discord

// message.go contains the structure that represents a discord message.

// MessageType represents the type of message that has been sent.
type MessageType uint8

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
	_
	MessageTypeGuildDiscoveryDisqualified
	MessageTypeGuildDiscoveryRequalified
	MessageTypeGuildDiscoveryGracePeriodInitialWarning
	MessageTypeGuildDiscoveryGracePeriodFinalWarning
	MessageTypeThreadCreated
	MessageTypeReply
	MessageTypeApplicationCommand
	MessageTypeThreadStarterMessage
	MessageTypeGuildInviteReminder
)

// MessageFlags represents the extra information on a message.
type MessageFlags uint8

const (
	MessageFlagCrossposted MessageFlags = 1 << iota
	MessageFlagIsCrosspost
	MessageFlagSuppressEmbeds
	MessageFlagSourceMessageDeleted
	MessageFlagUrgent
	MessageFlagHasThread
	MessageFlaEphemeral
	MessageFlagLoading
)

// MessageAllowedMentionsType represents all the allowed mention types.
type MessageAllowedMentionsType string

const (
	MessageAllowedMentionsTypeRoles    MessageAllowedMentionsType = "roles"
	MessageAllowedMentionsTypeUsers    MessageAllowedMentionsType = "users"
	MessageAllowedMentionsTypeEveryone MessageAllowedMentionsType = "everyone"
)

// MessageActivityType represents the type of message activity.
type MessageActivityType uint8

const (
	MessageActivityTypeJoin MessageActivityType = 1 + iota
	MessageActivityTypeSpectate
	MessageActivityTypeListen
	MessageActivityTypeJoinRequest
)

// Message represents a message on Discord.
type Message struct {
	ID        Snowflake  `json:"id"`
	ChannelID Snowflake  `json:"channel_id"`
	GuildID   *Snowflake `json:"guild_id,omitempty"`
	Author    *User      `json:"author"`
	Member    *Member    `json:"member,omitempty"`

	Content         string `json:"content"`
	Timestamp       string `json:"timestamp"`
	EditedTimestamp string `json:"edited_timestamp"`
	TTS             bool   `json:"tts"`

	MentionEveryone bool                     `json:"mention_everyone"`
	Mentions        []*User                  `json:"mentions"`
	MentionRoles    []Snowflake              `json:"mention_roles"`
	MentionChannels []*MessageChannelMention `json:"mention_channels,omitempty"`

	Attachments       []*MessageAttachment `json:"attachments"`
	Embeds            []*Embed             `json:"embeds"`
	Reactions         []*MessageReaction   `json:"reactions"`
	Nonce             *Snowflake           `json:"nonce,omitempty"`
	Pinned            bool                 `json:"pinned"`
	WebhookID         *Snowflake           `json:"webhook_id,omitempty"`
	Type              MessageType          `json:"type"`
	Activity          *MessageActivity     `json:"activity"`
	Application       *Application         `json:"application"`
	MessageReference  []*MessageReference  `json:"message_referenced,omitempty"`
	Flags             *MessageFlags        `json:"flags,omitempty"`
	Components        []*InteractionComponent
	Stickers          []*Sticker `json:"stickers,omitempty"`
	ReferencedMessage *Message   `json:"referenced_message,omitempty"`
}

// MessageChannelMention represents a mentioned channel.
type MessageChannelMention struct {
	ID      Snowflake   `json:"id"`
	GuildID Snowflake   `json:"guild_id"`
	Type    ChannelType `json:"type"`
	Name    string      `json:"name"`
}

// MessageReference represents crossposted messages or replys.
type MessageReference struct {
	ID              *Snowflake `json:"message_id,omitempty"`
	ChannelID       *Snowflake `json:"channel_id,omitempty"`
	GuildID         *Snowflake `json:"guild_id,omitempty"`
	FailIfNotExists *bool      `json:"fail_if_not_exists,omitempty"`
}

// MessageReaction represents a reaction to a message on Discord.
type MessageReaction struct {
	Count int    `json:"count"`
	Me    bool   `json:"me"`
	Emoji *Emoji `json:"emoji"`
}

// MessageAllowedMentions is the structure of the allowed mentions entry.
type MessageAllowedMentions struct {
	Parse       []MessageAllowedMentionsType `json:"parse"`
	Roles       []Snowflake                  `json:"roles"`
	Users       []Snowflake                  `json:"users"`
	RepliedUser bool                         `json:"replied_user"`
}

// MessageAttachment represents a message attachment on discord.
type MessageAttachment struct {
	ID       Snowflake `json:"id"`
	Filename string    `json:"filename"`
	Size     int       `json:"size"`
	URL      string    `json:"url"`
	ProxyURL string    `json:"proxy_url"`
	Height   int       `json:"height"`
	Width    int       `json:"width"`
}

// MessageActivity represents a message activity on Discord.
type MessageActivity struct {
	Type    MessageActivityType `json:"type"`
	PartyID *string             `json:"party_id,omitempty"`
}
