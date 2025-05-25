package sandwich

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
)

// EventProviderWithBlacklist is an event provider that will not handle events that are in the blacklist
// and not publish events that are in the produce blacklist.

type EventProviderWithBlacklist struct {
	dispatchProvider EventDispatchProvider

	producedPayloadPool *sync.Pool
}

func NewEventProviderWithBlacklist(dispatchProvider EventDispatchProvider) *EventProviderWithBlacklist {
	return &EventProviderWithBlacklist{
		dispatchProvider: dispatchProvider,

		producedPayloadPool: &sync.Pool{
			New: func() any {
				return &ProducedPayload{}
			},
		},
	}
}

func (p *EventProviderWithBlacklist) Dispatch(ctx context.Context, shard *Shard, event *discord.GatewayPayload, trace *Trace) error {
	eventBlacklist := shard.Application.Configuration.Load().EventBlacklist

	for _, blacklistedEvent := range eventBlacklist {
		if blacklistedEvent == event.Type {
			return nil
		}
	}

	result, continuable, err := p.dispatchProvider.Dispatch(ctx, shard, event, trace)
	if err != nil {
		if !errors.Is(err, ErrNoDispatchHandler) {
			return fmt.Errorf("failed to dispatch event: %w", err)
		}
	}

	if !continuable {
		return nil
	}

	produceBlacklist := shard.Application.Configuration.Load().ProduceBlacklist

	for _, blacklistedEvent := range produceBlacklist {
		if blacklistedEvent == event.Type {
			return nil
		}
	}

	if trace == nil {
		trace = &Trace{}
	}

	packet := p.producedPayloadPool.Get().(*ProducedPayload)

	packet.GatewayPayload = *event
	packet.Extra = result.Extra
	packet.Metadata = *shard.Metadata.Load()
	packet.Trace = *trace

	packet.Trace.Set("publish", time.Now().UnixNano())

	err = shard.Application.producer.Publish(ctx, shard, packet)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.producedPayloadPool.Put(packet)

	return nil
}
