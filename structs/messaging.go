package structs

// PublishEvent is the event sent to consumers
type PublishEvent struct {
	Data interface{} `msgpack:"d,omitempty"`
	From string      `msgpack:"f,omitempty"` // This is the manager identifier
	Type string      `msgpack:"t,omitempty"`
}

// MessagingStatusUpdate represents when a shard/shardgroup has a new state
type MessagingStatusUpdate struct {
	ShardID int   `msgpack:"shard,omitempty"`
	Status  int32 `msgpack:"status"`
}
