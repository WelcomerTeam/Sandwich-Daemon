package structs

import (
	"github.com/WelcomerTeam/Discord/discord"
	jsoniter "github.com/json-iterator/go"
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

// SandwichPayload represents the data that is sent to consumers.
type SandwichPayload struct {
	Op       discord.GatewayOp   `json:"op"`
	Data     jsoniter.RawMessage `json:"d"`
	Sequence int32               `json:"s"`
	Type     string              `json:"t"`

	Extra    *csmap.CsMap[string, jsoniter.RawMessage] `json:"__extra,omitempty"`
	Metadata SandwichMetadata                          `json:"__sandwich"`
	Trace    SandwichTrace                             `json:"__sandwich_trace"`
}
