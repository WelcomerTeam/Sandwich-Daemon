package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"reflect"
	"strconv"
	"time"
)

func contains(a interface{}, vars ...interface{}) bool {
	for _var := range vars {
		if _var == a {
			return true
		}
	}
	return false
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

// QuickHash simply returns hash from input
func QuickHash(hash string) string {
	h := sha256.New()
	h.Write([]byte(hash))
	return hex.EncodeToString(h.Sum(nil))
}

// DurationTimestamp outputs in a format similar to the timestamp String()
func DurationTimestamp(d time.Duration) (output string) {
	seconds := d.Seconds()
	if seconds > 86400 {
		days := math.Trunc(seconds / 86400)
		if days > 0 {
			output += strconv.Itoa(int(days)) + "d"
		}
		seconds = math.Mod(seconds, 86400)
	}
	if seconds > 3600 {
		hours := math.Trunc(seconds / 3600)
		if hours > 0 {
			output += strconv.Itoa(int(hours)) + "h"
		}
		seconds = math.Mod(seconds, 3600)
	}
	minutes := math.Trunc(seconds / 60)
	if minutes > 0 {
		output += strconv.Itoa(int(minutes)) + "m"
	}
	seconds = math.Mod(seconds, 60)
	if seconds > 0 {
		output += strconv.Itoa(int(seconds)) + "s"
	}
	return
}
