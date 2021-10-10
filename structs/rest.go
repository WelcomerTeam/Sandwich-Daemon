package structs

type StatusEndpointResponse struct {
	Managers []*StatusEndpointManager `json:"managers"`
}

type StatusEndpointManager struct {
	DisplayName string                      `json:"display_name"`
	ShardGroups []*StatusEndpointShardGroup `json:"shard_groups"`
}

type StatusEndpointShardGroup struct {
	// ShardID, Status, Latency (in milliseconds)
	ShardGroupID int              `json:"id"`
	Shards       [][5]int         `json:"shards"`
	Status       ShardGroupStatus `json:"status"`
}
