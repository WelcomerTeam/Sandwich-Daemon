package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	sandwich "github.com/WelcomerTeam/Sandwich-Daemon"
)

// Replace this with whatever PUBSUB/implementation you want to use.

type (
	NullProducerProvider struct{}
	NullProducer         struct{}
)

func NewNullProducerProvider() *NullProducerProvider {
	return &NullProducerProvider{}
}

func (p *NullProducerProvider) GetProducer(ctx context.Context, managerIdentifier, clientName string) (sandwich.Producer, error) {
	return &NullProducer{}, nil
}

func (p *NullProducer) Publish(ctx context.Context, shard *sandwich.Shard, payload sandwich.ProducedPayload) error {
	traceStr, _ := json.Marshal(payload.Trace)

	slog.Info("Publish", "type", payload.Type, "trace", string(traceStr))

	return nil
}

func (p *NullProducer) Close() error {
	return nil
}

func main() {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	stateProvider := sandwich.NewStateProviderMemoryOptimized()

	sandwich := sandwich.NewSandwich(
		// Replace this with whatever logger you want to use. It must be slog compatible.
		slog.Default(),

		// Replace this with whatever config provider you want to use. This can be a file, from database, etc.
		sandwich.NewConfigProviderFromPath("example_config.json"),

		// Replace this with whatever HTTP client you want to use. This can be a proxy or your own implementation.
		sandwich.NewProxyClient(*http.DefaultClient, url.URL{
			Scheme: "https",
			Host:   "discord.com",
		}),

		// If the builtin event handlers are not enough, you can implement your own.
		// This can be used to include more events.
		sandwich.NewEventProviderWithBlacklist(sandwich.NewBuiltinDispatchProvider(true)),

		// This handles the identify process for the shard. By default, it uses buckets in memory however
		// a connector for a URL is included or you can implement your own.
		sandwich.NewIdentifyViaBuckets(),

		// Replace this with whatever PUBSUB/custom implementation you want to use.
		NewNullProducerProvider(),

		// Replace this with whatever state provider you want to use. Sandwich includes a memory based
		// state provider however you can implement your own.
		stateProvider,
	)

	// TODO: GRPC, Prometheus, HTTP server configuration

	ctx, cancel := context.WithCancel(context.Background())

	err := sandwich.Start(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to start sandwich: %w", err))
	}

	// Wait for interrupt signal to gracefully shutdown

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	sandwich.Stop(ctx)

	cancel()
}
