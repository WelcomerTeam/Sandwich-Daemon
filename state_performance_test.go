package sandwich

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"
	"github.com/coder/websocket"
)

func testPerformanceState(t testing.TB) *StateProviderMemoryOptimized {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewStateProviderMemoryOptimizedWithContext(ctx)
}

func TestAdjustMemberCountPreservesCollections(t *testing.T) {
	s := testPerformanceState(t)
	ctx := context.Background()
	s.SetGuild(ctx, 1, discord.Guild{ID: 1, MemberCount: 5})
	s.SetGuildChannel(ctx, 1, discord.Channel{ID: 10, Name: "newest"})
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				adjustGuildMemberCount(ctx, s, 1, 1)
			}
		})
	}
	wg.Wait()
	_, reported, ok := s.GetGuildMemberCounts(ctx, 1)
	if !ok || reported != 1605 {
		t.Fatalf("member count = %d, exists = %v", reported, ok)
	}
	channel, _ := s.GetGuildChannel(ctx, 1, 10)
	if channel == nil || channel.Name != "newest" {
		t.Fatal("member-count update modified guild channels")
	}
	s.AdjustGuildMemberCount(ctx, 999, 1)
	if _, _, ok := s.GetGuildMemberCounts(ctx, 999); ok {
		t.Fatal("member-count update created missing guild")
	}
}

func TestVoiceStateCountsAcrossTransitions(t *testing.T) {
	s := testPerformanceState(t)
	ctx := context.Background()
	a, z := discord.Snowflake(10), discord.Snowflake(20)
	s.SetGuildChannel(ctx, 1, discord.Channel{ID: a})
	s.SetGuildChannel(ctx, 1, discord.Channel{ID: z})
	check := func(wantA, wantZ int32) {
		t.Helper()
		ca, _ := s.GetGuildChannel(ctx, 1, a)
		cz, _ := s.GetGuildChannel(ctx, 1, z)
		if ca.MemberCount != wantA || cz.MemberCount != wantZ {
			t.Fatalf("counts = (%d, %d), want (%d, %d)", ca.MemberCount, cz.MemberCount, wantA, wantZ)
		}
	}
	s.SetVoiceState(ctx, 1, discord.VoiceState{UserID: 100, ChannelID: &a})
	s.SetVoiceState(ctx, 1, discord.VoiceState{UserID: 200, ChannelID: &a})
	check(2, 0)
	s.SetVoiceState(ctx, 1, discord.VoiceState{UserID: 100, ChannelID: &a, Mute: true})
	check(2, 0)
	s.SetVoiceState(ctx, 1, discord.VoiceState{UserID: 100, ChannelID: &z})
	check(1, 1)
	s.RemoveVoiceState(ctx, 1, 100)
	check(1, 0)
	s.RemoveVoiceState(ctx, 1, 200)
	check(0, 0)
	if s.VoiceStates.Count() != 0 {
		t.Fatal("empty voice-state map retained")
	}
}

type performanceDispatchProvider struct {
	extra *Extra
}

func (p performanceDispatchProvider) Dispatch(context.Context, *Shard, *discord.GatewayPayload, *Trace) (DispatchResult, bool, error) {
	return DispatchResult{Extra: p.extra}, true, nil
}

type performanceProducer struct{ err error }

func (p performanceProducer) Publish(context.Context, *Shard, *ProducedPayload) error { return p.err }
func (p performanceProducer) Close() error {
	return nil
}

