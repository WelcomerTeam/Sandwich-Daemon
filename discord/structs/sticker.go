package discord

// sticker represents all structures for a sticker.

// StickerType represents the type of sticker.
type StickerType uint8

const (
	StickerTypeStandard StickerType = 1 + iota
	StickerTypeGuild
)

// StickerFormatType represents the sticker format.
type StickerFormatType uint8

const (
	StickerFormatTypePNG StickerFormatType = 1 + iota
	StickerFormatTypeAPNG
	StickerFormatTypeLOTTIE
)

// Sticker represents a sticker object.
type Sticker struct {
	ID          Snowflake          `json:"id"`
	PackID      *Snowflake         `json:"pack_id,omitempty"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Tags        *string            `json:"tags,omitempty"`
	Type        *StickerType       `json:"type"`
	FormatType  *StickerFormatType `json:"format_type"`
	Available   *bool              `json:"available,omitempty"`
	GuildID     *Snowflake         `json:"guild_id,omitempty"`
	User        *User              `json:"user,omitempty"`
	SortValue   *int               `json:"sort_value,omitempty"`
}

// MessageSticker represents a sticker in a message.
type MessageSticker struct {
	ID         Snowflake          `json:"id"`
	Name       string             `json:"name"`
	FormatType *StickerFormatType `json:"format_type"`
}
