package discord

// sticker represents all structures for a sticker.

// StickerType represents the type of sticker.
type StickerType uint16

const (
	StickerTypeStandard StickerType = 1 + iota
	StickerTypeGuild
)

// StickerFormatType represents the sticker format.
type StickerFormatType uint16

const (
	StickerFormatTypePNG StickerFormatType = 1 + iota
	StickerFormatTypeAPNG
	StickerFormatTypeLOTTIE
)

// Sticker represents a sticker object.
type Sticker struct {
	PackID      *Snowflake        `json:"pack_id,omitempty"`
	GuildID     *Snowflake        `json:"guild_id,omitempty"`
	User        *User             `json:"user,omitempty"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        string            `json:"tags"`
	ID          Snowflake         `json:"id"`
	SortValue   int32             `json:"sort_value"`
	Type        StickerType       `json:"type"`
	FormatType  StickerFormatType `json:"format_type"`
	Available   bool              `json:"available"`
}

// MessageSticker represents a sticker in a message.
type MessageSticker struct {
	Name       string            `json:"name"`
	ID         Snowflake         `json:"id"`
	FormatType StickerFormatType `json:"format_type"`
}
