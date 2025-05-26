package sandwich

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/WelcomerTeam/Discord/discord"
)

type DispatchHandler func(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) (result DispatchResult, ok bool, err error)

func NewTrace() *Trace {
	t := make(Trace)
	return &t
}

type Trace map[string]any

func (t *Trace) Set(key string, value any) *Trace {
	(*t)[key] = value

	return t
}

func NewExtra() *Extra {
	e := make(Extra)
	return &e
}

type Extra map[string]json.RawMessage

func (e *Extra) Set(key string, value any) *Extra {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("extra.Set(%s, %v): %v", key, value, err))
	}

	(*e)[key] = data

	return e
}

type DispatchResult struct {
	Data  any
	Extra *Extra
}

type EventDispatchProvider interface {
	Dispatch(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) (result DispatchResult, ok bool, err error)
}
