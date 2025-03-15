package internal

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReturnRangeInt32(t *testing.T) {
	rangeString := "0-4,6-7"
	max := int32(8)
	expected := []int32{0, 1, 2, 3, 4, 6, 7}

	result := returnRangeInt32(rangeString, max)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestReturnRangeInt32Single(t *testing.T) {
	rangeString := "0"
	max := int32(8)
	expected := []int32{0}

	result := returnRangeInt32(rangeString, max)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestReturnRangeInt32Empty(t *testing.T) {
	rangeString := ""
	max := int32(8)
	expected := []int32{}

	result := returnRangeInt32(rangeString, max)

	if len(result) != 0 {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestReturnRangeInt32Invalid(t *testing.T) {
	rangeString := "0-4,6-7,8"
	max := int32(8)
	expected := []int32{0, 1, 2, 3, 4, 6, 7}

	result := returnRangeInt32(rangeString, max)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestReplaceIfEmpty(t *testing.T) {
	v := replaceIfEmpty("", "default")
	expected := "default"

	if v != expected {
		t.Errorf("Expected %q, but got %q", expected, v)
	}

	v = replaceIfEmpty("value", "default")
	expected = "value"

	if v != expected {
		t.Errorf("Expected %q, but got %q", expected, v)
	}
}

func TestRandomHex(t *testing.T) {
	length := 16
	result := randomHex(length)
	if len(result) != length*2 {
		t.Errorf("Expected length %d, but got %d", length*2, len(result))
	}
}

func TestRandomHexZeroLength(t *testing.T) {
	length := 0
	result := randomHex(length)
	if len(result) != length*2 {
		t.Errorf("Expected length %d, but got %d", length*2, len(result))
	}
}

func TestRandomHexNegativeLength(t *testing.T) {
	length := -10
	result := randomHex(length)

	if len(result) != 0 {
		t.Errorf("Expected length 0, but got %d", len(result))
	}
}

func TestMakeExtra(t *testing.T) {
	extra := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	expected := map[string]json.RawMessage{
		"key1": []byte(`"value1"`),
		"key2": []byte(`123`),
		"key3": []byte(`true`),
	}

	result, err := makeExtra(extra)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestMakeExtraEmpty(t *testing.T) {
	extra := map[string]interface{}{}

	expected := map[string]json.RawMessage{}

	result, err := makeExtra(extra)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestMakeExtraError(t *testing.T) {
	extra := map[string]interface{}{
		"key1": "Hello world",
	}

	out, err := makeExtra(extra)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if out == nil {
		t.Errorf("Expected out, but got nil")
	}

	expected := json.RawMessage("\"Hello world\"")
	if !reflect.DeepEqual(out["key1"], expected) {
		t.Errorf("Expected %v, but got %v", string(expected), string(out["key1"]))
	}
}
