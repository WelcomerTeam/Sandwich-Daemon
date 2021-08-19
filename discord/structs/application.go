package discord

import "github.com/WelcomerTeam/RealRock/snowflake"

// application.go represents the application object.

// ApplicationTeamMemberState represents the state of a member
// in a team.
type ApplicationTeamMemberState int8

// Application team member states.
const (
	ApplicationTeamMemberStateInvited ApplicationTeamMemberState = 1 + iota
	ApplicationTeamMemberStateAccepted
)

// Application response from REST.
type Application struct {
	ID                  snowflake.ID     `json:"id"`
	Name                string           `json:"name"`
	Icon                string           `json:"icon,omitempty"`
	Description         string           `json:"description"`
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
	PrimarySKUID        snowflake.ID     `json:"primary_sku_id,omitempty"`
	Slug                string           `json:"slug,omitempty"`
	CoverImage          string           `json:"cover_image,omitempty"`
	Flags               int64            `json:"flags"`
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
