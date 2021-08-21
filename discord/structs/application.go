package discord

import (
	"time"

	"github.com/WelcomerTeam/RealRock/snowflake"
)

// application.go represents the application object and slash command interactions.

// ApplicationTeamMemberState represents the state of a member in a team.
type ApplicationTeamMemberState int8

const (
	ApplicationTeamMemberStateInvited ApplicationTeamMemberState = 1 + iota
	ApplicationTeamMemberStateAccepted
)

// ApplicationCommandType represents the different types of application command.
type ApplicationCommandType int8

const (
	ApplicationCommandTypeChatInput ApplicationCommandType = 1 + iota
	ApplicationCommandTypeUser
	ApplicationCommandTypeMessage
)

// ApplicationCommandOptionType represents the different types of options.
type ApplicationCommandOptionType int8

const (
	ApplicationCommandOptionTypeSubCommand ApplicationCommandOptionType = 1 + iota
	ApplicationCommandOptionTypeSubCommandGroup
	ApplicationCommandOptionTypeString
	ApplicationCommandOptionTypeInteger
	ApplicationCommandOptionTypeBoolean
	ApplicationCommandOptionTypeUser
	ApplicationCommandOptionTypeChannel
	ApplicationCommandOptionTypeRole
	ApplicationCommandOptionTypeMentionable
	ApplicationCommandOptionTypeNumber
)

// InteractionType represents the type of interaction.
type InteractionType int8

const (
	InteractionTypePing InteractionType = 1 + iota
	InteractionTypeApplicationCommand
	InteractionTypeMessageComponent
)

// IntegrationType represents the type of integration
type IntegrationType string

const (
	IntegrationTypeTwitch  IntegrationType = "twitch"
	IntegrationTypeYoutube IntegrationType = "youtube"
	IntegrationTypeDiscord IntegrationType = "discord"
)

// IntegrationExpireBehavior represents the integration expiration
type IntegrationExpireBehavior int8

const (
	IntegrationExpireBehaviorRemoveRole IntegrationExpireBehavior = iota
	IntegrationExpireBehaviorKick
)

// Application response from REST.
type Application struct {
	ID          snowflake.ID `json:"id"`
	Name        string       `json:"name"`
	Icon        string       `json:"icon,omitempty"`
	Description string       `json:"description"`

	RPCOrigins          []string         `json:"rpc_origins,omitempty"`
	BotPublic           bool             `json:"bot_public"`
	BotRequireCodeGrant bool             `json:"bot_require_code_grant"`
	TermsOfServiceURL   string           `json:"terms_of_service,omitempty"`
	PrivacyPolicyURL    string           `json:"privacy_policy_url,omitempty"`
	Owner               *PartialUser     `json:"owner,omitempty"`
	Summary             string           `json:"summary,omitempty"`
	VerifyKey           string           `json:"verify_key,omitempty"`
	Team                *ApplicationTeam `json:"team,omitempty"`
	GuildID             snowflake.ID     `json:"guild_id,omitempty"`

	PrimarySKUID snowflake.ID `json:"primary_sku_id,omitempty"`
	Slug         string       `json:"slug,omitempty"`
	CoverImage   string       `json:"cover_image,omitempty"`
	Flags        int64        `json:"flags"`
}

// ApplicationTeam represents the team of an application.
type ApplicationTeam struct {
	Icon        string                   `json:"icon,omitempty"`
	ID          snowflake.ID             `json:"id"`
	Members     []*ApplicationTeamMember `json:"members`
	Name        string                   `json:"name"`
	OwnerUserID snowflake.ID             `json:"owner_user_id"`
}

// ApplicationTeamMembers represents a member of a team.
type ApplicationTeamMember struct {
	MembershipState ApplicationTeamMemberState `json:"membership_state"`
	Permissions     []string                   `json:"permissions"`
	TeamID          snowflake.ID               `json:"team_id"`
	User            *PartialUser               `json:"user"`
}

