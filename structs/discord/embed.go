package structs

const (
	EmbedSandwich = 16701571
	EmbedWarning = 16760839
	EmbedDanger = 14431557
)

// Embed represents a message embed on Discord.
type Embed struct {
	Title       string          `json:"title,omitempty" msgpack:"title,omitempty"`
	Type        string          `json:"type,omitempty" msgpack:"type,omitempty"`
	Description string          `json:"description,omitempty" msgpack:"description,omitempty"`
	URL         string          `json:"url,omitempty" msgpack:"url,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty" msgpack:"timestamp,omitempty"`
	Color       int             `json:"color,omitempty" msgpack:"color,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty" msgpack:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty" msgpack:"image,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty" msgpack:"thumbnail,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty" msgpack:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty" msgpack:"provider,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty" msgpack:"author,omitempty"`
	Fields      []*EmbedField   `json:"fields,omitempty" msgpack:"fields,omitempty"`
}

// EmbedFooter represents the footer of an embed.
type EmbedFooter struct {
	Text         string `json:"text" msgpack:"text"`
	IconURL      string `json:"icon_url,omitempty" msgpack:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty" msgpack:"proxy_icon_url,omitempty"`
}

// EmbedImage represents an image in an embed.
type EmbedImage struct {
	URL      string `json:"url,omitempty" msgpack:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty" msgpack:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty" msgpack:"height,omitempty"`
	Width    int    `json:"width,omitempty" msgpack:"width,omitempty"`
}

// EmbedThumbnail represents the thumbnail of an embed.
type EmbedThumbnail struct {
	URL      string `json:"url,omitempty" msgpack:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty" msgpack:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty" msgpack:"height,omitempty"`
	Width    int    `json:"width,omitempty" msgpack:"width,omitempty"`
}

// EmbedVideo represents the video of an embed.
type EmbedVideo struct {
	URL    string `json:"url,omitempty" msgpack:"url,omitempty"`
	Height int    `json:"height,omitempty" msgpack:"height,omitempty"`
	Width  int    `json:"width,omitempty" msgpack:"width,omitempty"`
}

// EmbedProvider represents the provider of an embed.
type EmbedProvider struct {
	Name string `json:"name,omitempty" msgpack:"name,omitempty"`
	URL  string `json:"url,omitempty" msgpack:"url,omitempty"`
}

// EmbedAuthor represents the author of an embed.
type EmbedAuthor struct {
	Name         string `json:"name,omitempty" msgpack:"name,omitempty"`
	URL          string `json:"url,omitempty" msgpack:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty" msgpack:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty" msgpack:"proxy_icon_url,omitempty"`
}

// EmbedField represents a field in an embed.
type EmbedField struct {
	Name   string `json:"name" msgpack:"name"`
	Value  string `json:"value" msgpack:"value"`
	Inline bool   `json:"inline,omitempty" msgpack:"inline,omitempty"`
}
