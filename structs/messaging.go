package structs

import (
	"github.com/WelcomerTeam/Discord/discord"
	discord_structs "github.com/WelcomerTeam/Discord/structs"
	jsoniter "github.com/json-iterator/go"
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

// SandwichPayload represents the data that is sent to consumers.
type SandwichPayload struct {
	Op       discord_structs.GatewayOp `json:"op"`
	Data     jsoniter.RawMessage       `json:"d"`
	Sequence int32                     `json:"s"`
	Type     string                    `json:"t"`

	Extra    map[string]jsoniter.RawMessage `json:"__extra"`
	Metadata SandwichMetadata               `json:"__sandwich"`
	Trace    map[string]int                 `json:"__sandwich_trace"`
}
