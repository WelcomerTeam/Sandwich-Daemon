package discord

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	gotils_strconv "github.com/savsgio/gotils/strconv"
)

const (
	discordCreation = 1420070400000

	bitSize            = 64
	decimalBase        = 10
	maxInt64JsonLength = 22
)

var null = []byte("null")

// Placeholder type for easy identification.
type Snowflake int64

func (s *Snowflake) UnmarshalJSON(b []byte) error {
	if !bytes.Equal(b, null) {
		i, err := strconv.ParseInt(gotils_strconv.B2S(b[1:len(b)-1]), decimalBase, bitSize)
		if err != nil {
			return fmt.Errorf("failed to unmarshal json: %w", err)
		}

		*s = Snowflake(i)
	}

	return nil
}

func (s Snowflake) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, maxInt64JsonLength)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(s), decimalBase)
	buff = append(buff, '"')

	return buff, nil
}

func (s Snowflake) String() string {
	return strconv.FormatInt(int64(s), decimalBase)
}

// Time returns the creation time of the Snowflake.
func (s Snowflake) Time() time.Time {
	nsec := (int64(s) >> 22) + discordCreation

	return time.Unix(0, nsec*1000000)
}

// int64 to allow for marshalling support.
type Int64 int64

func (in *Int64) UnmarshalJSON(b []byte) error {
	if !bytes.Equal(b, null) {
		i, err := strconv.ParseInt(gotils_strconv.B2S(b[1:len(b)-1]), decimalBase, bitSize)
		if err != nil {
			return fmt.Errorf("failed to unmarshal json: %w", err)
		}

		*in = Int64(i)
	}

	return nil
}

func (in Int64) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, maxInt64JsonLength)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(in), decimalBase)
	buff = append(buff, '"')

	return buff, nil
}

func (in Int64) String() string {
	return strconv.FormatInt(int64(in), decimalBase)
}
