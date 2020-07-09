package structs

// ShardStatus represents the shard status
type ShardStatus int32

// Status Codes for Shard
const (
	ShardIdle         ShardStatus = iota // Represents a Shard that has been created but not opened yet
	ShardWaiting                         // Represents a Shard waiting for the identify ratelimit
	ShardConnecting                      // Represents a Shard connecting to the gateway
	ShardConnected                       // Represents a Shard that has connected to discords gateway
	ShardReady                           // Represents a Shard that has finished lazy loading
	ShardReconnecting                    // Represents a Shard that is reconnecting
	ShardClosed                          // Represents a Shard that has been closed
)

// ShardGroupStatus represents a shardgroups status
type ShardGroupStatus int32

// Status Codes for ShardGroups
const (
	ShardGroupIdle       ShardGroupStatus = iota // Represents a ShardGroup that has been created but not opened yet
	ShardGroupStarting                           // Represent a ShardGroup that is still starting up clients
	ShardGroupConnecting                         // Represents a ShardGroup waiting for shard to be ready
	ShardGroupReady                              // Represent a ShardGroup that has all its shards ready
	ShardGroupReplaced                           // Represent a ShardGroup that is going to be replaced soon by a new ShardGroup
	ShardGroupClosing                            // Represent a ShardGroup in the process of closing
	ShardGroupClosed                             // Represent a closed ShardGroup
)
