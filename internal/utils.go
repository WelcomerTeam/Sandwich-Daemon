package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
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
