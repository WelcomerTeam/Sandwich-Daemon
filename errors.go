package sandwich

import "errors"

var (
	ErrApplicationMissingIdentifier = errors.New("application missing identifier")
	ErrApplicationMissingBotToken   = errors.New("application missing bot token")
	ErrApplicationIdentifierExists  = errors.New("application identifier already exists")

	ErrApplicationInitializeFailed = errors.New("application initialize failed")
	ErrApplicationMissingShards    = errors.New("application missing shards")

	ErrShardConnectFailed            = errors.New("shard connect failed")
	ErrShardInvalidHeartbeatInterval = errors.New("shard invalid heartbeat interval")
	ErrShardStopping                 = errors.New("shard stopping")

	ErrNoGatewayHandler  = errors.New("no gateway handler found")
	ErrNoDispatchHandler = errors.New("no dispatch handler found")
)
