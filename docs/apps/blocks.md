# Blocks

The `blocks` app (`github.com/Nigel2392/go-django/contrib/blocks`) provides composable, structured content blocks that serialize to JSON in the database and render interactive form editors using Telepath and Stimulus on the frontend.

You can combine scalar field blocks with container blocks (`StructBlock`, `ListBlock`, and `StreamBlock`) to build rich, nested content schemas directly on your Go models.

## Table of Contents

- [Installation and App Initialization](#installation-and-app-initialization)
- [Block Interface and Core Types](#block-interface-and-core-types)
- [Field Blocks](#field-blocks)
- [Container Blocks](#container-blocks)
  - [StructBlock](#structblock)
  - [ListBlock](#listblock)
  - [StreamBlock](#streamblock)
- [Option Functions](#option-functions)
- [Value Types and Model Integration](#value-types-and-model-integration)
- [Forms, Widgets, and Validation Errors](#forms-widgets-and-validation-errors)
- [Template Rendering](#template-rendering)
- [Notes and Caveats](#notes-and-caveats)

---

## Installation and App Initialization

Register `blocks.NewAppConfig` in `django.Apps(...)`:

```go
package main

import (
	"github.com/Nigel2392/go-django/contrib/blocks"
	django "github.com/Nigel2392/go-django/src"
)

func main() {
	app := django.App(
		django.Apps(
			blocks.NewAppConfig,
		),
	)
	_ = app
}
```

Package initialization and `blocks.NewAppConfig()` set up the following automatically:

- **Database JSON mapping**: Package `init()` registers `dbtype.JSON` for `*blocks.StreamBlockValue` and `*blocks.ListBlockValue` (and `BoundValue[T].DBType()` also returns `dbtype.JSON`).
- **Model form field getters**: Package `init()` registers `attrs.RegisterFormFieldGetter` for `*blocks.StreamBlockValue`, `*blocks.ListBlockValue`, and `*blocks.StructBlockValue`. Any model field of those types automatically resolves its block definition via `Get<FieldName>Block()` and returns `blocks.BlockField(block, opts...)`.
- **Direct `Block` field hooks**: `cfg.Init` registers `attrs.HookFormFieldForType` and `attrs.DefaultForType` hooks via `goldcrest` for struct fields whose type implements `blocks.Block` and expose a `Get<FieldName>Def() Block` method.
- **Static assets and templates**: Registers embedded `blocks/css/blocks.css`, `blocks/js/index.js`, and template configuration (`AppName: "blocks"`, base template `"blocks/base.tmpl"`).
- **Telepath widget adapters**: Package `init()` registers Telepath adapters for `widgets.Widget` (`checkbox`, `radio`, `select`, `file`, and default widgets) so nested block editors can instantiate widgets dynamically in the browser.
- **Template functions**: Registers `render_block` and `block_value` in `tpl.Funcs` (with a request-aware `render_block` variant in `tpl.RequestFuncs`).

See [Apps](../apps.md) and [Static Files](../staticfiles.md) for general app and asset configuration.

---

## Block Interface and Core Types

Every block type implements `blocks.Block`:

```go
type Block interface {
	Name() string
	SetName(name string)
	SetLabel(label any)
	SetHelpText(helpText any)
	SetDefault(def interface{})
	Label(ctx context.Context) string
	HelpText(ctx context.Context) string
	Field() fields.Field
	SetField(field fields.Field)
	SetReadOnly(readonly bool)
	ValueAtPath(bound BoundBlockValue, parts []string) (interface{}, error)
	ValueFromDB(value json.RawMessage) (interface{}, error)
	RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, context ctx.Context) error
	Render(ctx context.Context, w io.Writer, value interface{}, context ctx.Context) error
	HasChanged(initial, data interface{}) bool
	GetDefault() interface{}
	telepath.AdapterGetter
	media.MediaDefiner
	widgets.FormValuer
	forms.Validator
	forms.Cleaner
}

type SaveableBlock interface {
	Block
	Save(ctx context.Context, value interface{}) (interface{}, error)
}
```

- `BaseBlock` implements the shared behavior for leaf blocks (`NewBaseBlock(opts ...OptFunc[*BaseBlock]) *BaseBlock`), storing `Name_`, `Template`, `FormField`, `Validators`, `Default`, `DataType`, `ValueFromDBFunc`, `LabelFunc`, and `HelpFunc`.
- `SaveableBlock` allows blocks (such as file or nested container blocks) to run field-level save logic before values are persisted.

---

## Field Blocks

`FieldBlock` embeds `*BaseBlock` and renders Telepath-backed form inputs for scalar and custom form fields.

```go
type FieldBlock struct {
	*BaseBlock
}

func NewFieldBlock(opts ...func(*FieldBlock)) *FieldBlock
func FormFieldBlock(field fields.Field, datatype any, opts ...func(*FieldBlock)) *FieldBlock
func CharBlock(opts ...func(*FieldBlock)) *FieldBlock
func BooleanBlock(opts ...func(*FieldBlock)) *FieldBlock
func NumberBlock(opts ...func(*FieldBlock)) *FieldBlock
func TextBlock(opts ...func(*FieldBlock)) *FieldBlock
func FileBlock(engine string, opts ...func(*FieldBlock)) *FieldBlock
func ImageBlock(engine string, opts ...func(*FieldBlock)) *FieldBlock
func EmailBlock(opts ...func(*FieldBlock)) *FieldBlock
func PasswordBlock(opts ...func(*FieldBlock)) *FieldBlock
func DateBlock(opts ...func(*FieldBlock)) *FieldBlock
func DateTimeBlock(opts ...func(*FieldBlock)) *FieldBlock
```

Built-in `FieldBlock` constructors wrap standard [form fields](../forms/fields.md) and [widgets](../forms/widgets.md):

| Constructor | Underlying Field / Widget | `DataType` |
| --- | --- | --- |
| `CharBlock` | `fields.CharField()` | `string` |
| `TextBlock` | `fields.CharField(fields.Widget(widgets.NewTextarea(nil)))` | `string` |
| `BooleanBlock` | `fields.BooleanField()` | `bool` |
| `NumberBlock` | `fields.NumberField[int]()` | `int` |
| `EmailBlock` | `fields.EmailField()` | `*mail.Address` |
| `PasswordBlock` | `fields.CharField(fields.Widget(widgets.NewPasswordInput(nil)))` | `string` |
| `DateBlock` | `fields.DateField(widgets.DateWidgetTypeDate)` | `time.Time` |
| `DateTimeBlock` | `fields.DateField(widgets.DateWidgetTypeDateTime)` | `time.Time` |
| `FileBlock(engine)` | `filefield.FileField(engine)` with `filewidget.NewFileInput` | `*mediafiles.SimpleStoredObject` |
| `ImageBlock(engine)` | `filefield.ImageFileField(engine)` with `filewidget.NewImageFileInput` | `*mediafiles.SimpleStoredObject` |
| `FormFieldBlock(field, datatype)` | Custom `fields.Field` | Caller-supplied `datatype` |

For example, you can wrap any third-party or contrib form field—such as the rich text editor field from `contrib/editor`—using `FormFieldBlock`:

```go
textBlock := blocks.FormFieldBlock(
	editor.EditorJSField(nil),
	&editor.EditorJSBlockData{},
	blocks.WithLabel[*blocks.FieldBlock](trans.S("Body")),
)
```

---

## Container Blocks

Container blocks compose multiple child blocks into structured objects, homogeneous lists, or heterogeneous streams.

### StructBlock

`StructBlock` groups named child blocks in an ordered map (`*orderedmap.OrderedMap[string, Block]`), storing values as `map[string]interface{}` inside a `*StructBlockValue`.

```go
type StructBlock struct {
	*BaseBlock
	Fields *orderedmap.OrderedMap[string, Block]
}

func NewStructBlock(opts ...func(*StructBlock)) *StructBlock
func (m *StructBlock) AddField(name string, block Block)
```

You can populate fields either with `AddField` or with `WithBlockField`:

```go
headingBlock := blocks.NewStructBlock(
	blocks.WithLabel[*blocks.StructBlock](trans.S("Heading")),
	blocks.WithTemplate[*blocks.StructBlock]("blog/blocks/heading.tmpl"),
	blocks.WithBlockField[*blocks.StructBlock]("title", blocks.CharBlock(
		blocks.WithLabel[*blocks.FieldBlock](trans.S("Title")),
	)),
	blocks.WithBlockField[*blocks.StructBlock]("subtitle", blocks.CharBlock(
		blocks.WithLabel[*blocks.FieldBlock](trans.S("Subtitle")),
	)),
)
```

Without a custom template, `StructBlock.Render` outputs a `<dl>` with each child block's label in `<dt>` and its rendered value in `<dd>`.

### ListBlock

`ListBlock` repeats a single child `Block` with ordering and optional minimum/maximum item constraints.

```go
type ListBlockData struct {
	ID    uuid.UUID   `json:"id"`
	Order int         `json:"order"`
	Data  interface{} `json:"data"`
}

type ListBlock struct {
	*BaseBlock
	Child Block
	Min   int
	Max   int
}

func NewListBlock(block Block, minMax ...int) *ListBlock
```

`NewListBlock` accepts zero, one (`min`), or two (`min, max`) bounds (defaults are `-1` for unbounded):

```go
tagsBlock := blocks.NewListBlock(
	blocks.CharBlock(blocks.WithLabel[*blocks.FieldBlock](trans.S("Tag"))),
	1, // minimum 1 item
	5, // maximum 5 items
)
```

Without a custom template, `ListBlock.Render` renders each item in order via `Child.Render`.

### StreamBlock

`StreamBlock` holds an ordered sequence of typed blocks chosen from named `Children` definitions (`*orderedmap.OrderedMap[string, Block]`), with optional `Min` and `Max` item counts.

```go
type StreamBlockData struct {
	ID    uuid.UUID   `json:"id"`
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
	Order int         `json:"-"`
}

type StreamBlock struct {
	*BaseBlock
	Children *orderedmap.OrderedMap[string, Block]
	Min      int
	Max      int
}

func NewStreamBlock(opts ...func(*StreamBlock)) *StreamBlock
func (l *StreamBlock) AddField(name string, block Block)
```

Example:

```go
contentBlock := blocks.NewStreamBlock(
	blocks.WithLabel[*blocks.StreamBlock](trans.S("Content")),
	blocks.WithMin[*blocks.StreamBlock](0),
	blocks.WithMax[*blocks.StreamBlock](10),
	blocks.WithBlockField[*blocks.StreamBlock]("heading", headingBlock),
	blocks.WithBlockField[*blocks.StreamBlock]("paragraph", blocks.TextBlock(
		blocks.WithLabel[*blocks.FieldBlock](trans.S("Paragraph")),
	)),
)
```

Without a custom template, `StreamBlock.Render` iterates through the items in `StreamBlockValue.V` and delegates rendering to each item's matching child block (skipping unknown types removed in later schema edits).

---

## Option Functions

Block constructors accept generic option functions (`OptFunc[T]` or `func(T)`):

```go
type OptFunc[T any] func(T)

func WithValidators[T any](validators ...func(context.Context, interface{}) error) OptFunc[T]
func WithLabel[T Block](label any) OptFunc[T]
func WithHelpText[T Block](text any) OptFunc[T]
func WithReadOnly[T Block](readonly bool) OptFunc[T]
func WithDefault[T Block](def interface{}) OptFunc[T]
func WithTemplate[T Block](template string) OptFunc[T]
func WithMin[T Block](min int) OptFunc[T]
func WithMax[T Block](max int) OptFunc[T]
func WithBlockField[T Block](name string, block Block) OptFunc[T]
```

- `WithLabel` and `WithHelpText` accept a `string` or a `func(context.Context) string` (such as `trans.S(...)`).
- `WithDefault` accepts either a static value or a zero-arg constructor function.
- `WithTemplate` sets the `Template` field used during `Render`.
- `WithMin` and `WithMax` set `Min` and `Max` on `ListBlock` or `StreamBlock`.
- `WithBlockField` calls `AddField(name, block)` on `StructBlock` or `StreamBlock`.

---

## Value Types and Model Integration

Block values are wrapped in `BoundValue[T]`, which implements `BoundBlockValue`, `RenderableValue`, `attrs.Binder`, `sql.Scanner`, `driver.Valuer`, `json.Marshaler`, and `json.Unmarshaler`:

```go
type BoundBlockValue interface {
	Block() Block
	Data() interface{}
}

type RenderableValue interface {
	Render(c context.Context, w io.Writer, ctxt ctx.Context) error
}

type BoundValue[T any] struct {
	BlockObject Block           `json:"-"`
	V           T               `json:"-"`
	_rawData    json.RawMessage `json:"-"`
}

type (
	FieldBlockValue  = BoundValue[interface{}]
	ListBlockValue   = BoundValue[[]*ListBlockData]
	StreamBlockValue = BoundValue[[]*StreamBlockData]
	StructBlockValue = BoundValue[map[string]interface{}]
)
```

When a model includes a `*blocks.StreamBlockValue`, `*blocks.ListBlockValue`, or `*blocks.StructBlockValue` field, implement a `Get<FieldName>Block()` method on the model returning the block definition (either `blocks.Block` or a concrete block pointer such as `*blocks.StreamBlock` or `*blocks.ListBlock`):

```go
package blog

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Article struct {
	models.Model `table:"articles"`
	ID           int64
	Title        string
	Content      *blocks.StreamBlockValue
}

func (a *Article) GetContentBlock() *blocks.StreamBlock {
	return blocks.NewStreamBlock(
		blocks.WithBlockField[*blocks.StreamBlock]("heading", blocks.CharBlock(
			blocks.WithLabel[*blocks.FieldBlock](trans.S("Heading")),
		)),
		blocks.WithBlockField[*blocks.StreamBlock]("body", blocks.TextBlock(
			blocks.WithLabel[*blocks.FieldBlock](trans.S("Body")),
		)),
	)
}

func (a *Article) FieldDefs(ctx context.Context) attrs.Definitions {
	return a.Model.Define(ctx, a,
		attrs.NewField(a, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(a, "Title", nil),
		attrs.NewField(a, "Content", &attrs.Config{
			Null:  true,
			Blank: true,
			Label: trans.S("Content"),
		}),
	)
}
```

When the ORM scans a row from the database into `Article`, `BoundValue.Scan` stores the raw JSON until `BindToModel` runs; `BindToModel` calls `GetContentBlock()` and decodes the JSON via `Block.ValueFromDB(...)` into `a.Content.V`.

See [Attrs](../attrs/attrs.md) and the [Blog App Example](../examples/blog.md) for a full model and admin integration walkthrough.

---

## Forms, Widgets, and Validation Errors

`blocks` integrates with `src/forms` via `BlockFormField` and `BlockWidget`:

```go
func BlockField(block Block, opts ...func(fields.Field)) *BlockFormField
func NewBlockWidget(blockDef Block) *BlockWidget
```

- `BlockField` wraps a `Block` as a `forms.Field` and defaults its widget to `NewBlockWidget(block)`. It also propagates `FormLabel` and `FormHelpText` to the block.
- `BlockWidget` implements `widgets.Widget`, rendering the Telepath block container and including `blocks/css/blocks.css` and `blocks/js/index.js` (plus any child block `Media()`) in `Media()`.

During `Render` and `RenderForm`, blocks populate a `*BlockContext`:

```go
type BlockContext struct {
	Request_  *http.Request
	BlockDef  Block
	ID        string
	Name      string
	BlockHTML template.HTML
	Value     interface{}
	Errors    []error
	Attrs     map[string]string
	Context   ctx.Context
}

func NewBlockContext(b Block, context ctx.Context) *BlockContext
```

Structured validation and deserialization errors are collected in `*BaseBlockValidationError[T]`:

```go
type BaseBlockValidationError[T comparable] struct {
	Errors         map[T][]error
	NonBlockErrors []error
}

func NewBlockErrors[T comparable](errors ...error) *BaseBlockValidationError[T]
```

`StructBlock` uses `BaseBlockValidationError[string]` (keyed by child field name), while `ListBlock` and `StreamBlock` use `BaseBlockValidationError[int]` (keyed by item index, with min/max count violations recorded in `NonBlockErrors`).

---

## Template Rendering

`blocks` registers two template functions when `blocks.NewAppConfig()` is created:

1. **`render_block`**: Renders any `RenderableValue` (such as a `*blocks.StreamBlockValue`, `*blocks.ListBlockValue`, `*blocks.StructBlockValue`, or `*blocks.FieldBlockValue`) using the active template context:

   ```html
   {{ render_block . .Data.page.Content }}
   ```

   You can also iterate over items in a `StreamBlockValue` and render each item's `Data` (which is itself a `BoundBlockValue` / `RenderableValue`):

   ```html
   {{ $ctx := . }}
   {{ range $item := .Data.page.Content.V }}
       <div class="block block--{{ $item.Type }}">
           {{ render_block $ctx $item.Data }}
       </div>
   {{ end }}
   ```

2. **`block_value`**: Resolves a dot-separated path on a `BoundBlockValue` or `*BlockContext` inside custom block templates (registered with `WithTemplate`):

   ```html
   {{ define "heading" }}
       <h1>{{ block_value . "title" }}</h1>
       <h2>{{ block_value . "subtitle" }}</h2>
   {{ end }}
   ```

   Path traversal delegates to `Block.ValueAtPath(bound, parts)`:
   - `StructBlock` traverses by field name (`"author.name"`).
   - `ListBlock` and `StreamBlock` traverse by zero-based numeric index (`"0.title"`).
   - `BaseBlock` / `FieldBlock` returns `bound.Data()`.

See [Rendering](../rendering.md) for template registration and context details.

---

## Notes and Caveats

- **Go `init()` vs `NewAppConfig()`**: `dbtype.Add` for `*StreamBlockValue`/`*ListBlockValue` and `attrs.RegisterFormFieldGetter` run in package `init()`, while static assets, templates, Telepath/template functions (`render_block`, `block_value`), and `Block`-interface hooks are only registered when `blocks.NewAppConfig` is added to `django.Apps(...)`.
- **`WithTemplate`, `WithMin`, `WithMax`, `WithBlockField` panics**: These option helpers use reflection/method lookup at construction time and panic if applied to a block type without a matching `Template`, `Min`, or `Max` struct field or `AddField(string, Block)` method.
- **`block_value` panics on unbound values**: Passing a value to `block_value` that is neither a `BoundBlockValue` nor a `*BlockContext` whose `Value` is a `BoundBlockValue` panics.
- **Custom template rendering for `FieldBlock`**: `BaseBlock.Render` handles `*FieldBlockValue` by calling `b.Template` if set, otherwise delegating to `RenderableValue.Render`, `HTML() template.HTML`, or HTML-escaping `fmt.Sprintf("%v", val)`.
