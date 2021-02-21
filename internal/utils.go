package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"golang.org/x/xerrors"
)

const (
	daySeconds            = 86400
	hourSeconds           = 3600
	minuteSeconds         = 60
	discordSnowflakeEpoch = 1420070400000
)

// We change the default Epoch of the snowflake to match discord's.
func init() { //nolint:gochecknoinits
	snowflake.Epoch = discordSnowflakeEpoch
}

type void struct{}

func replaceIfEmpty(v string, s string) string {
	if v == "" {
		return s
	}

	return v
}

// Returns the error.Error() if not null else empty.
func ReturnError(err error) string {
	if err != nil {
		return err.Error()
	}

	return ""
}

// DeepEqualExports compares exported values of two interfaces based on the
// tagName provided.
func DeepEqualExports(tagName string, a interface{}, b interface{}) bool {
	if tagName == "" {
		tagName = "msgpack"
	}

	elem := reflect.TypeOf(a).Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		tagValue, ok := field.Tag.Lookup(tagName)
		// We really should be checking the tagValues for both but this is
		// is a safe bet assuming both interfaces are the same type, which
		// it is intended for.
		if ok && tagValue != "-" && tagValue != "" {
			val1 := reflect.Indirect(reflect.ValueOf(a)).FieldByName(field.Name)
			val2 := reflect.Indirect(reflect.ValueOf(b)).FieldByName(field.Name)

			if !reflect.DeepEqual(val1.Interface(), val2.Interface()) {
				return false
			}
		}
	}

	return true
}

// QuickHash simply returns hash from input.
func QuickHash(hash string) (result string, err error) {
	h := sha256.New()

	if _, err := h.Write([]byte(hash)); err != nil {
		return "", xerrors.Errorf("Failed to write to sha256 writer: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// DurationTimestamp outputs in a format similar to the timestamp String().
func DurationTimestamp(d time.Duration) (output string) {
	seconds := d.Seconds()
	if seconds > daySeconds {
		days := math.Trunc(seconds / daySeconds)
		if days > 0 {
			output += strconv.Itoa(int(days)) + "d"
		}

		seconds = math.Mod(seconds, daySeconds)
	}

	if seconds > hourSeconds {
		hours := math.Trunc(seconds / hourSeconds)
		if hours > 0 {
			output += strconv.Itoa(int(hours)) + "h"
		}

		seconds = math.Mod(seconds, hourSeconds)
	}

	minutes := math.Trunc(seconds / minuteSeconds)
	if minutes > 0 {
		output += strconv.Itoa(int(minutes)) + "m"
	}

	seconds = math.Mod(seconds, minuteSeconds)
	if seconds > 0 {
		output += strconv.Itoa(int(seconds)) + "s"
	}

	return output
}

// ReturnRange converts a string like 0-4,6-7 to [0,1,2,3,4,6,7].
func ReturnRange(_range string, max int) (result []int) {
	for _, split := range strings.Split(_range, ",") {
		ranges := strings.Split(split, "-")
		if low, err := strconv.Atoi(ranges[0]); err == nil {
			if hi, err := strconv.Atoi(ranges[len(ranges)-1]); err == nil {
				for i := low; i < hi+1; i++ {
					if 0 <= i && i < max {
						result = append(result, i)
					}
				}
			}
		}
	}

	return result
}

// WebhookTime returns a formatted time.Time as a time accepted by webhooks.
func WebhookTime(_time time.Time) string {
	return _time.Format("2006-01-02T15:04:05Z")
}
