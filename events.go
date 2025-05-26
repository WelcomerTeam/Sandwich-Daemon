package sandwich

import (
	"context"
	"encoding/json"

	"github.com/WelcomerTeam/Discord/discord"
)

type ProducedPayload struct {
	discord.GatewayPayload

	Extra    map[string]json.RawMessage `json:"__extra,omitempty"`
	Metadata ProducedMetadata           `json:"__metadata,omitempty"`
	Trace    Trace                      `json:"__trace,omitempty"`
}

type ProducedMetadata struct {
	Identifier    string            `json:"i"`
	Application   string            `json:"a"`
	ApplicationID discord.Snowflake `json:"id"`
	Shard         [3]int32          `json:"s"`
}

type EventProvider interface {
	Dispatch(ctx context.Context, shard *Shard, event *discord.GatewayPayload, trace *Trace) error
}
