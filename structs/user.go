package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// User represents a user on Discord
type User struct {
	ID            snowflake.ID `json:"id" msgpack:"id"`
	Username      string       `json:"username" msgpack:"username"`
	Discriminator string       `json:"discriminator" msgpack:"discriminator"`
	Avatar        string       `json:"avatar" msgpack:"avatar"`
	Bot           bool         `json:"bot,omitempty" msgpack:"bot,omitempty"`
	MFAEnabled    bool         `json:"mfa_enabled,omitempty" msgpack:"mfa_enabled,omitempty"`
	Locale        string       `json:"locale,omitempty" msgpack:"locale,omitempty"`
	Verified      bool         `json:"verified,omitempty" msgpack:"verified,omitempty"`
	Email         string       `json:"email,omitempty" msgpack:"email,omitempty"`
	Flags         int          `json:"flags" msgpack:"flags"`
	PremiumType   int          `json:"premium_type" msgpack:"premium_type"`
}
