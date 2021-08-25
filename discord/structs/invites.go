package discord

// invites.go contains all structures for invites.

// ` represents the type of an invites target
type InviteTargetType uint8

const (
	InviteTargetTypeStream InviteTargetType = 1 + iota
	InviteTargetTypeEmbeddedApplication
)