func TestProducedPayloadReleasedOnEveryPublishExit(t *testing.T) {
	for _, publishErr := range []error{nil, errors.New("publish failed")} {
		packet := &ProducedPayload{}
		extra := Extra{"before": json.RawMessage(`{"id":"1"}`)}
		provider := NewEventProviderWithBlacklist(performanceDispatchProvider{extra: &extra})
		provider.producedPayloadPool = &sync.Pool{New: func() any { return packet }}
		application := &Application{
			Configuration: &atomic.Pointer[ApplicationConfiguration]{},
			producer:      performanceProducer{err: publishErr},
		}
		application.Configuration.Store(&ApplicationConfiguration{})
		shard := &Shard{Application: application, Metadata: &atomic.Pointer[ProducedMetadata]{}}
		shard.Metadata.Store(&ProducedMetadata{})
		data := json.RawMessage(`{"id":"1"}`)
		trace := Trace{"receive": int64(1)}
		err := provider.Dispatch(context.Background(), shard, &discord.GatewayPayload{Data: data}, &trace)
		if !errors.Is(err, publishErr) {
			t.Fatalf("error = %v, want %v", err, publishErr)
		}
		if !reflect.DeepEqual(*packet, ProducedPayload{}) {
			t.Fatal("pooled packet retained event references")
		}
		if string(data) != `{"id":"1"}` || string(extra["before"]) != `{"id":"1"}` || trace["receive"] != int64(1) {
			t.Fatal("pool cleanup mutated borrowed data")
		}
	}
}

func TestReleaseGatewayPayload(t *testing.T) {
	shard := &Shard{gatewayPayloadPool: &sync.Pool{}}
	data := json.RawMessage(`{"id":"1"}`)
	payload := &discord.GatewayPayload{Data: data, Type: "READY"}
	shard.releaseGatewayPayload(payload)
	if !reflect.DeepEqual(*payload, discord.GatewayPayload{}) {
		t.Fatal("gateway payload retained references or scalar fields")
	}
	if string(data) != `{"id":"1"}` {
		t.Fatal("gateway cleanup modified a borrowed buffer")
	}
}

func TestReadGatewayPayloadClearsOmittedFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"op":0,"t":"READY","s":123,"d":{"session_id":"test"}}`))
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"op":11}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer server.Close()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	shard := &Shard{gatewayPayloadPool: &sync.Pool{New: func() any { return &discord.GatewayPayload{} }}}
	first, err := shard.read(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	// Force reuse of the same object to make stale JSON fields deterministic.
	shard.gatewayPayloadPool = &sync.Pool{New: func() any { return first }}
	second, err := shard.read(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	if second.Type != "" || second.Sequence != 0 || second.Data != nil {
		t.Fatalf("ACK inherited dispatch fields: %+v", second)
	}
}

func TestCancelledChunkAttemptIsFinalized(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	shard := &Shard{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Sandwich: &Sandwich{
			guildChunks: syncmap.NewSyncMap[discord.Snowflake, *GuildChunk](),
		},
	}
	if err := shard.chunkGuild(ctx, 1, true); !errors.Is(err, context.Canceled) {
		t.Fatalf("chunk error = %v", err)
	}
	chunk, ok := shard.Sandwich.guildChunks.Load(1)
	if !ok || !chunk.complete.Load() || chunk.completedAt.Load() == nil {
		t.Fatal("cancelled attempt cannot be collected by chunk cleanup")
	}
}

func BenchmarkGuildMemberCounts(b *testing.B) {
	s := testPerformanceState(b)
	ctx := context.Background()
	s.SetGuild(ctx, 1, discord.Guild{ID: 1, MemberCount: 1000})
	for i := range 1000 {
		id := discord.Snowflake(i + 1)
		s.SetGuildMember(ctx, 1, discord.GuildMember{User: &discord.User{ID: id}})
	}
	b.Run("Materialized", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			members, _ := s.GetGuildMembers(ctx, 1)
			guild, _ := s.GetGuild(ctx, 1)
			if len(members) != int(guild.MemberCount) {
				b.Fatal("unexpected count")
			}
		}
	})
	b.Run("CountOnly", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cached, reported, _ := s.GetGuildMemberCounts(ctx, 1)
			if cached != reported {
				b.Fatal("unexpected count")
			}
		}
	})
}
