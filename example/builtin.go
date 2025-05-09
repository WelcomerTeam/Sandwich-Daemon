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
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich "github.com/WelcomerTeam/Sandwich-Daemon"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"

	_ "net/http/pprof"
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

func (p *NullProducerProvider) GetProducer(ctx context.Context, managerIdentifier, clientName string) (sandwich.Producer, error) {
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

func (p *NullProducer) Publish(ctx context.Context, shard *sandwich.Shard, payload sandwich.ProducedPayload) error {
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

	go func() {
		for {
			time.Sleep(time.Second * 10)

			guildChannelBuckets := 0
			guildChannelsCount := 0
			stateProvider.GuildChannels.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, sandwich.StateChannel]) (stop bool) {
				guildChannelBuckets++
				guildChannelsCount += value.Count()
				return false
			})

			guildEmojisBuckets := 0
			guildEmojisCount := 0
			stateProvider.GuildEmojis.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, sandwich.StateEmoji]) (stop bool) {
				guildEmojisBuckets++
				guildEmojisCount += value.Count()
				return false
			})

			guildMembersBuckets := 0
			guildMembersCount := 0
			stateProvider.GuildMembers.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, sandwich.StateGuildMember]) (stop bool) {
				guildMembersBuckets++
				guildMembersCount += value.Count()
				return false
			})

			guildsCount := stateProvider.Guilds.Count()

			userMutualsBuckets := 0
			userMutualsCount := 0
			stateProvider.UserMutuals.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, bool]) (stop bool) {
				userMutualsBuckets++
				userMutualsCount += value.Count()
				return false
			})

			usersCount := 0
			stateProvider.Users.Range(func(_ discord.Snowflake, value discord.User) (stop bool) {
				usersCount++
				return false
			})

			voiceStatesBuckets := 0
			voiceStatesCount := 0
			stateProvider.VoiceStates.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, sandwich.StateVoiceState]) (stop bool) {
				voiceStatesBuckets++
				voiceStatesCount += value.Count()
				return false
			})

			slog.Info("================")
			slog.Info("Guild channels", "buckets", guildChannelBuckets, "count", guildChannelsCount)
			slog.Info("Guilds", "count", guildsCount)
			slog.Info("Guild emojis", "buckets", guildEmojisBuckets, "count", guildEmojisCount)
			slog.Info("Guild members", "buckets", guildMembersBuckets, "count", guildMembersCount)
			slog.Info("User mutuals", "buckets", userMutualsBuckets, "count", userMutualsCount)
			slog.Info("Users", "count", usersCount)
			slog.Info("Voice states", "buckets", voiceStatesBuckets, "count", voiceStatesCount)
			slog.Info("================")
		}
	}()

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
	})

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
