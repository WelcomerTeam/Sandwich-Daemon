package internal

import (
	"time"

	"github.com/WelcomerTeam/Discord/discord"
)

func createDedupeMemberAddKey(guildID, memberID discord.Snowflake) string {
	return "MA:" + guildID.String() + ":" + memberID.String()
}

func createDedupeMemberRemoveKey(guildID, memberID discord.Snowflake) string {
	return "MR:" + guildID.String() + ":" + memberID.String()
}

// AddDedupe creates a new dedupe.
func (sg *Sandwich) AddDedupe(key string) {
	sg.dedupeMu.Lock()
	sg.Dedupe[key] = time.Now().Add(memberDedupeExpiration).Unix()
	sg.dedupeMu.Unlock()
}

// CheckDedupe returns if a dedupe is set. If true, event should be ignored.
func (sg *Sandwich) CheckDedupe(key string) bool {
	sg.dedupeMu.RLock()
	value := sg.Dedupe[key]
	sg.dedupeMu.RUnlock()

	return time.Now().Unix() < value && value != 0
}

// CheckAndAddDedupe returns if a dedupe is set. If true, event should be ignored.
// Adds dedupe if not set.
func (sg *Sandwich) CheckAndAddDedupe(key string) bool {
	sg.dedupeMu.Lock()
	defer sg.dedupeMu.Unlock()

	value := sg.Dedupe[key]

	has := time.Now().Unix() < value && value != 0

	if !has {
		sg.Dedupe[key] = time.Now().Add(memberDedupeExpiration).Unix()
	}

	// println("CheckAndAddDedupe", key, has)

	return has
}

// RemoveMemberDedupe removes a dedupe.
func (sg *Sandwich) RemoveDedupe(key string) {
	sg.dedupeMu.Lock()
	delete(sg.Dedupe, key)
	sg.dedupeMu.Unlock()
}
