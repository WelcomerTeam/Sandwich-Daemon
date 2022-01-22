package structs

// IdentifyPayload represents the payload for external identifying.
type IdentifyPayload struct {
	ShardID        int    `json:"shard_id"`
	ShardCount     int    `json:"shard_count"`
	Token          string `json:"token"`
	TokenHash      string `json:"token_hash"`
	MaxConcurrency int    `json:"max_concurrency"`
}

// IdentifyResponse represents the response to external identifying.
type IdentifyResponse struct {
	Success bool `json:"success"`

	// If Success is false and this is passed,
	// a value of 5000 represents waiting 5 seconds.
	Wait    int    `json:"wait"`
	Message string `json:"message"`
}
