package internal

import "github.com/prometheus/client_golang/prometheus"

var (
	sandwichEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help: "Sandwich Events",
		},
		[]string{"identifier"},
	)

	sandwichGuildEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help: "Sandwich Event Count by Guild",
		},
		[]string{"identifier", "guild_id"},
	)

	sandwichDispatchEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help: "Sandwich Dispatch Events",
		},
		[]string{"identifier", "type"},
	)

	sandwichGatewayLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help: "Sandwich Discord Gateway Latency",
		},
		[]string{"identifier", "shard_group", "shard"},
	)

	sandwichUnavailableGuildCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help: "Sandwich Unavailable Guilds",
		},
		[]string{"identifier", "shard_group"},
	)

	sandwichStateTotalCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State Total Count",
		},
	)

	sandwichStateGuildCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State Guild Count",
		},
	)

	sandwichStateGuildMembersCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State Guild Member Count",
		},
	)

	sandwichStateRoleCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State Guild Role Count",
		},
	)

	sandwichStateEmojiCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State Emoji Count",
		},
	)

	sandwichStateUserCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State User Count",
		},
	)

	sandwichStateChannel = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich State Channel Count",
		},
	)

	grpcCacheRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich GRPC Requests",
		},
	)

	grpcCacheHits = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich GRPC Cache Hits",
		},
	)

	grpcCacheMisses = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "Sandwich GRPC Cache Misses",
		},
	)

	// TODO: Message Tracing
	// Time between from discord GW and Produced
	// Time in state

	// Events waiting for ticket

	// Guild Count
	// Guild Join / Leave

	// Outbound WS count + Types
)
