package internal

import (
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
)

func createDedupeMemberAddKey(guildID discord.GuildID, memberID discord.UserID) string {
	return "MA:" + discord.Snowflake(guildID).String() + ":" + discord.Snowflake(memberID).String()
}

func createDedupeMemberRemoveKey(guildID discord.GuildID, memberID discord.UserID) string {
	return "MR:" + discord.Snowflake(guildID).String() + ":" + discord.Snowflake(memberID).String()
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

// CheckMemberDedupe returns if a dedupe is set. If true, event should be ignored.
// Adds dedupe if not set.
func (sg *Sandwich) CheckAndAddDedupe(key string) bool {
	value, ok := sg.Dedupe.Load(key)

	has := time.Now().Unix() < value && value != 0
	if !has || !ok {
		sg.Dedupe.Store(key, time.Now().Add(memberDedupeExpiration).Unix())
	}
	// println("CheckAndAddDedupe", key, has)
	return has && ok
}

// RemoveDedupe removes a dedupe.
func (sg *Sandwich) RemoveDedupe(key string) {
	sg.Dedupe.Delete(key)
}
