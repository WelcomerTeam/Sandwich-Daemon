package internal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strconv"
	"strings"
)

type void struct{}

func replaceIfEmpty(v string, s string) string {
	if v == "" {
		return s
	}

	return v
}

// quickHash returns hash from method and input.
func quickHash(hashMethod hash.Hash, text string) (string, error) {
	hashMethod.Reset()

	if _, err := hashMethod.Write([]byte(text)); err != nil {
		return "", fmt.Errorf("failed to hash text: %w", err)
	}

	return hex.EncodeToString(hashMethod.Sum(nil)), nil
}

// returnRangeInt32 converts a string like 0-4,6-7 to [0,1,2,3,4,6,7].
func returnRangeInt32(rangeString string, max int32) (result []int32) {
	splits := strings.Split(rangeString, ",")
	if len(splits) == 0 {
		splits = append(splits, rangeString)
	}

	for _, split := range splits {
		ranges := strings.Split(split, "-")

		if len(ranges) == 0 {
			if i, err := strconv.Atoi(split); err == nil {
				if 0 <= i && int32(i) < max {
					result = append(result, int32(i))
				}
			}
		} else {
			if low, err := strconv.Atoi(ranges[0]); err == nil {
				if hi, err := strconv.Atoi(ranges[len(ranges)-1]); err == nil {
					for i := int32(low); i < int32(hi+1); i++ {
						if 0 <= i && i < max {
							result = append(result, i)
						}
					}
				}
			}
		}
	}

	return result
}

func randomHex(length int) (result string) {
	if length <= 0 {
		return ""
	}

	buf := make([]byte, length)

	_, err := rand.Read(buf)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(buf)
}

// makeExtra converts from interfaces to RawMessages. Used for extra data in payloads.
func makeExtra(extra map[string]interface{}) (out map[string]json.RawMessage, err error) {
	out = make(map[string]json.RawMessage)

	for k, v := range extra {
		p, err := json.Marshal(v)
		if err != nil {
			return out, fmt.Errorf("failed to marshal extra: %w", err)
		}

		out[k] = p
	}

	return
}
