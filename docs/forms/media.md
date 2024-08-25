# Media

## MediaDefiner interface

A `MediaDefiner` is an object which is capable of returning a media object for use in forms and templates.

Below is an example of how this interface can be implemented, as well as how to use the `media` package to create or merge a media object.

```go
type MyObject struct {
    // ...
}

func (o *MyObject) Media() *forms.Media {
    var firstMediaObj = media.NewMedia()

    firstMediaObj.AddCSS(
     media.CSS("http://example.com/css/style.css"),
    )

    firstMediaObj.AddJS(
     &media.JSAsset{URL: "http://example.com/js/script.js"},
    )

    var secondMediaObj = media.NewMedia()

    secondMediaObj.AddCSS(
        media.CSS("http://example.com/css/style2.css"),
    )

    var mediaMerged = firstMediaObj.Merge(secondMediaObj)
    return mediaMerged
}
```

## Asset interface

An `Asset` is an object which represents a single asset, such as a CSS or JS file.

Assets must know how to render themselves, and should be able to return the correct HTML tag for the asset.

They must also be able to return the URL of the asset.

```go
type Asset interface {
 String() string
 Render() template.HTML
}
```

### `CSS`

The `CSS` type creates a new `CSS` object.

```go
func CSS(string) Asset
```

It takes a single argument, the URL of the CSS file.

The type is defined as follows:

```go
type CSS string
```

### `JS`

The `JS` function creates a new `JS` object.

```go
func JS(string) Asset
```

It takes a single argument, the URL of the JS file.

The type is defined as follows:

```go
type JS string
```

Script tags output by this function will be rendered as `<script src="..."></script>`.

Note that this allows for no custom attributes to be added to the script tag.

### `JSAsset`

The `JSAsset` type creates a new `JSAsset` object.

```go
type JSAsset struct {
    Type string
    URL  string
}
```

It can only be instantiated by creating a new object.

```go
var jsAsset = &media.JSAsset{
    URL: "http://example.com/js/script.js",
}
```

The `Type` field is optional and the attribute will be omitted from the script tag if it is empty.

## Media interface

The media interface allows for easily passing multiple types of assets to a template.

Media objects can be merged together to create a single media object, as well as retrieving the underlying lists of resources.

```go
type Media interface {
    // AddJS adds a JS asset to the media.
    AddJS(js ...media.Asset)

    // AddCSS adds a CSS asset to the media.
    AddCSS(css ...media.Asset)

    // Merge merges the media of the other Media object into this one.
    // It returns the merged Media object - it modifies the receiver.
    Merge(other media.Media) media.Media

    // A list of JS script tags to include.
    JS() []template.HTML

    // A list of CSS link tags to include.
    CSS() []template.HTML

    // The list of raw JS urls to include.
    JSList() []media.Asset

    // The list of raw CSS urls to include.
    CSSList() []media.Asset
}
```
