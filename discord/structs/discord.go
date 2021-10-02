package discord

import (
	"bytes"
	"strconv"

	gotils_strconv "github.com/savsgio/gotils/strconv"
)

const (
	bitSize            = 64
	decimalBase        = 10
	maxInt64JsonLength = 22
)

var null = []byte{'n', 'u', 'l', 'l'}

// Placeholder type for easy identification.
type Snowflake int64

func (s *Snowflake) UnmarshalJSON(b []byte) error {
	if bytes.Compare(b, null) != 0 {
		i, err := strconv.ParseInt(gotils_strconv.B2S(b[1:len(b)-1]), decimalBase, bitSize)
		if err != nil {
			return err
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

func (s *Snowflake) String() string {
	return strconv.FormatInt(int64(*s), decimalBase)
}

// JSON-Marshal compatable Int64.
type jInt64 int64

func (s *jInt64) UnmarshalJSON(b []byte) error {
	i, err := strconv.ParseInt(gotils_strconv.B2S(b[1:len(b)-1]), decimalBase, bitSize)
	if err != nil {
		return err
	}

	*s = jInt64(i)

	return nil
}

func (s jInt64) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, maxInt64JsonLength)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(s), decimalBase)
	buff = append(buff, '"')

	return buff, nil
}
