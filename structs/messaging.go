package structs

// StateResult represents the data a state handler would return which would be converted to
// a sandwich payload
type StateResult struct {
	Data  interface{}
	Extra map[string]interface{}
}

// SandwichPayload represents the data that is sent to consumers
type SandwichPayload struct {
	ReceivedPayload

	Data  interface{}            `json:"d,omitempty" msgpack:"d,omitempty"`
	Extra map[string]interface{} `json:"e,omitempty" msgpack:"e,omitempty"`

	Metadata SandwichMetadata `json:"__sandwich" msgpack:"__sandwich"`
	Trace    map[string]int   `json:"__trace,omitempty" msgpack:"__trace,omitempty"`
}

// SandwichMetadata represents the identification information that consumers will use
type SandwichMetadata struct {
	Version    string `json:"v" msgpack:"v"`
	Identifier string `json:"i" msgpack:"i"`
	Shard      [3]int `json:"s,omitempty" msgpack:"s,omitempty"` // ShardGroup ID, Shard ID, Shard Count
}

// MessagingStatusUpdate represents a shard status update.
type MessagingStatusUpdate struct {
	ShardID int   `msgpack:"shard,omitempty"`
	Status  int32 `msgpack:"status"`
}
