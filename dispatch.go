package sandwich

import (
	"context"

	"github.com/WelcomerTeam/Discord/discord"
)

type DispatchHandler func(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) (result DispatchResult, ok bool, err error)

type Trace map[string]any

func (t *Trace) Set(key string, value any) {
	(*t)[key] = value
}

type Extra map[string]any

type DispatchResult struct {
	Data  any
	Extra Extra
}

type EventDispatchProvider interface {
	Dispatch(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) (result DispatchResult, ok bool, err error)
}
