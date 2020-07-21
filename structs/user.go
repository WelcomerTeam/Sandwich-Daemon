package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// User represents a user on Discord
type User struct {
	ID            snowflake.ID `json:"id"`
	Username      string       `json:"username"`
	Discriminator string       `json:"discriminator"`
	Avatar        string       `json:"avatar"`
	Bot           bool         `json:"bot,omitempty"`
	MFAEnabled    bool         `json:"mfa_enabled,omitempty"`
	Locale        string       `json:"locale,omitempty"`
	Verified      bool         `json:"verified,omitempty"`
	Email         string       `json:"email,omitempty"`
	Flags         int          `json:"flags"`
	PremiumType   int          `json:"premium_type"`
}
