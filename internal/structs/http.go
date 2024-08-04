package structs

// IdentifyPayload represents the payload for external identifying.
type IdentifyPayload struct {
	Token          string `json:"token"`
	TokenHash      string `json:"token_hash"`
	ShardID        int32  `json:"shard_id"`
	ShardCount     int32  `json:"shard_count"`
	MaxConcurrency int32  `json:"max_concurrency"`
}

// IdentifyResponse represents the response to external identifying.
type IdentifyResponse struct {
	Message string `json:"message"`

	// If Success is false and this is passed,
	// a value of 5000 represents waiting 5 seconds.
	Wait    int32 `json:"wait"`
	Success bool  `json:"success"`
}
