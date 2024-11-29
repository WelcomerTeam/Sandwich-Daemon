package discord

import "encoding/json"

// interactions.go represents the interaction objects.

// InteractionType represents the type of interaction.
type InteractionType uint16

const (
	InteractionTypePing InteractionType = 1 + iota
	InteractionTypeApplicationCommand
	InteractionTypeMessageComponent
	InteractionTypeApplicationCommandAutocomplete
	InteractionTypeModalSubmit
)

// InteractionCallbackType represents the type of interaction callbacks.
type InteractionCallbackType uint16

const (
	InteractionCallbackTypePong InteractionCallbackType = 1 + iota

	_
	_

	// InteractionCallbackTypeChannelMessageSource responds to an interaction with a message.
	InteractionCallbackTypeChannelMessageSource

	// InteractionCallbackTypeDeferredChannelMessageSource acknowledges an interaction and
	// edits a response later, users see a loading state.
	InteractionCallbackTypeDeferredChannelMessageSource

	// InteractionCallbackTypeDeferredUpdateMessage acknowledges an interaction and edits
	// a response later, users do not see a loading state.
	InteractionCallbackTypeDeferredUpdateMessage

	// InteractionCallbackTypeUpdateMessage edits the message the component was attached to.
	InteractionCallbackTypeUpdateMessage

	// InteractionCallbackTypeAutocompleteResult responds to an autocomplete interaction.
	InteractionCallbackTypeAutocompleteResult

	// InteractionCallbackTypeModal responds to an interaction with a popup modal.
	InteractionCallbackTypeModal

	// InteractionCallbackTypePremiumRequired creates an ephemeral message shown to the
	// user that ran the interaction, instructing them that whatever they tried to do requires
	// the premium benefits of your app. It also contains an "Upgrade" button to subscribe.
	InteractionCallbackTypePremiumRequired
)

// InteractionComponentType represents the type of component.
type InteractionComponentType uint16

const (
	// InteractionComponentTypeActionRow is a non-interactive container for other components.
	// You can have up to 5 action rows per message and cannot contain other action rows.
	// No extra attributes are required, just type and components.
	InteractionComponentTypeActionRow InteractionComponentType = 1 + iota
	// InteractionComponentTypeButton is an interactive component that renders in messages.
	// They can be clicked by users and must be in an action row. There is a limit of 5 buttons
	// per action row and cannot be in an action row with any select menu component.
	InteractionComponentTypeButton
	// InteractionComponentTypeStringSelect allows for users to select from predefined text options.
	InteractionComponentTypeStringSelect
	// InteractionComponentTypeTextInput allows for users to freely input text.
	InteractionComponentTypeTextInput
	InteractionComponentTypeUserInput
	InteractionComponentTypeRoleSelect
	InteractionComponentTypeMentionableSelect
	InteractionComponentTypeChannelSelect
)

// InteractionComponentStyle represents the style of a component.
type InteractionComponentStyle uint16

const (
	InteractionComponentStylePrimary InteractionComponentStyle = 1 + iota
	InteractionComponentStyleSecondary
	InteractionComponentStyleSuccess
	InteractionComponentStyleDanger
	InteractionComponentStyleLink
)

const (
	// InteractionComponentStyleShort allows for a single-line input on text inputs.
	InteractionComponentStyleShort InteractionComponentStyle = 1 + iota
	// InteractionComponentParagraph allows for a multi-line input on text inputs.
	InteractionComponentStyleParagraph
)

// Interaction represents the structure of an interaction.
type Interaction struct {
	Member         *GuildMember     `json:"member,omitempty"`
	Message        *Message         `json:"message,omitempty"`
	AppPermissions *Int64           `json:"app_permissions"`
	Data           *InteractionData `json:"data,omitempty"`
	GuildID        *GuildID         `json:"guild_id,omitempty"`
	ChannelID      *ChannelID       `json:"channel_id,omitempty"`
	User           *User            `json:"user,omitempty"`
	Token          string           `json:"token"`
	Locale         string           `json:"locale,omitempty"`
	GuildLocale    string           `json:"guild_locale,omitempty"`
	Entitlements   []Entitlement    `json:"entitlements,omitempty"`
	ID             InteractionID    `json:"id"`
	ApplicationID  ApplicationID    `json:"application_id"`
	Version        int32            `json:"version"`
	Type           InteractionType  `json:"type"`
}

// InteractionFollowup represents a follow up message containing both message and the interaction parent.
type InteractionFollowup struct {
	*Message
	*Interaction
}

// InteractionResponse represents the interaction response object.
type InteractionResponse struct {
	Data *InteractionCallbackData `json:"data,omitempty"`
	Type InteractionCallbackType  `json:"type"`
}

