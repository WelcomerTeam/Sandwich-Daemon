package structs

// ShardStatus represents the shard status.
type ShardStatus int32

// Status Codes for Shard.
const (
	ShardIdle         ShardStatus = iota // Represents a Shard that has been created but not opened yet
	ShardWaiting                         // Represents a Shard waiting for the identify ratelimit
	ShardConnecting                      // Represents a Shard connecting to the gateway
	ShardConnected                       // Represents a Shard that has connected to discords gateway
	ShardReady                           // Represents a Shard that has finished lazy loading
	ShardReconnecting                    // Represents a Shard that is reconnecting
	ShardClosed                          // Represents a Shard that has been closed
)

func (ss *ShardStatus) String() string {
	switch *ss {
	case ShardIdle:
		return "Idle"
	case ShardWaiting:
		return "Waiting"
	case ShardConnecting:
		return "Connecting"
	case ShardConnected:
		return "Connected"
	case ShardReady:
		return "Ready"
	case ShardReconnecting:
		return "Reconnecting"
	case ShardClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

func (ss *ShardStatus) Colour() int {
	switch *ss {
	case ShardIdle:
		return 0
	case ShardWaiting:
		return 1548214
	case ShardConnecting:
		return 1548214
	case ShardConnected:
		return 1548214
	case ShardReady:
		return 2664005
	case ShardReconnecting:
		return 16760839
	case ShardClosed:
		return 0
	default:
		return 16701571
	}
}

// ShardGroupStatus represents a shardgroups status.
type ShardGroupStatus int32

// Status Codes for ShardGroups.
const (
	ShardGroupIdle       ShardGroupStatus = iota // Represents a ShardGroup that has been created but not opened yet
	ShardGroupStarting                           // Represent a ShardGroup that is still starting up clients (connecting to gateway)
	ShardGroupConnecting                         // Represents a ShardGroup that has successfully connected to gateway and is waiting for chunking to finish
	ShardGroupReady                              // Represent a ShardGroup that has all its shards ready
	ShardGroupReplaced                           // Represent a ShardGroup that is going to be replaced soon by a new ShardGroup
	ShardGroupClosing                            // Represent a ShardGroup in the process of closing
	ShardGroupClosed                             // Represent a closed ShardGroup
	ShardGroupError                              // Represents a closed ShardGroup that closed unexpectedly due to an error
)
