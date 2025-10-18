package media

import (
	"html/template"
)

type Asset interface {
	String() string
	Render() template.HTML
}

type WeightedAsset interface {
	Asset
	Priority() int // The higher the priority, the earlier it is included.
}

type AddableMedia interface {
	// AddJS adds a JS asset to the media.
	AddJS(js ...Asset)

	// AddCSS adds a CSS asset to the media.
	AddCSS(css ...Asset)
}

type Media interface {
	// Merge merges the media of the other Media object into this one.
	// It returns the merged Media object - it modifies the receiver.
	Merge(other Media) Media

	// A list of JS script tags to include.
	JS() []template.HTML

	// A list of CSS link tags to include.
	CSS() []template.HTML

	// The list of raw JS urls to include.
	JSList() []Asset

	// The list of raw CSS urls to include.
	CSSList() []Asset

	AddableMedia
}

type MediaDefiner interface {
	Media() Media
}

var _ Media = (*MediaObject)(nil)
var _ AddableMedia = (*MediaObject)(nil)
