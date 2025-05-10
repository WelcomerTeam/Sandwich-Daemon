package sandwich

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/WelcomerTeam/Discord/discord"
)

func randomHex(length int) string {
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

// returnRangeInt32 converts a string like 0-4,6-7 to [0,1,2,3,4,6,7].
func returnRangeInt32(nodeCount, nodeID int32, rangeString string, max int32) (result []int32) {
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

	if nodeCount > 1 {
		filtered := make([]int32, 0, len(result))

		for _, id := range result {
			if id%nodeCount == nodeID {
				filtered = append(filtered, id)
			}
		}

		result = filtered
	}

	return result
}

func unmarshalPayload(payload *discord.GatewayPayload, out any) error {
	err := json.Unmarshal(payload.Data, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}
