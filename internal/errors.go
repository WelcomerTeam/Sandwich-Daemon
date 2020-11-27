package gateway

import "errors"

// ErrSessionLimitExhausted is returned when the sessions remaining
// is less than the ShardGroup is starting with.
var ErrSessionLimitExhausted = errors.New("the session limit has been reached")

// ErrInvalidToken is returned when an invalid token is used.
var ErrInvalidToken = errors.New("token passed is not valid")

// ErrReconnect is used to distinguish if the shard simply wants to reconnect.
var ErrReconnect = errors.New("reconnect is required")
