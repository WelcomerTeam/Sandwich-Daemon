package sandwich

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
)

type ProducedPayload struct {
	discord.GatewayPayload

	Extra    map[string]any   `json:"__extra"`
	Metadata ProducedMetadata `json:"__metadata"`
	Trace    Trace            `json:"__trace"`
}

type ProducedMetadata struct {
	Identifier    string            `json:"i"`
	Application   string            `json:"a"`
	ApplicationID discord.Snowflake `json:"id"`
	Shard         [3]int32          `json:"s"`
}

type EventProvider interface {
	Dispatch(ctx context.Context, shard *Shard, event discord.GatewayPayload, trace *Trace) error
}

// EventProviderWithBlacklist is an event provider that will not handle events that are in the blacklist
// and not publish events that are in the produce blacklist.

type EventProviderWithBlacklist struct {
	dispatchProvider EventDispatchProvider
}

func NewEventProviderWithBlacklist(dispatchProvider EventDispatchProvider) *EventProviderWithBlacklist {
	return &EventProviderWithBlacklist{
		dispatchProvider: dispatchProvider,
	}
}

func (p *EventProviderWithBlacklist) Dispatch(ctx context.Context, shard *Shard, event discord.GatewayPayload, trace *Trace) error {
	eventBlacklist := shard.manager.configuration.Load().EventBlacklist

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

	produceBlacklist := shard.manager.configuration.Load().ProduceBlacklist

	for _, blacklistedEvent := range produceBlacklist {
		if blacklistedEvent == event.Type {
			return nil
		}
	}

	configuration := shard.manager.configuration.Load()

	packet := ProducedPayload{
		GatewayPayload: event,
		Extra:          result.Extra,
		Metadata: ProducedMetadata{
			Identifier:    configuration.ProducerIdentifier,
			Application:   configuration.ApplicationIdentifier,
			ApplicationID: shard.manager.user.Load().ID,
			Shard: [3]int32{
				0,
				shard.shardID,
				shard.manager.shardCount.Load(),
			},
		},
		Trace: *trace,
	}

	packet.Trace.Set("publish", time.Now().UnixNano())

	err = shard.manager.producer.Publish(ctx, shard, packet)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
