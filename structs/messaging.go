package structs

import (
	"encoding/json"

	"github.com/WelcomerTeam/Discord/discord"
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

type SandwichTrace map[string]discord.Int64

// SandwichPayload represents the data that is sent to consumers.
type SandwichPayload struct {
	Op       discord.GatewayOp `json:"op"`
	Data     json.RawMessage   `json:"d"`
	Sequence int32             `json:"s"`
	Type     string            `json:"t"`

	Extra    map[string]json.RawMessage `json:"__extra"`
	Metadata SandwichMetadata           `json:"__sandwich"`
	Trace    SandwichTrace              `json:"__sandwich_trace"`
}
