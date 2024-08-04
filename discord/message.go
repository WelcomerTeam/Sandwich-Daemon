package discord

// message.go contains the structure that represents a discord message.

// MessageType represents the type of message that has been sent.
type MessageType uint16

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
type MessageFlags uint16

const (
	MessageFlagCrossposted MessageFlags = 1 << iota
	MessageFlagIsCrosspost
	MessageFlagSuppressEmbeds
	MessageFlagSourceMessageDeleted
	MessageFlagUrgent
	MessageFlagHasThread
	MessageFlagEphemeral
	MessageFlagLoading
	MessageFlagFailedToMentionSomeRolesInThread
)

// MessageAllowedMentionsType represents all the allowed mention types.
type MessageAllowedMentionsType string

const (
	MessageAllowedMentionsTypeRoles    MessageAllowedMentionsType = "roles"
	MessageAllowedMentionsTypeUsers    MessageAllowedMentionsType = "users"
	MessageAllowedMentionsTypeEveryone MessageAllowedMentionsType = "everyone"
)

// MessageActivityType represents the type of message activity.
type MessageActivityType uint16

const (
	MessageActivityTypeJoin MessageActivityType = 1 + iota
	MessageActivityTypeSpectate
	MessageActivityTypeListen
	MessageActivityTypeJoinRequest
)

// Message represents a message on discord.
type Message struct {
	Timestamp         Timestamp               `json:"timestamp"`
	EditedTimestamp   Timestamp               `json:"edited_timestamp"`
	Author            User                    `json:"author"`
	WebhookID         *Snowflake              `json:"webhook_id,omitempty"`
	Member            *GuildMember            `json:"member,omitempty"`
	GuildID           *Snowflake              `json:"guild_id,omitempty"`
	Thread            *Channel                `json:"thread,omitempty"`
	Interaction       *MessageInteraction     `json:"interaction,omitempty"`
	ReferencedMessage *Message                `json:"referenced_message,omitempty"`
	Flags             *MessageFlags           `json:"flags,omitempty"`
	Application       *Application            `json:"application,omitempty"`
	Activity          *MessageActivity        `json:"activity,omitempty"`
	Content           string                  `json:"content"`
	Embeds            []Embed                 `json:"embeds"`
	MentionRoles      []Snowflake             `json:"mention_roles"`
	Reactions         []MessageReaction       `json:"reactions"`
	StickerItems      []MessageSticker        `json:"sticker_items,omitempty"`
	Attachments       []MessageAttachment     `json:"attachments"`
	Components        []InteractionComponent  `json:"components,omitempty"`
	MentionChannels   []MessageChannelMention `json:"mention_channels,omitempty"`
	Mentions          []User                  `json:"mentions"`
	MessageReference  []MessageReference      `json:"message_referenced,omitempty"`
	ID                Snowflake               `json:"id"`
	ChannelID         Snowflake               `json:"channel_id"`
	MentionEveryone   bool                    `json:"mention_everyone"`
	TTS               bool                    `json:"tts"`
	Type              MessageType             `json:"type"`
	Pinned            bool                    `json:"pinned"`
}

// MessageInteraction represents an executed interaction.
type MessageInteraction struct {
	User User            `json:"user"`
	Type InteractionType `json:"type"`
	Name string          `json:"name"`
	ID   Snowflake       `json:"id"`
}

// MessageChannelMention represents a mentioned channel.
type MessageChannelMention struct {
	Name    string      `json:"name"`
	ID      Snowflake   `json:"id"`
	GuildID Snowflake   `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// MessageReference represents crossposted messages or replys.
type MessageReference struct {
	ID              *Snowflake `json:"message_id,omitempty"`
	ChannelID       *Snowflake `json:"channel_id,omitempty"`
	GuildID         *Snowflake `json:"guild_id,omitempty"`
	FailIfNotExists bool       `json:"fail_if_not_exists"`
}

// MessageReaction represents a reaction to a message on discord.
type MessageReaction struct {
	Emoji        Emoji                       `json:"emoji"`
	BurstColors  []string                    `json:"burst_colors"`
	CountDetails MessageReactionCountDetails `json:"count_details"`
	Count        int32                       `json:"count"`
	BurstCount   int32                       `json:"burst_count"`
	MeBurst      bool                        `json:"me_burst"`
	BurstMe      bool                        `json:"burst_me"`
	Me           bool                        `json:"me"`
}

// MessageReactionCountDetails represents the count details of a message reaction.
type MessageReactionCountDetails struct {
	Burst  int32 `json:"burst"`
	Normal int32 `json:"normal"`
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
	Filename  string    `json:"filename"`
	URL       string    `json:"url"`
	ProxyURL  string    `json:"proxy_url"`
	ID        Snowflake `json:"id"`
	Size      int32     `json:"size"`
	Height    int32     `json:"height"`
	Width     int32     `json:"width"`
	Ephemeral bool      `json:"ephemeral"`
}

// MessageActivity represents a message activity on discord.
type MessageActivity struct {
	PartyID string              `json:"party_id,omitempty"`
	Type    MessageActivityType `json:"type"`
}
