package internal

import (
	"encoding/hex"
	"hash"
	"strconv"
	"strings"
	"time"
)

const (
	daySeconds    = 86400
	hourSeconds   = 3600
	minuteSeconds = 60
)

type void struct{}

func replaceIfEmpty(v string, s string) string {
	if v == "" {
		return s
	}

	return v
}

func returnError(err error) string {
	if err != nil {
		return err.Error()
	}

	return ""
}

// quickHash returns hash from method and input.
func quickHash(hashMethod hash.Hash, text string) (result string, err error) {
	hashMethod.Reset()

	if _, err := hashMethod.Write([]byte(text)); err != nil {
		return "", err
	}

	return hex.EncodeToString(hashMethod.Sum(nil)), nil
}

// returnRange converts a string like 0-4,6-7 to [0,1,2,3,4,6,7].
func returnRange(_range string, max int) (result []int) {
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

// webhookTime returns a formatted time.Time as a time accepted by webhooks.
func webhookTime(_time time.Time) string {
	return _time.Format("2006-01-02T15:04:05Z")
}
