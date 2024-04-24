package discord

// role.go represents all structures for a discord guild role.

// Role represents a role on discord.
type Role struct {
	GuildID      *Snowflake `json:"guild_id,omitempty"`
	Tags         *RoleTag   `json:"tags,omitempty"`
	Name         string     `json:"name"`
	Icon         string     `json:"icon,omitempty"`
	UnicodeEmoji string     `json:"unicode_emoji,omitempty"`
	ID           Snowflake  `json:"id"`
	Permissions  Int64      `json:"permissions"`
	Color        int32      `json:"color"`
	Position     int32      `json:"position"`
	Hoist        bool       `json:"hoist"`
	Managed      bool       `json:"managed"`
	Mentionable  bool       `json:"mentionable"`
}

// RoleTag represents extra information about a role.
type RoleTag struct {
	BotID                 *Snowflake `json:"bot_id"`
	IntegrationID         *Snowflake `json:"integration_id"`
	PremiumSubscriber     *bool      `json:"premium_subscriber"`
	SubscriptionListingID *Snowflake `json:"subscription_listing_id"`
	AvailableForPurchase  *bool      `json:"available_for_purchase"`
	GuildConnections      *bool      `json:"guild_connections"`
}
