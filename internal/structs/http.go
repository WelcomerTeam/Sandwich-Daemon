package structs

// IdentifyPayload represents the payload for external identifying.
type IdentifyPayload struct {
	ShardID        int32  `json:"shard_id"`
	ShardCount     int32  `json:"shard_count"`
	Token          string `json:"token"`
	TokenHash      string `json:"token_hash"`
	MaxConcurrency int32  `json:"max_concurrency"`
}

// IdentifyResponse represents the response to external identifying.
type IdentifyResponse struct {
	Success bool `json:"success"`

	// If Success is false and this is passed,
	// a value of 5000 represents waiting 5 seconds.
	Wait    int32  `json:"wait"`
	Message string `json:"message"`
}
