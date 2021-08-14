package structs

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload.
type StateResult struct {
	Data  interface{}
	Extra map[string]interface{}
}
