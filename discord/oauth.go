package discord

import "time"

// AuthorizationInformation represents the current oauth authorization.
type AuthorizationInformation struct {
	Expires     time.Time   `json:"expires"`
	Scopes      []string    `json:"scopes"`
	Application Application `json:"application"`
	User        User        `json:"user"`
}
