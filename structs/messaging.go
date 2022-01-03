package structs

import (
	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
)

// SandwichMetadata represents the identification information that consumers will use.
type SandwichMetadata struct {
	Version       string `json:"v"`
	Identifier    string `json:"i"`
	Application   string `json:"a"`
	ApplicationID int64  `json:"id"`
	// ShardGroup ID, Shard ID, Shard Count
	Shard [3]int `json:"s"`
}

// SandwichPayload represents the data that is sent to consumers.
type SandwichPayload struct {
	Op       discord.GatewayOp `json:"op"`
	Data     interface{}       `json:"d"`
	Sequence int64             `json:"s"`
	Type     string            `json:"t"`

	Extra    map[string]interface{} `json:"__extra"`
	Metadata SandwichMetadata       `json:"__sandwich"`
	Trace    map[string]int         `json:"__sandwich_trace"`
}
