package internal

import (
	"encoding/hex"
	"fmt"
	"hash"
	"math/rand"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type void struct{}

func replaceIfEmpty(v string, s string) string {
	if v == "" {
		return s
	}

	return v
}

// quickHash returns hash from method and input.
func quickHash(hashMethod hash.Hash, text string) (result string, err error) {
	hashMethod.Reset()

	if _, err := hashMethod.Write([]byte(text)); err != nil {
		return "", fmt.Errorf("failed to hash text: %w", err)
	}

	return hex.EncodeToString(hashMethod.Sum(nil)), nil
}

// returnRangeInt32 converts a string like 0-4,6-7 to [0,1,2,3,4,6,7].
func returnRangeInt32(_range string, max int32) (result []int32) {
	for _, split := range strings.Split(_range, ",") {
		ranges := strings.Split(split, "-")
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

	return result
}

func randomHex(length int) (result string) {
	buf := make([]byte, length)
	rand.Read(buf)

	return hex.EncodeToString(buf)
}

// webhookTime returns a formatted time.Time as a time accepted by webhooks.
func webhookTime(_time time.Time) string {
	return _time.Format("2006-01-02T15:04:05Z")
}

// makeExtra converts from interfaces to RawMessages. Used for extra data in payloads.
func makeExtra(extra map[string]interface{}) (out map[string]jsoniter.RawMessage, err error) {
	out = make(map[string]jsoniter.RawMessage)

	for k, v := range extra {
		p, err := jsoniter.Marshal(v)
		if err != nil {
			return out, fmt.Errorf("failed to marshal extra: %w", err)
		}

		out[k] = p
	}

	return
}
