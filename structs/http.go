package structs

// TooManyRequests represents the payload of a TooManyRequests response
type TooManyRequests struct {
	Message    string `json:"message"`
	RetryAfter int    `json:"retry_after"`
	Global     bool   `json:"global"`
}
