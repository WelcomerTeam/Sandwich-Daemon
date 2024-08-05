package structs

import (
	"encoding/json"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

// SandwichMetadata represents the identification information that consumers will use.
type SandwichMetadata struct {
	Version       string            `json:"v"`
	Identifier    string            `json:"i"`
	Application   string            `json:"a"`
	ApplicationID discord.Snowflake `json:"id"`
	// ShardGroup ID, Shard ID, Shard Count
	Shard [3]int32 `json:"s"`
}

type SandwichTrace = *csmap.CsMap[string, discord.Int64]

// Used as a key for virtual shard dispatches etc., must be set for all events
type EventDispatchIdentifier struct {
	GuildID        *discord.Snowflake
	GloballyRouted bool // Whether or not the event should be globally routed
}

// SandwichPayload represents the data that is sent to consumers.
type SandwichPayload struct {
	Extra    map[string]json.RawMessage `json:"__extra,omitempty"`
	Metadata *SandwichMetadata          `json:"__sandwich"`
	Trace    SandwichTrace              `json:"__sandwich_trace"`
	Type     string                     `json:"t"`

	Data                    json.RawMessage          `json:"d"`
	Sequence                int32                    `json:"s"`
	Op                      discord.GatewayOp        `json:"op"`
	EventDispatchIdentifier *EventDispatchIdentifier `json:"-"`
}
