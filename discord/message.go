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
	WebhookID         *WebhookID              `json:"webhook_id,omitempty"`
	Member            *GuildMember            `json:"member,omitempty"`
	GuildID           *GuildID                `json:"guild_id,omitempty"`
	Thread            *Channel                `json:"thread,omitempty"`
	Interaction       *MessageInteraction     `json:"interaction,omitempty"`
	ReferencedMessage *Message                `json:"referenced_message,omitempty"`
	Flags             *MessageFlags           `json:"flags,omitempty"`
	Application       *Application            `json:"application,omitempty"`
	Activity          *MessageActivity        `json:"activity,omitempty"`
	Timestamp         Timestamp               `json:"timestamp"`
	EditedTimestamp   Timestamp               `json:"edited_timestamp"`
	Content           string                  `json:"content"`
	Embeds            EmbedList               `json:"embeds"`
	MentionRoles      RoleIDList              `json:"mention_roles"`
	Reactions         []MessageReaction       `json:"reactions"`
	StickerItems      []MessageSticker        `json:"sticker_items,omitempty"`
	Attachments       []MessageAttachment     `json:"attachments"`
	Components        []InteractionComponent  `json:"components,omitempty"`
	MentionChannels   []MessageChannelMention `json:"mention_channels,omitempty"`
	Mentions          UserList                `json:"mentions"`
	MessageReference  []MessageReference      `json:"message_referenced,omitempty"`
	Author            User                    `json:"author"`
	ID                MessageID               `json:"id"`
	ChannelID         ChannelID               `json:"channel_id"`
	Type              MessageType             `json:"type"`
	MentionEveryone   bool                    `json:"mention_everyone"`
	TTS               bool                    `json:"tts"`
	Pinned            bool                    `json:"pinned"`
}

// MessageInteraction represents an executed interaction.
type MessageInteraction struct {
	Name string          `json:"name"`
	User User            `json:"user"`
	ID   InteractionID   `json:"id"`
	Type InteractionType `json:"type"`
}

// MessageChannelMention represents a mentioned channel.
type MessageChannelMention struct {
	Name    string      `json:"name"`
	ID      ChannelID   `json:"id"`
	GuildID GuildID     `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// MessageReference represents crossposted messages or replys.
type MessageReference struct {
	ID              *MessageID `json:"message_id,omitempty"`
	ChannelID       *ChannelID `json:"channel_id,omitempty"`
	GuildID         *GuildID   `json:"guild_id,omitempty"`
	FailIfNotExists bool       `json:"fail_if_not_exists"`
}

// MessageReaction represents a reaction to a message on discord.
type MessageReaction struct {
	BurstColors  []string                    `json:"burst_colors"`
	Emoji        Emoji                       `json:"emoji"`
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
	Roles       []RoleID                     `json:"roles"`
	Users       []UserID                     `json:"users"`
	RepliedUser bool                         `json:"replied_user"`
}

// MessageAttachment represents a message attachment on discord.
type MessageAttachment struct {
	Filename  string       `json:"filename"`
	URL       string       `json:"url"`
	ProxyURL  string       `json:"proxy_url"`
	ID        AttachmentID `json:"id"`
	Size      int32        `json:"size"`
	Height    int32        `json:"height"`
	Width     int32        `json:"width"`
	Ephemeral bool         `json:"ephemeral"`
}

// MessageActivity represents a message activity on discord.
type MessageActivity struct {
	PartyID string              `json:"party_id,omitempty"`
	Type    MessageActivityType `json:"type"`
}
