package sandwich

import "context"

type ProducerProvider interface {
	GetProducer(ctx context.Context, applicationIdentifier, clientName string) (Producer, error)
}

type Producer interface {
	// Publish must finish using payload before returning. Implementations that
	// enqueue work asynchronously must copy the payload and any referenced data
	// they retain; the caller reuses the payload object after this call.
	Publish(ctx context.Context, shard *Shard, payload *ProducedPayload) error
	Close() error
}
