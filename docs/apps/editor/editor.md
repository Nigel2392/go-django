# Rich Text Editor (`editor`)

The `editor` app integrates [Editor.js](https://editorjs.io/) into Go-Django as an `attrs` model field, form field, Stimulus-powered widget, HTML renderer, and revision diff provider. Block content is stored as structured JSON (`dbtype.JSON`) and hydrated into typed `FeatureBlock` instances so blocks can be validated, tuned, prefetched, and rendered cleanly on the server.

## Installation and Feature Imports

Register `editor.NewAppConfig` with your `django.App` and blank-import the built-in feature packages you want available in the registry. Without at least one feature package imported, the editor registry starts empty.

```go
package main

import (
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	_ "github.com/Nigel2392/go-django/contrib/editor/features"
	_ "github.com/Nigel2392/go-django/contrib/editor/features/images"
	_ "github.com/Nigel2392/go-django/contrib/editor/features/links"
	"github.com/Nigel2392/go-django/contrib/images"
	"github.com/Nigel2392/go-django/contrib/pages"
	django "github.com/Nigel2392/go-django/src"
)

func main() {
	app := django.App(
		django.Apps(
			pages.NewAppConfig,
			images.NewAppConfig(&images.Options{ /* ... */ }),
			blocks.NewAppConfig,
			editor.NewAppConfig,
		),
	)
	_ = app
}
```

Do note the app dependencies for the optional feature subpackages:

- `github.com/Nigel2392/go-django/contrib/editor/features` has no extra app dependencies and registers core block, tune, and inline features.
- `github.com/Nigel2392/go-django/contrib/editor/features/images` registers the `image` and `images` blocks and **panics during `Ready`** if either the `editor` or `images` app is not installed (see [Images](../images.md)).
- `github.com/Nigel2392/go-django/contrib/editor/features/links` registers the `pagelink` inline tool and relies on the [Pages](../pages/readme.md) app (`pages_redirect`, `admin:apps:model:chooser:list`, and `pages.NewPageQuerySet()`).

During `NewAppConfig()`, the `editor` app performs the following setup:

- Registers Telepath adapters (`JSONAdapter` for `EditorJSBlockData` and `EditorJSData`, and an object adapter for `EditorJSWidget` under `"editor.widgets.EditorJSWidget"`) for compatibility with [Blocks](../blocks.md) such as `StreamBlock` and `ListBlock`.
- Registers a global `attrs.RegisterFormFieldGetter` hook for `*editor.EditorJSBlockData` so fields defined with `attrs.NewField` automatically receive an `EditorJSField`.
- Embeds and registers static assets under `editorjs/` (`.js` and `.css`) with `staticfiles`.
- During `Ready`, creates a route group at `editor.ROUTE_PATH` (`"/__go-editorjs__"`), calls `feature.OnRegister(editorNs)` on each registered feature, and mounts the group on `django.Global.Mux` if any feature registered child routes.

---

## Configuration Settings and Globals

Behavior can be customized via `django.Configure(...)` settings and package-level variables:

```go
const (
	ROUTE_PATH = "/__go-editorjs__"

	APPVAR_WRAP_RENDERED_BLOCKS  = "APPVAR_WRAP_RENDERED_BLOCKS"
	APPVAR_ENABLE_INLINE_TOOLBAR = "APPVAR_ENABLE_INLINE_TOOLBAR"
)

var (
	RENDER_ERRORS = true
)
```

- **`APPVAR_WRAP_RENDERED_BLOCKS`** (`bool`, default `true`): When `true`, `EditorJSBlockData.Render()` wraps every rendered block in `<div class="editorjs-block block-<type>" data-block-id="<id>">...</div>`.
- **`APPVAR_ENABLE_INLINE_TOOLBAR`** (`bool`, default `true`): Sets `inlineToolbar: true` in the Editor.js tool configuration for every non-inline block feature.
- **`ROUTE_PATH`**: Base URL path (`"/__go-editorjs__"`) passed to `feature.OnRegister(...)` during `Ready`.
- **`RENDER_ERRORS`** (`bool`, default `true`): When `true`, if a block's `Render` method returns an error, `EditorJSBlockData.Render()` writes `Error (<type>): <err>` directly into the output buffer instead of silently omitting it.

---

## Data Types and Defining Model Fields

The `editor` package defines three core types representing Editor.js payloads in Go:

```go
type BlockData struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
	Tunes map[string]interface{} `json:"tunes,omitempty"`
}

type EditorJSData struct {
	Time    int64       `json:"time"`
	Blocks  []BlockData `json:"blocks"`
	Version string      `json:"version"`
}

type EditorJSBlockData struct {
	Time     int64          `json:"time"`
	Blocks   []FeatureBlock `json:"blocks"`
	Version  string         `json:"version"`
	Features []BaseFeature  `json:"-"`
}
```

`EditorJSBlockData` implements `DBType() dbtype.Type` (`dbtype.JSON`), `driver.Valuer`, `sql.Scanner`, `json.Marshaler`, `json.Unmarshaler`, `Render() (template.HTML, error)`, `MustRender() template.HTML`, and `HTML() template.HTML`.

### Option 1: `editor.NewField` (Recommended)

Use `editor.NewField` with `editor.FieldConfig` inside `FieldDefs` when you want explicit per-field feature lists, column overrides, and form options without adding extra methods to your model:

```go
type FieldConfig struct {
	Label         func(ctx context.Context) string
	HelpText      func(ctx context.Context) string
	Features      []string
	Default       *EditorJSBlockData
	ReadOnly      bool
	Blank         bool
	Nullable      bool
	Column        string
	Embedded      bool
	Attributes    map[string]any
	FormFieldOpts []func(fields.Field)
}

func NewField(forModel attrs.Definer, name string, cnf ...FieldConfig) *Field
```

For example, in a model's `FieldDefs`:

```go
package blog

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Article struct {
	models.Model `table:"articles"`
	ID           int64                     `attrs:"primary;readonly"`
	Title        string
	Body         *editor.EditorJSBlockData
}

func (a *Article) FieldDefs(ctx context.Context) attrs.Definitions {
	return a.Model.Define(ctx, a,
		attrs.Unbound("ID"),
		attrs.Unbound("Title"),
		editor.NewField(a, "Body", editor.FieldConfig{
			Label:    trans.S("Body"),
			HelpText: trans.S("Rich article content."),
			Blank:    true,
			Nullable: true,
			Features: []string{
				"header",
				"paragraph",
				"list",
				"text-align",
				"marker",
				"inline-code",
			},
		}),
	)
}
```

When `Features` is empty, `editor.NewField` enables all currently registered features. On `SetValue`/`Scan`, `editor.Field` also assigns `data.Features` directly on the `*EditorJSBlockData` value.

### Option 2: `attrs.NewField` with `Get<FieldName>Features()`

If you define the field with standard `attrs.NewField`, `NewAppConfig()` registers a form-field getter for `*editor.EditorJSBlockData` that looks for a `Get<FieldName>Features() []string` method on the model instance:

```go
func (a *Article) GetBodyFeatures() []string {
	return []string{"header", "paragraph", "list"}
}
```

If that method is absent, all registered features are enabled in the form field.

---

## Form Field, Widget, and Blocks Integration

You can construct form fields and widgets directly when building standalone forms or embedding an editor inside a [Blocks](../blocks.md) `StreamBlock`:

```go
func EditorJSField(features []string, opts ...func(fields.Field)) fields.Field
func NewEditorJSWidget(features ...string) *EditorJSWidget
```

For example, inside a `blocks.StreamBlock`:

```go
blocks.WithBlockField[*blocks.StreamBlock]("text", blocks.FormFieldBlock(
	editor.EditorJSField(nil),
	&editor.EditorJSBlockData{},
	blocks.WithLabel[*blocks.FieldBlock](trans.S("Text")),
	blocks.WithHelpText[*blocks.FieldBlock](trans.S("A block of text.")),
))
```

### Widget Behavior and JavaScript Events

`EditorJSWidget` renders a Stimulus container (`data-controller="editorjs-widget"`) containing a hidden `<input>` (`data-editorjs-widget-target="input"`) and an editor mount element (`data-editorjs-widget-target="editor"`). Its `Media()` method merges the CSS/JS assets of every configured feature together with `editorjs/css/index.css` and `editorjs/js/index.js`.

In the browser:

- Global state is exposed at `window.GoEditorJS` (`{ Widget: EditorJSWidget, editors: [] }`).
- Custom DOM events are dispatched on the hidden `<input>` element (`event.detail` contains `{ editor, widget }` plus `data` or `reason`):
  - `editorjs:ready`: Fired when Editor.js initializes and syncs initial JSON to the hidden input.
  - `editorjs:change`: Fired whenever Editor.js content changes, keeping the hidden input synchronized.
  - `editorjs:error`: Fired if Editor.js fails to initialize.
- Form submit buttons inside the closest `<form>` trigger `this.editor.save()` before submitting so pending edits are flushed into the hidden input.
- For Telepath (`contrib/blocks`), `contrib/editor` registers `editor.widgets.EditorJSWidget` on `window.telepath` automatically.

---

## Feature Registry

Features and block tunes are managed through the global `EditorRegistry` and package-level helpers:

```go
func Register(feature BaseFeature)
func Unregister(featureName string)
func Features(f ...string) []BaseFeature
func FeatureNames(f ...BaseFeature) []string
func Tune(tuneName string)
func TuneFeature(featureName string, tuneName string)
func ValueToForm(data *EditorJSBlockData) *EditorJSData
func ValueToGo(tools []string, data EditorJSData) (*EditorJSBlockData, error)
```

- **`Register(feature)`**: Registers or replaces a feature keyed by `feature.Name()`.
- **`Unregister(featureName)`**: Removes a feature and any per-feature tunes attached to it.
- **`Features(names...)`**: Looks up features by name in order; if no names are passed, returns all registered features.
- **`FeatureNames(features...)`**: Extracts `[]string` names from a slice of `BaseFeature`.
- **`Tune(tuneName)`**: Applies a registered block tune globally to all rendered editor widgets (whose enabled feature list includes `tuneName`).
- **`TuneFeature(featureName, tuneName)`**: Applies a block tune to a specific block feature only.

Do note that `Tune(...)` and `TuneFeature(...)` configure the Editor.js UI config (`tunes` in the JSON config). When restricting features per field via `FieldConfig.Features`, make sure any tune names (such as `"text-align"`, `"text-color"`, or `"background-color"`) are included in the field's `Features` slice as well.

---

## Built-in Features

### `contrib/editor/features`

Blank-importing `github.com/Nigel2392/go-django/contrib/editor/features` registers the following tools:

| Feature Name | Kind | JS Constructor | Notes / HTML Output |
| :--- | :--- | :--- | :--- |
| `header` | Block | `Header` | `<h1..h6 class="heading ...">` using `data.level` (defaults to `1`) |
| `paragraph` | Block | `Paragraph` | `<p class="paragraph ...">` with raw inner HTML |
| `delimiter` | Block | `Delimiter` | `<div class="delimiter ...">` with three `<span class="delimiter__dot">` |
| `alert` | Block | `Alert` | `<div class="alert alert-<type> align-<align> ...">`; validates `type` against `primary`, `secondary`, `success`, `danger`, `warning`, `info`, `light`, `dark` |
| `list` | Block | `EditorjsList` | Nested `<ol class="list list-ordered">`, `<ul class="list list-unordered">`, or `<ul class="list list-checklist">` (`is-checked` on checked `<li>`) |
| `table` | Block | `Table` | `<table class="table ...">` (`table-headings`, `table-stretched`) |
| `text-align` | Block Tune (global) | `AlignmentBlockTune` | Adds `text-align-<alignment>` class and `data-text-align` attribute to tuned block |
| `text-color` | Block Tune (global) | `TextColorTune` | Wraps block in `<div class="text-color ..." style="--text-color: <color>;">` |
| `background-color` | Block Tune (global) | `BackgroundColorTune` | Wraps block in `<div class="background-color ..." style="--background-color: <color>;">` (plus `background-stretched` if stretched) |
| `marker` | Inline | `Marker` | Rewrites `<mark class="cdx-marker">` to `<mark class="marker">` |
| `inline-code` | Inline | `InlineCode` | Rewrites `<span class="inline-code">` to `<span class="inline-code">` |
| `underline` | Inline | `Underline` | Rewrites `<u class="cdx-underline">` to `<u class="underline">` |
| `strikethrough` | Inline | `Strikethrough` | Rewrites `<s class="cdx-strikethrough">` to `<s class="strikethrough">` |
| `hotkey` | Inline | `EditorJSInlineHotkey` | Rewrites `<kbd class="editorjs-inline-hotkey">` to `<kbd class="hotkey">` |

### `contrib/editor/features/images`

Requires the `editor` and `images` apps:

| Feature Name | Kind | JS Constructor | Notes |
| :--- | :--- | :--- | :--- |
| `image` | Block (`PrefetchableFeature`) | `GoDjangoImageTool` | Uses the [Chooser](../admin/app.md#modal-chooser) modal for `images.Image`; renders `<img data-block-id="..." src="..." alt="..." data-id="..." class="image ..." />` |
| `images` | Block (`PrefetchableFeature`) | `GoDjangoImagesTool` | Multi-image selector; renders `<div class="multi-images ...">` containing `<div class="image-wrapper"><img ... /></div>` |

### `contrib/editor/features/links`

Requires the `pages` app:

| Feature Name | Kind | JS Constructor | Notes |
| :--- | :--- | :--- | :--- |
| `pagelink` | Inline | `PageLinkTool` | Opens the `pages` chooser modal; during rendering, queries `pages.NewPageQuerySet().GetNodesByIDs(...)` in batch and rewrites `<a class="page-link" data-page-id="...">` elements with live `href` URLs via `pages.URLPath` |

---

## Authoring Custom Features

Every feature implements `editor.BaseFeature` and one or more capability interfaces depending on its role:

```go
type BaseFeature interface {
	Name() string
	Config(widgetContext ctx.Context) map[string]interface{}
	OnRegister(mux.Multiplexer) error
	OnValidate(BlockData) error
	Constructor() string
	Media() media.Media
}

type FeatureBlockRenderer interface {
	BaseFeature
	Render(BlockData) FeatureBlock
}

type PrefetchableFeature interface {
	BaseFeature
	Render(BlockData) FeatureBlock // returned block must implement PrefetchableFeatureBlock
	PrefetchData(ctx context.Context, data []BlockData) (map[string]BlockData, error)
}

type BlockTuneFeature interface {
	BaseFeature
	Tune(FeatureBlock, interface{}) FeatureBlock
}

type InlineFeature interface {
	BaseFeature
	ParseInlineData(soup *goquery.Selection) error
}
```

The `contrib/editor/features` package provides ready-to-use building blocks (`features.BaseFeature`, `features.Block`, `features.BlockTune`, `features.WrapperTune`, `features.PrefetchableFeature`, `features.InlineFeature`, `features.FeatureBlock`, and `features.AttributeWrapperBlock`) so you rarely need to implement every method by hand:

```go
package myeditor

import (
	"context"
	"fmt"
	"html"
	"io"

	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/editor/features"
)

var QuoteFeature = &features.Block{
	BaseFeature: features.BaseFeature{
		Type:          "quote",
		JSConstructor: "Quote",
		JSFiles: []string{
			"myapp/js/editorjs/quote.js",
		},
		Validate: func(bd editor.BlockData) error {
			if text, _ := bd.Data["text"].(string); text == "" {
				return fmt.Errorf("quote text cannot be empty")
			}
			return nil
		},
	},
	RenderFunc: func(fb editor.FeatureBlock, ctx context.Context, w io.Writer) error {
		text, _ := fb.Data().Data["text"].(string)
		caption, _ := fb.Data().Data["caption"].(string)
		_, err := fmt.Fprintf(
			w,
			`<blockquote class="quote %s" data-block-id="%s"><p>%s</p><cite>%s</cite></blockquote>`,
			html.EscapeString(fb.ClassName()),
			html.EscapeString(fb.ID()),
			text,
			html.EscapeString(caption),
		)
		return err
	},
}

func init() {
	editor.Register(QuoteFeature)
}
```

If you implement a `PrefetchableFeature`, `PrefetchData` receives all `BlockData` items for your block type across the document and returns a `map[blockID]BlockData`. Every block returned by your `Render(BlockData)` must implement `editor.PrefetchableFeatureBlock` (such as `*features.PrefetchableFeatureBlock`), otherwise `EditorJSBlockData.Render()` panics when calling `WithData`.

---

## Rendering Pipeline

To render saved Editor.js content in templates or Go code, call `Render()`, `MustRender()`, or `HTML()` on `*editor.EditorJSBlockData`:

```html
<article class="post-body">
    {{ $page := .Get "blog_page" }}
    {{ $page.Editor.HTML }}
</article>
```

Internally, `EditorJSBlockData.Render()` runs in three stages:

1. **Batch Prefetch**: Any feature in `e.Features` implementing `PrefetchableFeature` runs `PrefetchData(ctx, blocks)` once for the whole document, then passes enriched block data back via `PrefetchableFeatureBlock.WithData`.
2. **Block Rendering**: Iterates `e.Blocks` in order. If `APPVAR_WRAP_RENDERED_BLOCKS` is `true` (default), each block is wrapped in `<div class="editorjs-block block-<type>" data-block-id="<id>">...</div>`. On render error, if `editor.RENDER_ERRORS` is `true`, an inline error string is emitted.
3. **Inline Post-Processing**: If any enabled feature implements `InlineFeature`, the rendered HTML is parsed with `goquery` and each `InlineFeature.ParseInlineData(bodySelection)` mutates matching inline DOM nodes (for example, resolving `pagelink` URLs in a single batched query).

`MustRender()` and `HTML()` log any error via `logger.Errorf` and return an empty `template.HTML` on failure.

---

## Revision Comparison (`compare`)

In `compare.go`, `contrib/editor` registers `EditorComparison` with [Revision Comparison](../admin/app.md#revision-comparison) for `reflect.TypeOf((*EditorJSBlockData)(nil))`:

```go
func EditorComparison(
	ctx context.Context,
	label func(context.Context) string,
	fieldName string,
	modelMeta attrs.ModelMeta,
	old, new attrs.Definer,
) (compare.Comparison, error)
```

When comparing two revisions (for example in [Revisions](../revisions.md)):

- `HasChanged()` delegates to `EditorJSFormField.HasChanged`, comparing `Version`, block count, and each block's `Data()`, while intentionally ignoring the `Time` timestamp.
- `HTMLDiff()` computes a longest-common-subsequence (LCS) alignment over block IDs (falling back to `<type>#<index>` when an ID is empty), wraps added/removed blocks in `<div class="diff-added">` / `<div class="diff-removed">`, and runs `compare.DiffText` inside `<div class="diff-modified">` for changed blocks that kept the same ID.
