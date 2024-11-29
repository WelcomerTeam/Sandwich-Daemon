package discord

import "encoding/json"

// application.go represents the application object and integrations.

// ApplicationTeamMemberState represents the state of a member in a team.
type ApplicationTeamMemberState uint16

const (
	ApplicationTeamMemberStateInvited ApplicationTeamMemberState = 1 + iota
	ApplicationTeamMemberStateAccepted
)

// ApplicationCommandType represents the different types of application command.
type ApplicationCommandType uint16

const (
	ApplicationCommandTypeChatInput ApplicationCommandType = 1 + iota
	ApplicationCommandTypeUser
	ApplicationCommandTypeMessage
)

// ApplicationCommandOptionType represents the different types of options.
type ApplicationCommandOptionType uint16

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
	ApplicationCommandOptionTypeAttachment
)

// ApplicationCommandPermissionType represents the target for a command permission.
type ApplicationCommandPermissionType uint16

const (
	ApplicationCommandPermissionTypeRole ApplicationCommandPermissionType = 1 + iota
	ApplicationCommandPermissionTypeUser
)

// IntegrationType represents the type of integration.
type IntegrationType string

const (
	IntegrationTypeTwitch  IntegrationType = "twitch"
	IntegrationTypeYoutube IntegrationType = "youtube"
	IntegrationTypeDiscord IntegrationType = "discord"
)

// IntegrationExpireBehavior represents the integration expiration.
type IntegrationExpireBehavior uint16

const (
	IntegrationExpireBehaviorRemoveRole IntegrationExpireBehavior = iota
	IntegrationExpireBehaviorKick
)

