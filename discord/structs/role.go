package discord

// role.go represents all structures for a discord guild role.

// Role represents a role on Discord.
type Role struct {
	ID          Snowflake  `json:"id"`
	Name        string     `json:"name"`
	Color       int        `json:"color"`
	Hoist       bool       `json:"hoist"`
	Position    int        `json:"position"`
	Permissions int        `json:"permissions"`
	Managed     bool       `json:"managed"`
	Mentionable bool       `json:"mentionable"`
	Tags        []*RoleTag `json:"tags,omitempty"`
}

// RoleTag represents extra information about a role.
type RoleTag struct {
	BotID             *Snowflake `json:"bot_id,omitempty"`
	IntegrationID     *Snowflake `json:"integration_id,omitempty"`
	PremiumSubscriber *bool      `json:"premium_subscriber,omitempty"`
}
