package sandwich

import "context"

type ProducerProvider interface {
	GetProducer(ctx context.Context, managerIdentifier, clientName string) (Producer, error)
}

type Producer interface {
	Publish(ctx context.Context, shard *Shard, payload ProducedPayload) error
	Close() error
}
