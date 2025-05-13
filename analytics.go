package sandwich

import (
	"strconv"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EventMetrics tracks event-related metrics
var EventMetrics = struct {
	EventsTotal    *prometheus.CounterVec
	GatewayLatency *prometheus.GaugeVec
}{
	EventsTotal: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sandwich_events_total",
			Help: "Total number of events processed, split by identifier and event type",
		},
		[]string{"application_identifier", "event_type", "guild_id"},
	),
	GatewayLatency: promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sandwich_gateway_latency_seconds",
			Help: "Gateway latency in seconds, measured by heartbeat",
		},
		[]string{"application_identifier"},
	),
}

func RecordEvent(identifier, eventType string) {
	EventMetrics.EventsTotal.WithLabelValues(identifier, eventType, "").Inc()
}

func RecordEventWithGuildID(identifier, eventType string, guildID discord.Snowflake) {
	EventMetrics.EventsTotal.WithLabelValues(identifier, eventType, guildID.String()).Inc()
}

func UpdateGatewayLatency(identifier string, latency float64) {
	EventMetrics.GatewayLatency.WithLabelValues(identifier).Set(latency)
}

// ShardMetrics tracks shard-related metrics
var ShardMetrics = struct {
	ApplicationStatus *prometheus.GaugeVec
	ShardStatus       *prometheus.GaugeVec
}{
	ApplicationStatus: promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sandwich_shard_application_status",
			Help: "Status of the shard application",
		},
		[]string{"application_identifier"},
	),
	ShardStatus: promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sandwich_shard_status",
			Help: "Status of the shard",
		},
		[]string{"application_identifier", "shard_id"},
	),
}

func UpdateApplicationStatus(identifier string, status ApplicationStatus) {
	ShardMetrics.ApplicationStatus.WithLabelValues(identifier).Set(float64(status))
}

func UpdateShardStatus(identifier string, shardID int32, status ShardStatus) {
	ShardMetrics.ShardStatus.WithLabelValues(identifier, strconv.Itoa(int(shardID))).Set(float64(status))
}

// StateMetrics tracks state-related metrics
var StateMetrics = struct {
	GuildMembers      prometheus.Gauge
	GuildRoles        prometheus.Gauge
	Emojis            prometheus.Gauge
	Users             prometheus.Gauge
	Channels          prometheus.Gauge
	Stickers          prometheus.Gauge
	Guilds            prometheus.Gauge
	VoiceStates       prometheus.Gauge
	UnavailableGuilds prometheus.Gauge
}{
	GuildMembers: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_guild_members",
			Help: "Total number of guild members in state",
		},
	),
	GuildRoles: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_guild_roles",
			Help: "Total number of guild roles in state",
		},
	),
	Emojis: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_emojis",
			Help: "Total number of emojis in state",
		},
	),
	Users: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_users",
			Help: "Total number of users in state",
		},
	),
	Channels: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_channels",
			Help: "Total number of channels in state",
		},
	),
	Stickers: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_stickers",
			Help: "Total number of stickers in state",
		},
	),
	Guilds: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_guilds",
			Help: "Total number of guilds in state",
		},
	),
	VoiceStates: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_state_voice_states",
			Help: "Total number of voice states in state",
		},
	),
	UnavailableGuilds: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sandwich_unavailable_guilds",
			Help: "Number of currently unavailable guilds",
		},
	),
}

func UpdateUnavailableGuilds(count float64) {
	StateMetrics.UnavailableGuilds.Set(count)
}

func UpdateStateMetrics(members, roles, emojis, users, channels, stickers, guilds, voiceStates int) {
	StateMetrics.GuildMembers.Set(float64(members))
	StateMetrics.GuildRoles.Set(float64(roles))
	StateMetrics.Emojis.Set(float64(emojis))
	StateMetrics.Users.Set(float64(users))
	StateMetrics.Channels.Set(float64(channels))
	StateMetrics.Stickers.Set(float64(stickers))
	StateMetrics.Guilds.Set(float64(guilds))
	StateMetrics.VoiceStates.Set(float64(voiceStates))
}
