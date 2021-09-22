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

	// TODO: Implement.
	sandwichGuildEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_events_by_guild_id_total",
			Help: "Sandwich Event Count by Guild",
		},
		[]string{"identifier", "guild_id"},
	)

	// TODO: Implement.
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

	// TODO: Implement.
	sandwichStateTotalCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_count",
			Help: "Sandwich State Total Count",
		},
	)

	// TODO: Implement.
	sandwichStateGuildCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_guild_count",
			Help: "Sandwich State Guild Count",
		},
	)

	// TODO: Implement.
	sandwichStateGuildMembersCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_member_count",
			Help: "Sandwich State Guild Member Count",
		},
	)

	// TODO: Implement.
	sandwichStateRoleCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_role_count",
			Help: "Sandwich State Guild Role Count",
		},
	)

	// TODO: Implement.
	sandwichStateEmojiCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_emoji_count",
			Help: "Sandwich State Emoji Count",
		},
	)

	// TODO: Implement.
	sandwichStateUserCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_user_count",
			Help: "Sandwich State User Count",
		},
	)

	// TODO: Implement.
	sandwichStateChannelCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_channel_count",
			Help: "Sandwich State Channel Count",
		},
	)

	// TODO: Implement.
	grpcCacheRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sandwich_grpc_requests_total",
			Help: "Sandwich GRPC Requests",
		},
	)

	// TODO: Implement.
	grpcCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_grpc_cache_hits_total",
			Help: "Sandwich GRPC Cache Hits",
		},
		[]string{"identifier"},
	)

	// TODO: Implement.
	grpcCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_grpc_cache_misses_total",
			Help: "Sandwich GRPC Cache Misses",
		},
		[]string{"identifier"},
	)

	// TODO: Message Tracing
	// Time between from discord GW and Produced
	// -   Time in state
	// Events waiting for ticket
	// Guild Count
	// Guild Join / Leave
	// Outbound WS count + Types.
)
