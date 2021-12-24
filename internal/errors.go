package internal

import (
	"golang.org/x/xerrors"
)

// ErrSessionLimitExhausted is returned when the sessions remaining
// is less than the ShardGroup is starting with.
var ErrSessionLimitExhausted = xerrors.New("The session limit has been reached")

// ErrInvalidToken is returned when an invalid token is used.
var ErrInvalidToken = xerrors.New("Token passed is not valid")

// ErrReconnect is used to distinguish if the shard simply wants to reconnect.
var ErrReconnect = xerrors.New("Reconnect is required")

var (
	ErrInvalidManager    = xerrors.New("No manager with this name exists")
	ErrInvalidShardGroup = xerrors.New("Invalid shard group id specified")
	ErrInvalidShard      = xerrors.New("Invalid shard id specified")
	ErrChunkTimeout      = xerrors.New("Timed out on initial member chunks")
	ErrMissingShards     = xerrors.New("ShardGroup has no shards")
)

var (
	ErrReadConfigurationFailure        = xerrors.New("Failed to read configuration")
	ErrLoadConfigurationFailure        = xerrors.New("Failed to load configuration")
	ErrConfigurationValidateIdentify   = xerrors.New("Configuration missing valid Identify URI")
	ErrConfigurationValidatePrometheus = xerrors.New("Configuration missing valid Prometheus Host")
	ErrConfigurationValidateGRPC       = xerrors.New("Configuration missing valid GRPC Host")
	ErrConfigurationValidateHTTP       = xerrors.New("Configuration missing valid HTTP Host")
)

var (
	ErrNoGatewayHandler  = xerrors.New("No registered handler for gateway event")
	ErrNoDispatchHandler = xerrors.New("No registered handler for dispatch event")
	ErrProducerMissing   = xerrors.New("No producer client found")
)
