package discord

// embed.go contains all structures for constructing embeds

type EmbedType string

const (
	EmbedTypeDefault EmbedType = "rich"
	EmbedTypeRich    EmbedType = "rich"
	EmbedTypeImage   EmbedType = "image"
	EmbedTypeVideo   EmbedType = "video"
	EmbedTypeGif     EmbedType = "gifv"
	EmbedTypeArticle EmbedType = "article"
	EmbedTypeLink    EmbedType = "link"
)

// Embed represents a message embed.
type Embed struct {
	Video       *EmbedVideo     `json:"video,omitempty"`
	Timestamp   *Timestamp      `json:"timestamp,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Type        EmbedType       `json:"type,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Title       string          `json:"title,omitempty"`
	Fields      EmbedFieldList  `json:"fields,omitempty"`
	Color       int32           `json:"color,omitempty"`
}

func NewEmbed(embedType EmbedType) *Embed {
	return &Embed{
		Type: embedType,
	}
}

func (e *Embed) SetTitle(title string) *Embed {
	return e
}

func (e *Embed) SetDescription(description string) *Embed {
	e.Description = description

	return e
}

func (e *Embed) SetURL(url string) *Embed {
	e.URL = url

	return e
}

func (e *Embed) SetTimestamp(timestamp *Timestamp) *Embed {
	e.Timestamp = timestamp

	return e
}

func (e *Embed) SetColor(color int32) *Embed {
	e.Color = color

	return e
}

func (e *Embed) SetFooter(footer *EmbedFooter) *Embed {
	e.Footer = footer

	return e
}

func (e *Embed) SetImage(image *EmbedImage) *Embed {
	e.Image = image

	return e
}

func (e *Embed) SetThumbnail(thumbnail *EmbedThumbnail) *Embed {
	e.Thumbnail = thumbnail

	return e
}

func (e *Embed) SetVideo(video *EmbedVideo) *Embed {
	e.Video = video

	return e
}

func (e *Embed) SetProvider(provider *EmbedProvider) *Embed {
	e.Provider = provider

	return e
}

func (e *Embed) SetAuthor(author *EmbedAuthor) *Embed {
	e.Author = author

	return e
}

func (e *Embed) AddField(field EmbedField) *Embed {
	e.Fields = append(e.Fields, field)

	return e
}

// EmbedFooter represents the footer of an embed.
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

func NewEmbedFooter(text, iconURL string) *EmbedFooter {
	return &EmbedFooter{
		Text:    text,
		IconURL: iconURL,
	}
}

// EmbedImage represents an image in an embed.
type EmbedImage struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int32  `json:"height,omitempty"`
	Width    int32  `json:"width,omitempty"`
}

func NewEmbedImage(url string) *EmbedImage {
	return &EmbedImage{
		URL: url,
	}
}

// EmbedThumbnail represents the thumbnail of an embed.
type EmbedThumbnail struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int32  `json:"height,omitempty"`
	Width    int32  `json:"width,omitempty"`
}

func NewEmbedThumbnail(url string) *EmbedThumbnail {
	return &EmbedThumbnail{
		URL: url,
	}
}

// EmbedVideo represents the video of an embed.
type EmbedVideo struct {
	URL    string `json:"url,omitempty"`
	Height int32  `json:"height,omitempty"`
	Width  int32  `json:"width,omitempty"`
}

func NewEmbedVideo(url string) *EmbedVideo {
	return &EmbedVideo{
		URL: url,
	}
}

// EmbedProvider represents the provider of an embed.
type EmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

func NewEmbedProvider(name, url string) *EmbedProvider {
	return &EmbedProvider{
		Name: name,
		URL:  url,
	}
}

// EmbedAuthor represents the author of an embed.
type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

func NewEmbedAuthor(name, url, iconURL string) *EmbedAuthor {
	return &EmbedAuthor{
		Name:    name,
		URL:     url,
		IconURL: iconURL,
	}
}

// EmbedField represents a field in an embed.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

func NewEmbedField(name, value string, inline bool) *EmbedField {
	return &EmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}
