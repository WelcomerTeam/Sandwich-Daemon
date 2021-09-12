package structs

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  interface{}
	Extra map[string]interface{}
}

type StateGuild struct{}
type StateGuildMembers struct{}
type StateRole struct{}
type StateEmoji struct{}
type StateUser struct{}
type StateChannel struct{}
