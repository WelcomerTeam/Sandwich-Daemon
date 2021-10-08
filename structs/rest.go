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
	Shards [][3]int         `json:"shards"`
	Status ShardGroupStatus `json:"status"`
}