// Application response from REST.
type Application struct {
	Owner               *User            `json:"owner,omitempty"`
	Bot                 *User            `json:"bot,omitempty"`
	PrimarySKUID        *Snowflake       `json:"primary_sku_id,omitempty"`
	GuildID             *GuildID         `json:"guild_id,omitempty"`
	Team                *ApplicationTeam `json:"team,omitempty"`
	PrivacyPolicyURL    string           `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL   string           `json:"terms_of_service,omitempty"`
	VerifyKey           string           `json:"verify_key"`
	Description         string           `json:"description"`
	Icon                string           `json:"icon,omitempty"`
	Slug                string           `json:"slug,omitempty"`
	CoverImage          string           `json:"cover_image,omitempty"`
	Name                string           `json:"name"`
	RPCOrigins          []string         `json:"rpc_origins,omitempty"`
	ID                  ApplicationID    `json:"id"`
	Flags               int32            `json:"flags,omitempty"`
	BotRequireCodeGrant bool             `json:"bot_require_code_grant"`
	BotPublic           bool             `json:"bot_public"`
}

// ApplicationTeam represents the team of an application.
type ApplicationTeam struct {
	Icon        string                  `json:"icon,omitempty"`
	Name        string                  `json:"name"`
	Members     []ApplicationTeamMember `json:"members"`
	ID          ApplicationTeamID       `json:"id"`
	OwnerUserID Snowflake               `json:"owner_user_id"`
}

// ApplicationTeamMembers represents a member of a team.
type ApplicationTeamMember struct {
	Permissions     StringList                 `json:"permissions"`
	User            User                       `json:"user"`
	TeamID          Snowflake                  `json:"team_id"`
	MembershipState ApplicationTeamMemberState `json:"membership_state"`
}

// ApplicationCommand represents an application's command.
type ApplicationCommand struct {
	DefaultMemberPermission  *Int64                     `json:"default_member_permissions,omitempty"`
	Type                     *ApplicationCommandType    `json:"type,omitempty"`
	ApplicationID            *ApplicationID             `json:"application_id,omitempty"`
	GuildID                  *GuildID                   `json:"guild_id,omitempty"`
	NameLocalizations        map[string]string          `json:"name_localizations,omitempty"`
	DescriptionLocalizations map[string]string          `json:"description_localizations,omitempty"`
	ID                       *ApplicationCommandID      `json:"id,omitempty"`
	DMPermission             *bool                      `json:"dm_permission,omitempty"`
	DefaultPermission        *bool                      `json:"default_permission,omitempty"`
	Name                     string                     `json:"name"`
	Description              string                     `json:"description,omitempty"`
	Options                  []ApplicationCommandOption `json:"options,omitempty"`
	Version                  Int64                      `json:"version,omitempty"`
}

// GuildApplicationCommandPermissions represent a guilds application permissions.
type GuildApplicationCommandPermissions struct {
	Permissions   []ApplicationCommandPermissions `json:"permissions"`
	ID            ApplicationCommandPermissionsID `json:"id"`
	ApplicationID ApplicationID                   `json:"application_id"`
	GuildID       GuildID                         `json:"guild_id"`
}

// ApplicationCommandPermissions represents the rules for enabling or disabling a command.
type ApplicationCommandPermissions struct {
	ID      ApplicationCommandPermissionsID  `json:"id"`
	Type    ApplicationCommandPermissionType `json:"type"`
	Allowed bool                             `json:"permission"`
}

// ApplicationCommandOption represents the options for an application command.
type ApplicationCommandOption struct {
	MinValue                 *int32                           `json:"min_value,omitempty"`
	Autocomplete             *bool                            `json:"autocomplete,omitempty"`
	NameLocalizations        map[string]string                `json:"name_localizations,omitempty"`
	MaxLength                *int32                           `json:"max_length,omitempty"`
	DescriptionLocalizations map[string]string                `json:"description_localizations,omitempty"`
	MinLength                *int32                           `json:"min_length,omitempty"`
	MaxValue                 *int32                           `json:"max_value,omitempty"`
	Description              string                           `json:"description,omitempty"`
	Name                     string                           `json:"name"`
	ChannelTypes             []ChannelType                    `json:"channel_types,omitempty"`
	Options                  []ApplicationCommandOption       `json:"options,omitempty"`
	Choices                  []ApplicationCommandOptionChoice `json:"choices,omitempty"`
	Required                 bool                             `json:"required,omitempty"`
	Type                     ApplicationCommandOptionType     `json:"type"`
}

// ApplicationCommandOptionChoice represents the different choices.
type ApplicationCommandOptionChoice struct {
	Name              string            `json:"name"`
	NameLocalizations map[string]string `json:"name_localizations,omitempty"`
	Value             json.RawMessage   `json:"value"`
}

// ApplicationSelectOption represents the structure of select options.
type ApplicationSelectOption struct {
	Emoji       *Emoji `json:"emoji,omitempty"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Default     bool   `json:"default,omitempty"`
}

// Integration represents the structure of an integration.
type Integration struct {
	SyncedAt          Timestamp                  `json:"synced_at,omitempty"`
	ExpireBehavior    *IntegrationExpireBehavior `json:"expire_behavior,omitempty"`
	User              *User                      `json:"user,omitempty"`
	Application       *Application               `json:"application,omitempty"`
	GuildID           *GuildID                   `json:"guild_id,omitempty"`
	RoleID            *RoleID                    `json:"role_id,omitempty"`
	Account           IntegrationAccount         `json:"account"`
	Type              IntegrationType            `json:"type"`
	Name              string                     `json:"name"`
	ID                IntegrationID              `json:"id"`
	ExpireGracePeriod int32                      `json:"expire_grace_period,omitempty"`
	SubscriberCount   int32                      `json:"subscriber_count,omitempty"`
	EnableEmoticons   bool                       `json:"enable_emoticons"`
	Syncing           bool                       `json:"syncing"`
	Revoked           bool                       `json:"revoked"`
	Enabled           bool                       `json:"enabled"`
}

// IntegrationAccount represents the account of the integration.
type IntegrationAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
