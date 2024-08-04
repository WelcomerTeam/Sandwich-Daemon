package discord

import "time"

// AuthorizationInformation represents the current oauth authorization.
type AuthorizationInformation struct {
	Expires     time.Time   `json:"expires"`
	Application Application `json:"application"`
	User        User        `json:"user"`
	Scopes      []string    `json:"scopes"`
}
