package structs

type ShardGroupStatus uint8

const (
	ShardGroupStatusIdle ShardGroupStatus = iota
	ShardGroupStatusConnecting
	ShardGroupStatusConnected

	// Set on all non-closed ShardGroups when Manager scales.
	ShardGroupStatusMarkedForClosure

	ShardGroupStatusClosing
	ShardGroupStatusClosed
	ShardGroupStatusErroring
)

type ShardStatus uint8

const (
	ShardStatusIdle ShardStatus = iota
	ShardStatusConnecting
	ShardStatusConnected

	// Set when a Shard has received READY event and handled.
	ShardStatusReady
	ShardStatusReconnecting

	ShardStatusClosing
	ShardStatusClosed
	ShardStatusErroring
)

type ShardGroupStatusUpdate struct {
	Manager    string `json:"manager"`
	ShardGroup int64  `json:"shard_group"`
	Status     int    `json:"status"`
}

type ShardStatusUpdate struct {
	Manager    string `json:"manager"`
	ShardGroup int64  `json:"shard_group"`
	Shard      int    `json:"shard_id"`
	Status     int    `json:"status"`
}
