package discord

// role.go represents all structures for a discord guild role.

// Role represents a role on discord.
type Role struct {
	GuildID      *GuildID `json:"guild_id,omitempty"`
	Tags         *RoleTag `json:"tags,omitempty"`
	Name         string   `json:"name"`
	Icon         string   `json:"icon,omitempty"`
	UnicodeEmoji string   `json:"unicode_emoji,omitempty"`
	ID           RoleID   `json:"id"`
	Permissions  Int64    `json:"permissions"`
	Color        int32    `json:"color"`
	Position     int32    `json:"position"`
	Hoist        bool     `json:"hoist"`
	Managed      bool     `json:"managed"`
	Mentionable  bool     `json:"mentionable"`
}

// RoleTag represents extra information about a role.
type RoleTag struct {
	BotID                 *UserID        `json:"bot_id"`
	IntegrationID         *IntegrationID `json:"integration_id"`
	PremiumSubscriber     *bool          `json:"premium_subscriber"`
	SubscriptionListingID *Snowflake     `json:"subscription_listing_id"`
	AvailableForPurchase  *bool          `json:"available_for_purchase"`
	GuildConnections      *bool          `json:"guild_connections"`
}
