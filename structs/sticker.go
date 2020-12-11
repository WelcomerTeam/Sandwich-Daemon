package structs

import "github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"

// Sticker represents a sticker object
type Sticker struct {
	ID           snowflake.ID `json:"id" msgpack:"id"`
	PackID       snowflake.ID `json:"pack_id" msgpack:"pack_id"`
	Name         string       `json:"name" msgpack:"name"`
	Description  string       `json:"description" msgpack:"description"`
	Tags         []string     `json:"tags,omitempty" msgpack:"tags,omitempty"`
	Asset        string       `json:"asset" msgpack:"asset"`
	PreviewAsset string       `json:"preview_asset" msgpack:"preview_asset"`
	FormatType   StickerType  `json:"format_type" msgpack:"format_type"`
}

type StickerType int64

// sticker types.
const (
	StickerPNG StickerType = iota + 1
	StickerAPNG
	StickerLOTTIE
)
