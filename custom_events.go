package sandwich

const (
	SandwichEventConfigUpdate       = "SW_CONFIGURATION_RELOAD"
	SandwichShardStatusUpdate       = "SW_SHARD_STATUS_UPDATE"
	SandwichApplicationStatusUpdate = "SW_APPLICATION_STATUS_UPDATE"
)

type ShardStatusUpdateEvent struct {
	Identifier string      `json:"identifier"`
	ShardID    int32       `json:"shard_id"`
	Status     ShardStatus `json:"status"`
}

type ApplicationStatusUpdateEvent struct {
	Identifier string            `json:"identifier"`
	Status     ApplicationStatus `json:"status"`
}
