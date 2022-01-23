package discord

// http.go represents the structures of common endpoints we use.

// Gateway represents a GET /gateway response.
type Gateway struct {
	URL string `json:"url"`
}

// GatewayBot represents a GET /gateway/bot response.
type GatewayBot struct {
	URL               string `json:"url"`
	Shards            int32  `json:"shards"`
	SessionStartLimit struct {
		Total          int32 `json:"total"`
		Remaining      int32 `json:"remaining"`
		ResetAfter     int32 `json:"reset_after"`
		MaxConcurrency int32 `json:"max_concurrency"`
	} `json:"session_start_limit"`
}

// TooManyRequests represents the payload of a TooManyRequests response.
type TooManyRequests struct {
	Message    string `json:"message"`
	RetryAfter int32  `json:"retry_after"`
	Global     bool   `json:"global"`
}

// CreateDMChannel create a new DM channel with a user. Returns a DM channel object.
type CreateDMChannel struct {
	RecipientID Snowflake `json:"recipient_id"`
}
