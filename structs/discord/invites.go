package structs

// InviteTargetUserType represents the type of user targeted for this invite.
type InviteTargetUserType int

// Targer user type.
const (
	TargetUserTypeStream InviteTargetUserType = iota + 1
)

// Invite represents an invite on Discord.
type Invite struct {
	Code                     string               `json:"code" msgpack:"code"`
	Guild                    *Guild               `json:"guild" msgpack:"guild"`
	Channel                  *Channel             `json:"channel" msgpack:"channel"`
	Inviter                  *User                `json:"inviter,omitempty" msgpack:"inviter,omitempty"`
	TargetUser               *User                `json:"target_user,omitempty" msgpack:"target_user,omitempty"`
	TargetUserType           InviteTargetUserType `json:"target_user_type,omitempty" msgpack:"target_user_type,omitempty"`
	ApproximateMemberCount   int                  `json:"approximate_member_count,omitempty" msgpack:"approximate_member_count,omitempty"`
	ApproximatePresenceCount int                  `json:"approximate_presence_count,omitempty" msgpack:"approximate_presence_count,omitempty"`
}
