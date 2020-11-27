package structs

// RPCManagerShardGroupCreateEvent is the data structure of a RPCManagerShardGroupCreate request.
type RPCManagerShardGroupCreateEvent struct {
	Manager          string `json:"manager"`
	RawShardIDs      string `json:"shardIDs"`
	ShardCount       int    `json:"shardCount"`
	ShardIDs         []int  `json:"finalShardIDs"`
	AutoIDs          bool   `json:"autoIDs"`
	AutoShard        bool   `json:"autoShard"`
	StartImmediately bool   `json:"startImmediately"`
}

// RPCManagerShardGroupStopEvent is the data structure of a RPCManagerShardGroupStop request.
type RPCManagerShardGroupStopEvent struct {
	Manager    string `json:"manager"`
	ShardGroup int32  `json:"shardgroup"`
}

// RPCManagerShardGroupDeleteEvent is the data structure of a RPCManagerShardGroupDelete request.
type RPCManagerShardGroupDeleteEvent struct {
	Manager    string `json:"manager"`
	ShardGroup int32  `json:"shardgroup"`
}

// RPCManagerCreateEvent is the data structure of a RPCManagerCreate request.
type RPCManagerCreateEvent struct {
	Persist    bool   `json:"persist"`
	Identifier string `json:"identifier"`

	Token   string `json:"token"`
	Prefix  string `json:"prefix"`
	Client  string `json:"client"`
	Channel string `json:"channel"`
}

// RPCManagerDeleteEvent is the data structure of a RPCManagerDelete request.
type RPCManagerDeleteEvent struct {
	Manager string `json:"manager"`
	Confirm string `json:"confirm"`
}

// RPCManagerRestartEvent is the data structure of a RPCManagerRestart request.
type RPCManagerRestartEvent struct {
	Manager string `json:"manager"`
	Confirm string `json:"confirm"`
}

// RPCManagerRefreshGatewayEvent is the data structure of a RPCManagerRefreshGateway request.
type RPCManagerRefreshGatewayEvent struct {
	Manager string `json:"manager"`
}
