package discord

import (
	"bytes"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	"golang.org/x/xerrors"
	"strconv"
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
	if !bytes.Equal(b, null) {
		i, err := strconv.ParseInt(gotils_strconv.B2S(b[1:len(b)-1]), decimalBase, bitSize)
		if err != nil {
			return xerrors.Errorf("Failed to unmarshal json: %v", err)
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

// int64 to allow for marshalling support.
type Int64 int64

func (in *Int64) UnmarshalJSON(b []byte) error {
	if !bytes.Equal(b, null) {
		i, err := strconv.ParseInt(gotils_strconv.B2S(b[1:len(b)-1]), decimalBase, bitSize)
		if err != nil {
			return xerrors.Errorf("Failed to unmarshal json: %v", err)
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

func (in *Int64) String() string {
	return strconv.FormatInt(int64(*in), decimalBase)
}
