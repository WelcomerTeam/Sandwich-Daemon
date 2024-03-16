package structs

import (
	"github.com/WelcomerTeam/Discord/discord"
	jsoniter "github.com/json-iterator/go"
)

type SnowflakeList []discord.Snowflake

func (s SnowflakeList) MarshalJSON() ([]byte, error) {
	// If len(s) == 0, return []byte("[]"), nil
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]discord.Snowflake(s))
}

type StringList []string

func (s StringList) MarshalJSON() ([]byte, error) {
	// If len(s) == 0, return []byte("[]"), nil
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]string(s))
}

type StageInstanceList []discord.StageInstance

func (s StageInstanceList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]discord.StageInstance(s))
}

type StickerList []discord.Sticker

func (s StickerList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]discord.Sticker(s))
}

type ScheduledEventList []discord.ScheduledEvent

func (s ScheduledEventList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]discord.ScheduledEvent(s))
}

type StateRoleList []*StateRole

func (s StateRoleList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*StateRole(s))
}

type EmojiList []discord.Emoji

func (e EmojiList) MarshalJSON() ([]byte, error) {
	if len(e) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]discord.Emoji(e))
}

type VoiceStateList []*discord.VoiceState

func (v VoiceStateList) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*discord.VoiceState(v))
}

type GuildMemberList []*discord.GuildMember

func (m GuildMemberList) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*discord.GuildMember(m))
}

type ChannelList []*discord.Channel

func (c ChannelList) MarshalJSON() ([]byte, error) {
	if len(c) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*discord.Channel(c))
}
