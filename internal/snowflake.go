package gateway

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// We change the default Epoch of the snowflake to match discord's.
func init() { //nolint:gochecknoinits
	snowflake.Epoch = 1420070400000
}
