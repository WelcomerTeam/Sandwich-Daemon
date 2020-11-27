package structs

// TooManyRequests represents the payload of a TooManyRequests response.
type TooManyRequests struct {
	Message    string `json:"message" msgpack:"message"`
	RetryAfter int    `json:"retry_after" msgpack:"retry_after"`
	Global     bool   `json:"global" msgpack:"global"`
}

// RestTunnelAliveResponse the RestTunnel alive response payload.
type RestTunnelAliveResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Reverse bool   `json:"reverse"`
	} `json:"data"`
}
