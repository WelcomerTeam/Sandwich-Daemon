package sandwich

import "errors"

var (
	ErrManagerMissingIdentifier = errors.New("manager missing identifier")
	ErrManagerMissingBotToken   = errors.New("manager missing bot token")
	ErrManagerIdentifierExists  = errors.New("manager identifier already exists")

	ErrManagerInitializeFailed = errors.New("manager initialize failed")
	ErrManagerMissingShards    = errors.New("manager missing shards")

	ErrShardConnectFailed            = errors.New("shard connect failed")
	ErrShardInvalidHeartbeatInterval = errors.New("shard invalid heartbeat interval")

	ErrNoGatewayHandler  = errors.New("no gateway handler found")
	ErrNoDispatchHandler = errors.New("no dispatch handler found")
)
