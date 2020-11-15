package structs

import (
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	jsoniter "github.com/json-iterator/go"
)

// BaseResponse is the response when returning REST requests and RPC calls
type BaseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RPCRequest is the structure the client sends when an RPC call is made
type RPCRequest struct {
	Method string              `json:"method"`
	Data   jsoniter.RawMessage `json:"data"`
}

// DataStamp stores time and its corresponding value
type DataStamp struct {
	Time  interface{} `json:"x"`
	Value interface{} `json:"y"`
}

// LineChart stores the data structure for a ChartJS LineChart
type LineChart struct {
	Labels   []string  `json:"labels,omitempty"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset is stores the representation of a Dataset in ChartJS
type Dataset struct {
	Label            string        `json:"label"`
	BackgroundColour string        `json:"backgroundColor,omitempty"`
	BorderColour     string        `json:"borderColor,omitempty"`
	Data             []interface{} `json:"data"`
}

// DiscordUser is the structure of a /users/@me request
type DiscordUser struct {
	ID            snowflake.ID `json:"id" msgpack:"id"`
	Username      string       `json:"username" msgpack:"username"`
	Discriminator string       `json:"discriminator" msgpack:"discriminator"`
	Avatar        string       `json:"avatar" msgpack:"avatar"`
	MFAEnabled    bool         `json:"mfa_enabled,omitempty" msgpack:"mfa_enabled,omitempty"`
	Locale        string       `json:"locale,omitempty" msgpack:"locale,omitempty"`
	Verified      bool         `json:"verified,omitempty" msgpack:"verified,omitempty"`
	Email         string       `json:"email,omitempty" msgpack:"email,omitempty"`
	Flags         int          `json:"flags" msgpack:"flags"`
	PremiumType   int          `json:"premium_type" msgpack:"premium_type"`
}

// APIMe is the response payload for a /api/me request
type APIMe struct {
	Authenticated bool         `json:"authenticated"`
	User          *DiscordUser `json:"user"`
}

// APIStatusResult is the main /api/status body where both the managers
// and its uptime is handled
type APIStatusResult struct {
	Managers []APIStatusManager `json:"managers"`
	Uptime   int64              `json:"uptime"`
}

// APIStatusManager is the structure of a manager
type APIStatusManager struct {
	DisplayName string                `json:"name"`
	Guilds      int64                 `json:"guilds"`
	ShardGroups []APIStatusShardGroup `json:"shard_groups"`
}

// APIStatusShardGroup is the structure of a shardgroup
type APIStatusShardGroup struct {
	ID     int32            `json:"id"`
	Status ShardGroupStatus `json:"status"`
	Shards []APIStatusShard `json:"shards"`
}

// APIStatusShard is the structure of a shard
type APIStatusShard struct {
	Status  ShardStatus `json:"status"`
	Latency int64       `json:"latency"`
	Uptime  int64       `json:"uptime"`
}

// APIAnalyticsResult is the structure of the /api/analytics request
type APIAnalyticsResult struct {
	Graph    LineChart            `json:"chart"`
	Guilds   int64                `json:"guilds"`
	Uptime   string               `json:"uptime"`
	Events   int64                `json:"events"`
	Managers []ManagerInformation `json:"managers"`
}

// ManagerInformation is the structure of the manager in the /api/analytics request
type ManagerInformation struct {
	Name      string                     `json:"name"`
	Guilds    int64                      `json:"guilds"`
	Status    map[int32]ShardGroupStatus `json:"status"`
	AutoStart bool                       `json:"autostart"`
}

// APIConfigurationResponse is the structure of the thread safe /api/configuration endpoint
type APIConfigurationResponse struct {
	Start             time.Time                                  `json:"uptime"`
	Configuration     interface{}                                `json:"configuration"`
	Managers          map[string]APIConfigurationResponseManager `json:"managers"`
	RestTunnelEnabled bool                                       `json:"rest_tunnel_enabled"`
}

// APIConfigurationResponseManager is the structure of the manager in the /api/configuration endpoint
type APIConfigurationResponseManager struct {
	ShardGroups   map[int32]APIConfigurationResponseShardGroup `json:"shard_groups"`
	Configuration interface{}                                  `json:"configuration"`
	Gateway       interface{}                                  `json:"gateway"`
	Error         string                                       `json:"error"`
}

// APIConfigurationResponseShardGroup is the structure of a shardgroup in the /api/configuration endpoint
type APIConfigurationResponseShardGroup struct {
	Status     ShardGroupStatus    `json:"status"`
	Error      string              `json:"error"`
	Start      time.Time           `json:"uptime"`
	WaitingFor int32               `json:"waiting_for"`
	ID         int32               `json:"id"`
	ShardCount int                 `json:"shard_count"`
	ShardIDs   []int               `json:"shard_ids"`
	Shards     map[int]interface{} `json:"shards"`
}

// APIConfigurationResponseShard is the structure of a shard in the /api/configuration endpoint
type APIConfigurationResponseShard struct {
	ShardID              int           `json:"shard_id"`
	Status               ShardStatus   `json:"status"`
	User                 *User         `json:"user"`
	LastHeartbeatAck     time.Time     `json:"last_heartbeat_ack"`
	LastHeartbeatSent    time.Time     `json:"last_heartbeat_sent"`
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`
	MaxHeartbeatFailures time.Duration `json:"max_heartbeat_failures"`
	Start                time.Time     `json:"start"`
	Retries              int32         `json:"retries"`
}
