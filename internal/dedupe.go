package internal

import (
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
)

func createDedupeMemberAddKey(guildID discord.Snowflake, memberID discord.Snowflake) string {
	return "MA:" + guildID.String() + ":" + memberID.String()
}

func createDedupeMemberRemoveKey(guildID discord.Snowflake, memberID discord.Snowflake) string {
	return "MR:" + guildID.String() + ":" + memberID.String()
}

// AddDedupe creates a new dedupe.
func (sg *Sandwich) AddDedupe(key string) {
	sg.Dedupe.Store(key, time.Now().Add(memberDedupeExpiration).Unix())
}

// CheckDedupe returns if a dedupe is set. If true, event should be ignored.
func (sg *Sandwich) CheckDedupe(key string) bool {
	value, ok := sg.Dedupe.Load(key)

	if !ok {
		return false
	}

	return time.Now().Unix() < value && value != 0
}

// RemoveDedupe removes a dedupe.
func (sg *Sandwich) RemoveDedupe(key string) {
	sg.Dedupe.Delete(key)
}
