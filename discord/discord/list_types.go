package discord

import jsoniter "github.com/json-iterator/go"

type SnowflakeList []Snowflake

func (s SnowflakeList) MarshalJSON() ([]byte, error) {
	// If len(s) == 0, return []byte("[]"), nil
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]Snowflake(s))
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

type StageInstanceList []StageInstance

func (s StageInstanceList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]StageInstance(s))
}

type StickerList []*Sticker

func (s StickerList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*Sticker(s))
}

type ScheduledEventList []ScheduledEvent

func (s ScheduledEventList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]ScheduledEvent(s))
}

type RoleList []*Role

func (s RoleList) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*Role(s))
}

type EmojiList []*Emoji

func (e EmojiList) MarshalJSON() ([]byte, error) {
	if len(e) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*Emoji(e))
}

type VoiceStateList []*VoiceState

func (v VoiceStateList) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*VoiceState(v))
}

type GuildMemberList []*GuildMember

func (m GuildMemberList) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*GuildMember(m))
}

type ChannelList []*Channel

func (c ChannelList) MarshalJSON() ([]byte, error) {
	if len(c) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*Channel(c))
}

type NullMap bool

func (n NullMap) MarshalJSON() ([]byte, error) {
	return []byte("{}"), nil
}

type NullSeq bool

func (n NullSeq) MarshalJSON() ([]byte, error) {
	return []byte("[]"), nil
}

type ActivityList []*Activity

func (s ActivityList) MarshalJSON() ([]byte, error) {
	// If len(s) == 0, return []byte("[]"), nil
	if len(s) == 0 {
		return []byte("[]"), nil
	}

	// Just marshal normally
	return jsoniter.Marshal([]*Activity(s))
}
