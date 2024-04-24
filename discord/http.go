package discord

import (
	"io"
)

// http.go represents the structures of common endpoints we use.

// File stores information about a file sent in a message.
type File struct {
	Reader      io.Reader
	Name        string
	ContentType string
}

// Gateway represents a GET /gateway response.
type GatewayResponse struct {
	URL string `json:"url"`
}

// GatewayBot represents a GET /gateway/bot response.
type GatewayBotResponse struct {
	URL               string `json:"url"`
	Shards            int32  `json:"shards"`
	SessionStartLimit struct {
		Total          int32 `json:"total"`
		Remaining      int32 `json:"remaining"`
		ResetAfter     int32 `json:"reset_after"`
		MaxConcurrency int32 `json:"max_concurrency"`
	} `json:"session_start_limit"`
}
