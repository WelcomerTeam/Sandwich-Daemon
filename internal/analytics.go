package internal

import "github.com/prometheus/client_golang/prometheus"

var (
	sandwichEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_events_total",
			Help: "Sandwich Events",
		},
		[]string{"identifier"},
	)

	sandwichEventInflightCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_events_inflight_count",
			Help: "Count of dispatch events currently being processed",
		},
	)

	sandwichDiscardedEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_events_discarded_total",
			Help: "Count of discarded gateway events",
		},
		[]string{"identifier"},
	)

	sandwichGuildEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_events_by_guild_id_total",
			Help: "Sandwich Event Count by Guild",
		},
		[]string{"identifier", "guild_id"},
	)

	sandwichDispatchEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_dispatch_events_by_type_total",
			Help: "Sandwich Dispatch Events",
		},
		[]string{"identifier", "type"},
	)

	sandwichGatewayLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sandwich_discord_gateway_latency",
			Help: "Sandwich Discord Gateway Latency",
		},
		[]string{"identifier", "shard_group", "shard"},
	)

	// TODO: Implement.
	sandwichUnavailableGuildCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sandwich_unavailable_guilds_count",
			Help: "Sandwich Unavailable Guilds",
		},
		[]string{"identifier", "shard_group"},
	)

	sandwichStateTotalCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_count",
			Help: "Sandwich State Total Count",
		},
	)

	sandwichStateGuildCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_guild_count",
			Help: "Sandwich State Guild Count",
		},
	)

	sandwichStateGuildMembersCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_member_count",
			Help: "Sandwich State Guild Member Count",
		},
	)

	sandwichStateRoleCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_role_count",
			Help: "Sandwich State Guild Role Count",
		},
	)

	sandwichStateEmojiCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_emoji_count",
			Help: "Sandwich State Emoji Count",
		},
	)

	sandwichStateUserCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_user_count",
			Help: "Sandwich State User Count",
		},
	)

	sandwichStateChannelCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_channel_count",
			Help: "Sandwich State Channel Count",
		},
	)

	grpcCacheRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sandwich_grpc_requests_total",
			Help: "Sandwich GRPC Requests",
		},
	)

	grpcCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_grpc_cache_hits_total",
			Help: "Sandwich GRPC Cache Hits",
		},
		[]string{"guild_id"},
	)

	grpcCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_grpc_cache_misses_total",
			Help: "Sandwich GRPC Cache Misses",
		},
		[]string{"guild_id"},
	)

	// TODO: Message Tracing
	// Time between from discord GW and Produced
	// -   Time in state
	// Guild Count
	// Guild Join / Leave
	// Outbound WS count + Types.
)
