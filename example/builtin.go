package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"time"

	sandwich "github.com/WelcomerTeam/Sandwich-Daemon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Replace this with whatever PUBSUB/implementation you want to use.

type (
	NullProducerProvider struct{}
	NullProducer         struct {
		counter *atomic.Int64
	}
)

func NewNullProducerProvider() *NullProducerProvider {
	return &NullProducerProvider{}
}

func (p *NullProducerProvider) GetProducer(_ context.Context, applicationIdentifier, clientName string) (sandwich.Producer, error) {
	producer := &NullProducer{
		counter: &atomic.Int64{},
	}

	go func() {
		for {
			time.Sleep(time.Second * 1)
			slog.Info("Events/s", "events", producer.counter.Load())
			producer.counter.Store(0)
		}
	}()

	return producer, nil
}

func (p *NullProducer) Publish(_ context.Context, shard *sandwich.Shard, payload *sandwich.ProducedPayload) error {
	traceStr, _ := json.Marshal(payload.Trace)

	p.counter.Add(1)
	slog.Debug("Publish", "type", payload.Type, "trace", string(traceStr))

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
		sandwich.NewConfigProviderFromPath("config.json.local"),

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
	).WithPanicHandler(func(_ *sandwich.Sandwich, r any) {
		slog.Error("Panic occurred", "error", r)

		// Write stack trace to file and print it
		stackTrace := debug.Stack()
		println(string(stackTrace))

		// Write to file with timestamp
		filename := fmt.Sprintf("logs/panic_%s.log", time.Now().Format("2006-01-02_15-04-05"))

		// Ensure logs directory exists
		if err := os.MkdirAll("logs", 0o755); err != nil {
			slog.Error("Failed to create logs directory", "error", err)
		}

		if err := os.WriteFile(filename, stackTrace, 0o600); err != nil {
			slog.Error("Failed to write stack trace to file", "error", err)
		}
	}).WithPrometheusAnalytics(
		&http.Server{
			Addr:              ":10000",
			WriteTimeout:      time.Second * 10,
			ReadTimeout:       time.Second * 10,
			ReadHeaderTimeout: time.Second * 10,
			IdleTimeout:       time.Second * 10,
			ErrorLog:          slog.NewLogLogger(slog.With("service", "prometheus").Handler(), slog.LevelError),
		},
		prometheus.NewPedanticRegistry(),
		promhttp.HandlerOpts{},
	)

	// TODO: GRPC, HTTP server configuration

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
