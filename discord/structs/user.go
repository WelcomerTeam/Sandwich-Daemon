package discord

import "github.com/WelcomerTeam/RealRock/snowflake"

// user.go represents all structures for a discord user.

// UserFlags represents the flags on a user's account.
type UserFlags int

// User flags.
const (
	UserFlagsNone UserFlags = 1 << iota
	UserFlagsDiscordEmployee
	UserFlagsPartneredServerOwner
	UserFlagsHypeSquadEvents
	UserFlagsBugHunterLevel1
	UserFlagsHouseBravery
	UserFlagsHouseBrilliance
	UserFlagsHouseBalance
	UserFlagsEarlySupporter
	UserFlagsTeamUser
	UserFlagsSystem
	UserFlagsBugHunterLevel2
	UserFlagsVerifiedBot
	UserFlagsEarlyVerifiedBotDeveloper
)

// UserPremiumType represents the type of Nitro on a user's account.
type UserPremiumType int

// User premium type.
const (
	UserPremiumTypeNone UserPremiumType = iota
	UserPremiumTypeNitroClassic
	UserPremiumTypeNitro
)

// User represents a user on Discord.
type User struct {
	ID            snowflake.ID    `json:"id" msgpack:"id"`
	Username      string          `json:"username" msgpack:"username"`
	Discriminator string          `json:"discriminator" msgpack:"discriminator"`
	Avatar        string          `json:"avatar" msgpack:"avatar"`
	Bot           bool            `json:"bot,omitempty" msgpack:"bot,omitempty"`
	System        bool            `json:"system,omitempty" msgpack:"system,omitempty"`
	MFAEnabled    bool            `json:"mfa_enabled,omitempty" msgpack:"mfa_enabled,omitempty"`
	Locale        string          `json:"locale,omitempty" msgpack:"locale,omitempty"`
	Verified      bool            `json:"verified,omitempty" msgpack:"verified,omitempty"`
	Email         string          `json:"email,omitempty" msgpack:"email,omitempty"`
	Flags         UserFlags       `json:"flags,omitempty" msgpack:"flags,omitempty"`
	PremiumType   UserPremiumType `json:"premium_type,omitempty" msgpack:"premium_type,omitempty"`
	PublicFlags   UserFlags       `json:"public_flags,omitempty" msgpack:"public_flags,omitempty"`
}
