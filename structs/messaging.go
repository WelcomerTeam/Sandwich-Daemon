package structs

import (
	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
	jsoniter "github.com/json-iterator/go"
)

// SandwichMetadata represents the identification information that consumers will use.
type SandwichMetadata struct {
	Version       string            `json:"v"`
	Identifier    string            `json:"i"`
	Application   string            `json:"a"`
	ApplicationID discord.Snowflake `json:"id,string"`
	// ShardGroup ID, Shard ID, Shard Count
	Shard [3]int `json:"s"`
}

// SandwichPayload represents the data that is sent to consumers.
type SandwichPayload struct {
	Op       discord.GatewayOp   `json:"op"`
	Data     jsoniter.RawMessage `json:"d"`
	Sequence int64               `json:"s,string"`
	Type     string              `json:"t"`

	Extra    map[string]jsoniter.RawMessage `json:"__extra"`
	Metadata SandwichMetadata               `json:"__sandwich"`
	Trace    map[string]int                 `json:"__sandwich_trace"`
}
