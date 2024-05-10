package discord

import (
	"fmt"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
)

type Timestamp string

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t == "" {
		return []byte("null"), nil
	}

	if t != "" {
		if _, err := time.Parse(time.RFC3339, string(t)); err != nil {
			fmt.Printf("Timestamp is corrupted (is %v)\n", t)
			t = ""
		}
	}

	return sandwichjson.Marshal(string(t))
}
