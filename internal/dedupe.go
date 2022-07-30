package internal

import (
	"time"

	"github.com/WelcomerTeam/Discord/discord"
)

func createDedupeMemberAddKey(guildID discord.Snowflake, memberID discord.Snowflake) (key string) {
	return "MA:" + guildID.String() + ":" + memberID.String()
}

func createDedupeMemberRemoveKey(guildID discord.Snowflake, memberID discord.Snowflake) (key string) {
	return "MR:" + guildID.String() + ":" + memberID.String()
}

// AddMemberDedupe creates a new dedupe.
func (sg *Sandwich) AddDedupe(key string) {
	sg.dedupeMu.Lock()
	sg.Dedupe[key] = time.Now().Add(memberDedupeExpiration).Unix()
	sg.dedupeMu.Unlock()
}

// CheckMemberDedupe returns if a dedupe is set. If true, event should be ignored.
func (sg *Sandwich) CheckDedupe(key string) (shouldDedupe bool) {
	sg.dedupeMu.RLock()
	value := sg.Dedupe[key]
	sg.dedupeMu.RUnlock()

	return time.Now().Unix() < value && value != 0
}

// RemoveMemberDedupe removes a dedupe.
func (sg *Sandwich) RemoveDedupe(key string) {
	sg.dedupeMu.Lock()
	delete(sg.Dedupe, key)
	sg.dedupeMu.Unlock()
}