// InteractionData represents the structure of interaction data.
type InteractionData struct {
	TargetID      *Snowflake                `json:"target_id,omitempty"`
	Resolved      *InteractionResolvedData  `json:"resolved,omitempty"`
	GuildID       *GuildID                  `json:"guild_id,omitempty"`
	ComponentType *InteractionComponentType `json:"component_type,omitempty"`
	Name          string                    `json:"name"`
	CustomID      string                    `json:"custom_id,omitempty"`
	Options       []InteractionDataOption   `json:"options,omitempty"`
	Values        []ApplicationSelectOption `json:"values,omitempty"`
	Components    []InteractionComponent    `json:"components,omitempty"`
	Value         json.RawMessage           `json:"value,omitempty"`
	ID            ApplicationCommandID      `json:"id"`
	Type          ApplicationCommandType    `json:"type"`
	Focused       bool                      `json:"focused,omitempty"`
}

// InteractionData represents the structure of the interaction callback data.
// Not all message fields are supported, allowed fields are: tts, content
// embeds, allowed_mentions, flags, components and attachments.
type InteractionCallbackData struct {
	Content         string                           `json:"content,omitempty"`
	Title           string                           `json:"title,omitempty"`
	CustomID        string                           `json:"custom_id,omitempty"`
	Embeds          []Embed                          `json:"embeds,omitempty"`
	AllowedMentions []MessageAllowedMentions         `json:"allowed_mentions,omitempty"`
	Attachments     []MessageAttachment              `json:"attachments,omitempty"`
	Files           []File                           `json:"-"`
	Components      []InteractionComponent           `json:"components,omitempty"`
	Choices         []ApplicationCommandOptionChoice `json:"choices,omitempty"`
	Flags           uint32                           `json:"flags,omitempty"`
	TTS             bool                             `json:"tts,omitempty"`
}

// InteractionDataOption represents the structure of an interaction option.
type InteractionDataOption struct {
	Name    string                       `json:"name"`
	Value   json.RawMessage              `json:"value,omitempty"`
	Options []InteractionDataOption      `json:"options,omitempty"`
	Type    ApplicationCommandOptionType `json:"type"`
	Focused bool                         `json:"focused,omitempty"`
}

// InteractionResolvedData represents any extra payload data for an interaction.
type InteractionResolvedData struct {
	Users       map[UserID]User                    `json:"users,omitempty"`
	Members     map[UserID]GuildMember             `json:"members,omitempty"`
	Roles       map[RoleID]Role                    `json:"roles,omitempty"`
	Channels    map[ChannelID]Channel              `json:"channels,omitempty"`
	Messages    map[MessageID]Message              `json:"messages,omitempty"`
	Attachments map[AttachmentID]MessageAttachment `json:"attachments,omitempty"`
}

// InteractionComponent represents the structure of a component.
type InteractionComponent struct {
	Emoji        *Emoji                    `json:"emoji,omitempty"`
	MaxValues    *int32                    `json:"max_values,omitempty"`
	MinValues    *int32                    `json:"min_values,omitempty"`
	Placeholder  string                    `json:"placeholder,omitempty"`
	CustomID     string                    `json:"custom_id,omitempty"`
	URL          string                    `json:"url,omitempty"`
	Label        string                    `json:"label,omitempty"`
	Options      []ApplicationSelectOption `json:"options,omitempty"`
	ChannelTypes []ChannelType             `json:"channel_types,omitempty"`
	Components   []InteractionComponent    `json:"components,omitempty"`
	Disabled     bool                      `json:"disabled,omitempty"`
	Type         InteractionComponentType  `json:"type"`
	Style        InteractionComponentStyle `json:"style,omitempty"`
}

func NewInteractionComponent(componentType InteractionComponentType) *InteractionComponent {
	return &InteractionComponent{
		Type: componentType,
	}
}

func (ic *InteractionComponent) SetCustomID(customID string) *InteractionComponent {
	ic.CustomID = customID

	return ic
}

func (ic *InteractionComponent) SetDisabled(disabled bool) *InteractionComponent {
	ic.Disabled = disabled

	return ic
}

func (ic *InteractionComponent) SetStyle(style InteractionComponentStyle) *InteractionComponent {
	ic.Style = style

	return ic
}

func (ic *InteractionComponent) SetLabel(label string) *InteractionComponent {
	ic.Label = label

	return ic
}

func (ic *InteractionComponent) SetEmoji(emoji *Emoji) *InteractionComponent {
	ic.Emoji = emoji

	return ic
}

func (ic *InteractionComponent) SetURL(url string) *InteractionComponent {
	ic.URL = url

	return ic
}

func (ic *InteractionComponent) AddOption(option ApplicationSelectOption) *InteractionComponent {
	ic.Options = append(ic.Options, option)

	return ic
}

func (ic *InteractionComponent) SetPlaceholder(placeholder string) *InteractionComponent {
	ic.Placeholder = placeholder

	return ic
}

func (ic *InteractionComponent) SetMinMaxValues(minValue *int32, maxValue *int32) *InteractionComponent {
	ic.MinValues = minValue
	ic.MaxValues = maxValue

	return ic
}

func (ic *InteractionComponent) AddComponent(component InteractionComponent) *InteractionComponent {
	ic.Components = append(ic.Components, component)

	return ic
}
