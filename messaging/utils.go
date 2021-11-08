package mqclients

import "strings"

// MQClients lists all current mqclients we have available.
var MQClients = []string{}

// Returns first match from a map and handles keys as non case sensitive.
func GetEntry(m map[string]interface{}, key string) interface{} {
	key = strings.ToLower(key)
	for i, k := range m {
		if strings.ToLower(i) == key {
			return k
		}
	}

	return nil
}
