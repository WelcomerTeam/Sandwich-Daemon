package sandwich

import (
	"context"

	"github.com/WelcomerTeam/Discord/discord"
)

var dispatchHandlers = make(map[string]DispatchHandler)

type BuiltinDispatchProvider struct {
	allowEventPassthrough bool
	dispatchHandlers      map[string]DispatchHandler
}

func NewBuiltinDispatchProvider(allowEventPassthrough bool) *BuiltinDispatchProvider {
	return &BuiltinDispatchProvider{
		allowEventPassthrough: allowEventPassthrough,
		dispatchHandlers:      dispatchHandlers,
	}
}

// Dispatch dispatches an event to the appropriate handler.
func (p *BuiltinDispatchProvider) Dispatch(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) (DispatchResult, bool, error) {
	if handler, ok := p.dispatchHandlers[msg.Type]; ok {
		return handler(ctx, shard, msg, trace)
	}

	// When event passthrough is allowed, we will allow the event to be published even
	// if there is no dispatch handler for it. We will just return the original event data.
	if p.allowEventPassthrough {
		return DispatchResult{
			Data:  msg.Data,
			Extra: nil,
		}, true, nil
	}

	return DispatchResult{nil, nil}, false, ErrNoDispatchHandler
}

func registerDispatchHandler(eventType string, handler DispatchHandler) {
	dispatchHandlers[eventType] = handler
}
