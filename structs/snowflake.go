package structs

import "github.com/bwmarrin/snowflake"

// We change the default Epoch of the snowflake to match discord's
func init() {
	snowflake.Epoch = 1420070400000
}
