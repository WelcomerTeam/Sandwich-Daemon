package discord

type Entitlement struct {
	UserID         *UserID         `json:"user_id,omitempty"`
	GiftCodeFlags  *GiftCodeFlags  `json:"gift_code_flags,omitempty"`
	StartsAt       *Timestamp      `json:"starts_at,omitempty"`
	EndsAt         *Timestamp      `json:"ends_at,omitempty"`
	GuildID        *GuildID        `json:"guild_id,omitempty"`
	SubscriptionID *Snowflake      `json:"subscription_id,omitempty"`
	ID             Snowflake       `json:"id"`
	SkuID          Snowflake       `json:"sku_id"`
	ApplicationID  ApplicationID   `json:"application_id"`
	Type           EntitlementType `json:"type"`
	Deleted        bool            `json:"deleted"`
}

// EntitlementParams represents the payload sent to discord.
type EntitlementParams struct {
	SkuID     Snowflake `json:"sku_id"`
	OwnerId   Snowflake `json:"owner_id"`
	OwnerType OwnerType `json:"owner_type"`
}

// EntitlementType represents the type of an entitlement.
type EntitlementType uint16

const (
	EntitlementTypeApplicationSubscription EntitlementType = 8
)

// GiftCodeFlags is undocumented, but present in the API.
type GiftCodeFlags uint16

// OwnerType represents who owns the entitlement.
type OwnerType uint16

const (
	OwnerTypeGuild OwnerType = 1
	OwnerTypeUser  OwnerType = 2
)
