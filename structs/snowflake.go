package structs

import "github.com/TheRockettek/snowflake"

// We change the default Epoch of the snowflake to match discord's.
func init() { //nolint:gochecknoinits
	snowflake.Epoch = 1420070400000
}
