package internal

import "errors"

// ErrSessionLimitExhausted is returned when the sessions remaining
// is less than the ShardGroup is starting with.
var ErrSessionLimitExhausted = errors.New("the session limit has been reached")

// ErrInvalidToken is returned when an invalid token is used.
var ErrInvalidToken = errors.New("token passed is not valid")

// ErrReconnect is used to distinguish if the shard simply wants to reconnect.
var ErrReconnect = errors.New("reconnect is required")

var (
	ErrInvalidManager    = errors.New("no manager with this name exists")
	ErrInvalidShardGroup = errors.New("invalid shard group id specified")
	ErrInvalidShard      = errors.New("invalid shard id specified")
	ErrChunkTimeout      = errors.New("timed out on initial member chunks")
	ErrMissingShards     = errors.New("shardGroup has no shards")
)

var (
	ErrReadConfigurationFailure        = errors.New("failed to read configuration")
	ErrLoadConfigurationFailure        = errors.New("failed to load configuration")
	ErrConfigurationValidateIdentify   = errors.New("configuration missing valid Identify URI")
	ErrConfigurationValidatePrometheus = errors.New("configuration missing valid Prometheus Host")
	ErrConfigurationValidateGRPC       = errors.New("configuration missing valid GRPC Host")
	ErrConfigurationValidateHTTP       = errors.New("configuration missing valid HTTP Host")
)

var (
	ErrNoGatewayHandler  = errors.New("no registered handler for gateway event")
	ErrNoDispatchHandler = errors.New("no registered handler for dispatch event")
	ErrProducerMissing   = errors.New("no producer client found")
)
