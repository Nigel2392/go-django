# Media

The `media` package (`github.com/Nigel2392/go-django/src/forms/media`) collects, deduplicates, orders, and renders CSS and JavaScript assets required by forms and widgets.

---

## `MediaDefiner` Interface

Forms, widgets, and custom components expose required assets by implementing `MediaDefiner`:

```go
type MediaDefiner interface {
	Media() Media
}
```

When a `Form` computes `form.Media()`, it creates an empty `Media` container and merges `widget.Media()` from each field's widget.

```go
type MyObject struct{}

func (o *MyObject) Media() media.Media {
	firstMediaObj := media.NewMedia()
	firstMediaObj.AddCSS(
		media.CSS("https://example.com/css/style.css"),
	)
	firstMediaObj.AddJS(
		&media.JSAsset{URL: "https://example.com/js/script.js"},
	)

	secondMediaObj := media.NewMedia()
	secondMediaObj.AddCSS(
		media.CSS("https://example.com/css/style2.css"),
	)

	return firstMediaObj.Merge(secondMediaObj)
}
```

---

## Asset Types and Interfaces

### `Asset` and `WeightedAsset`

An `Asset` represents a single CSS or JS resource capable of returning its URL/identifier (`String()`) and HTML tag (`Render()`). `WeightedAsset` extends `Asset` with a `Priority()` integer where higher values are intended to sort earlier:

```go
type Asset interface {
	String() string
	Render() template.HTML
}

type WeightedAsset interface {
	Asset
	Priority() int // The higher the priority, the earlier it is included.
}
```

### `CSS` and `JS`

`CSS` and `JS` are named `string` types, **not** constructor functions — convert a URL string using Go type conversions such as `media.CSS("/static/app.css")` or `media.JS("/static/app.js")`:

```go
type CSS string

func (c CSS) String() string        { return string(c) }
func (c CSS) Render() template.HTML // <link rel="stylesheet" href="...">

type JS string

func (j JS) String() string         { return string(j) }
func (j JS) Render() template.HTML  // <script src="..."></script>
```

Note that `JS` does not support extra `<script>` attributes such as `type="module"`. Use `JSAsset` when you need a `type` attribute.

### `JSAsset`

```go
type JSAsset struct {
	Type string
	URL  string
}
```

`JSAsset` renders `<script src="..."></script>` and includes `type="..."` before `src` when `Type` is non-empty:

```go
jsAsset := &media.JSAsset{
	Type: "module",
	URL:  "https://example.com/js/module.js",
}
```

### `PriorityAsset` and `AssetWithPriority`

Wrap any `Asset` with an explicit priority using `AssetWithPriority`:

```go
type PriorityAsset struct {
	Asset
	Importance int // The higher the priority, the earlier it is included.
}

func AssetWithPriority(asset Asset, priority int) *PriorityAsset
func (p *PriorityAsset) Priority() int
```

Example:

```go
m := media.NewMedia()
m.AddJS(
	media.AssetWithPriority(media.JS("/static/vendor.js"), 100),
	media.JS("/static/app.js"),
)
```

---

## `Media`, `AddableMedia`, and `MediaObject`

### `AddableMedia` and `Media`

```go
type AddableMedia interface {
	AddJS(js ...Asset)
	AddCSS(css ...Asset)
}

type Media interface {
	// Merge merges the assets of other into this Media object, modifying the receiver.
	Merge(other Media) Media

	// JS returns rendered <script> tags sorted by Priority (highest first).
	JS() []template.HTML

	// CSS returns rendered <link> tags.
	CSS() []template.HTML

	// JSList returns raw JS Asset entries in insertion order.
	JSList() []Asset

	// CSSList returns raw CSS Asset entries in insertion order.
	CSSList() []Asset

	AddableMedia
}
```

### `MediaObject` and `NewMedia`

`MediaObject` is the default implementation returned by `media.NewMedia()`:

```go
type MediaObject struct {
	Css *orderedmap.OrderedMap[string, Asset]
	Js  *orderedmap.OrderedMap[string, Asset]
}

func NewMedia() *MediaObject
func (m *MediaObject) String() string
```

- `AddJS` / `AddCSS` key assets by `asset.String()`, updating any existing entry with the same URL.
- `Merge(other)` appends assets from `other` whose `String()` key is not already present in `m`, modifying and returning `m`.

### `CSS()` Priority-Sorting Caveat

In `MediaObject.JS()`, assets are collected into a slice, sorted stably by `WeightedAsset.Priority()` (higher priority first), and rendered from the sorted slice. In `MediaObject.CSS()`, the method sorts a local `cssAssets` slice by priority, **but then loops over the unsorted `m.Css` ordered map** when building the returned `[]template.HTML`. As a result, `PriorityAsset` weights affect `JS()` output order, while `CSS()` always renders in insertion order.
