package discord

import "github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"

// user.go represents all structures for a discord user.

// UserFlags represents the flags on a user's account.
type UserFlags uint32

// User flags.
const (
	UserFlagsDiscordEmployee UserFlags = 1 << iota
	UserFlagsPartneredServerOwner
	UserFlagsHypeSquadEvents
	UserFlagsBugHunterLevel1
	_
	_
	UserFlagsHouseBravery
	UserFlagsHouseBrilliance
	UserFlagsHouseBalance
	UserFlagsEarlySupporter
	UserFlagsTeamUser
	_
	_
	_
	UserFlagsBugHunterLevel2
	_
	UserFlagsVerifiedBot
	UserFlagsVerifiedDeveloper
	UserFlagsCertifiedModerator
	UserFlagsBotHTTPInteractions
	_
	_
	UserFlagsActiveDeveloper
)

// UserPremiumType represents the type of Nitro on a user's account.
type UserPremiumType int

// User premium type.
const (
	UserPremiumTypeNone UserPremiumType = iota
	UserPremiumTypeNitroClassic
	UserPremiumTypeNitro
)

// User represents a user on discord.
type User struct {
	DMChannelID      *ChannelID      `json:"dm_channel_id"`
	Banner           string          `json:"banner,omitempty"`
	GlobalName       string          `json:"global_name"`
	Avatar           *string         `json:"avatar"`
	AvatarDecoration *string         `json:"avatar_decoration,omitempty"`
	Username         string          `json:"username"`
	Discriminator    string          `json:"discriminator"`
	Locale           string          `json:"locale,omitempty"`
	Email            string          `json:"email,omitempty"`
	ID               UserID          `json:"id"`
	PremiumType      UserPremiumType `json:"premium_type"`
	Flags            UserFlags       `json:"flags"`
	AccentColor      int32           `json:"accent_color"`
	PublicFlags      UserFlags       `json:"public_flags"`
	MFAEnabled       bool            `json:"mfa_enabled"`
	Verified         bool            `json:"verified"`
	Bot              bool            `json:"bot"`
	System           bool            `json:"system"`
}

// Used to avoid a marshal loop.
type marshalUser User

func (u User) MarshalJSON() ([]byte, error) {
	// Patch for discriminator
	if u.Discriminator == "" {
		u.Discriminator = "0"
	}

	return sandwichjson.Marshal(marshalUser(u))
}

// ClientUser aliases User to provide current user specific methods.
type ClientUser User