// ApplicationCommand represents an application's command.
type ApplicationCommand struct {
	ID                snowflake.ID                `json:"id"`
	Type              *ApplicationCommandType     `json:"type,omitempty"`
	ApplicationID     snowflake.ID                `json:"application_id"`
	GuildID           *snowflake.ID               `json:"guild_id,omitempty"`
	Name              string                      `json:"name"`
	Description       string                      `json:"description"`
	Options           []*ApplicationCommandOption `json:"options,omitempty"`
	DefaultPermission *bool                       `json:"default_permission"`
}

// ApplicationCommandOption represents the options for an application command.
type ApplicationCommandOption struct {
	Type        ApplicationCommandOptionType      `json:"type"`
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Required    *bool                             `json:"required,omitempty"`
	Choices     []*ApplicationCommandOptionChoice `json:"choices,omitempty"`
	Options     []*ApplicationCommandOption       `json:"options,omitempty"`
}

// ApplicationCommandOptionChoice represents the different choices.
type ApplicationCommandOptionChoice struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Interaction represents the structure of an interaction.
type Interaction struct {
	ID            snowflake.ID     `json:"id"`
	ApplicationID snowflake.ID     `json:"application_id"`
	Type          *InteractionType `json:"type,omitempty"`
	Data          *InteractionData `json:"data,omitempty"`

	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	Member    *GuildMember `json:"member,omitempty"`
	User      *User        `json:"user,omitempty"`
	Token     string       `json:"token"`
	Version   int          `json:"version"`
	Message   *Message     `json:"message,omitempty"`
}

// InteractionData represents the structure of interaction data.
type InteractionData struct {
	ID            snowflake.ID              `json:"id"`
	Name          string                    `json:"name"`
	Type          ApplicationCommandType    `json:"type"`
	Resolved      *InteractionResolvedData  `json:"resolved,omitempty"`
	Options       []InteractionDataOption   `json:"option,omitempty"`
	CustomID      *string                   `json:"custom_id,omitempty"`
	ComponentType *ApplicationCommandType   `json:"component_type,omitempty"`
	Values        []ApplicationSelectOption `json:"values,omitempty"`
	TargetID      *snowflake.ID             `json:"target_id,omitempty"`
}

// InteractionDataOption represents the structure of an interaction option.
type InteractionDataOption struct {
	Name    string                       `json:"name"`
	Type    ApplicationCommandOptionType `json:"type"`
	Value   interface{}                  `json:"value"`
	Options []InteractionDataOption      `json:"options,omitempty"`
}

// InteractionResolvedData represents any extra payload data for an interaction.
type InteractionResolvedData struct {
	Users    []*User           `json:"users,omitempty"`
	Members  []*PartialMember  `json:"members,omitempty"`
	Roles    []*Role           `json:"roles,omitempty"`
	Channels []*PartialChannel `json:"channels,omitempty"`
	Messages []*Message        `json:"messages,omitempty"`
}

// ApplicationSelectOption represents the structure of select options.
type ApplicationSelectOption struct {
	Label       string        `json:"label"`
	Value       string        `json:"value"`
	Description *string       `json:"description,omitempty"`
	Emoji       *PartialEmoji `json:"emoji,omitempty"`
	Default     *bool         `json:"default,omitempty"`
}

// Integration represents the structure of an integration.
type Integration struct {
	ID              snowflake.ID    `json:"id"`
	Name            string          `json:"name"`
	Type            IntegrationType `json:"type"`
	Enabled         bool            `json:"enabled"`
	Syncing         *bool           `json:"syncing`
	RoleID          *snowflake.ID   `json:"role_id,omitempty"`
	EnableEmoticons *bool           `json:"enable_emoticons,omitempty"`

	ExpireBehavior    *IntegrationExpireBehavior `json:"expire_behavior,omitempty"`
	ExpireGracePeriod int                        `json:"expire_grace_period"`
	User              *User                      `json:"user,omitempty"`
	Account           IntegrationAccount         `json:"account"`
	SyncedAt          *time.Time                 `json:"synced_at,omitempty"`
	SubscriberCount   int                        `json:"subscriber_count,omitempty"`
	Revoked           *bool                      `json:"revoked,omitempty"`
	Application       *Application               `json:"application,omitempty"`
}

// IntegrationAccount represents the account of the integration.
type IntegrationAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
