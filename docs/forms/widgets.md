# Widgets Package Documentation

The `widgets` package (`github.com/Nigel2392/go-django/src/forms/widgets`) provides HTML input components for rendering form fields, extracting submitted data from HTTP requests, converting values between HTML and Go types, and declaring required CSS/JS [`media.Media`](./media.md).

---

## The `Widget` Interface

Every widget implements `widgets.Widget` (aliased from `internal/forms.Widget`):

```go
type Widget interface {
	IsHidden() bool
	Hide(hidden bool)
	FormType() string
	Field() Field
	BindField(field Field)
	SetAttrs(attrs map[string]string)
	IdForLabel(id string) string
	GetContextData(ctx context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context
	RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string, context ctx.Context) error
	Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error
	Validate(ctx context.Context, value interface{}) []error

	FormValuer
	media.MediaDefiner
}
```

`FormValuer` comprises:

- `ValueToGo(value interface{}) (interface{}, error)`
- `ValueToForm(value interface{}) interface{}`
- `ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool`
- `ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error)`

Widgets can also implement `BinderWidget` or `BoundFieldBinderWidget` to customize how `BoundField` instances are constructed in [`forms.NewBoundFormField`](./readme.md#boundfield-and-boundform).

---

## Base Widget and Standard Widgets

### `BaseWidget`

`NewBaseWidget` builds the base widget implementation shared by most HTML inputs:

```go
func NewBaseWidget(type_ string, templateName string, attrs map[string]string) *BaseWidget
```

If `templateName` is `""`, it defaults to `"forms/widgets/input.html"`. If `type_` is `""`, it defaults to `"text"`.

### Predefined Input Widgets

In `github.com/Nigel2392/go-django/src/forms/widgets`:

- `NewTextInput(attrs map[string]string) Widget`: Renders `<input type="text">` via `"forms/widgets/text.html"`.
- `NewTextarea(attrs map[string]string) Widget`: Renders `<textarea>` via `"forms/widgets/textarea.html"`.
- `NewEmailInput(attrs map[string]string) Widget`: Renders `<input type="email">` (`*EmailInput`), parsing submitted strings via `mail.ParseAddress` into `*mail.Address`.
- `NewPasswordInput(attrs map[string]string) Widget`: Renders `<input type="password">`.
- `NewHiddenInput(attrs map[string]string) Widget`: Renders `<input type="hidden">`.
- `NewNumberInput[T NumberType](attrs map[string]string) Widget`: Renders `<input type="number">` (`*NumberWidget[T]`), parsing strings to integers, unsigned integers, or floats.
- `NewDecimalInput(attrs map[string]string, decimalPlaces int) Widget`: Renders `<input type="number">` (`*DecimalWidget`) converting strings to `shopspring/decimal.Decimal`. When `decimalPlaces == 0`, it defaults `DecimalPlaces` to `2` and sets `attrs["step"]`.
- `NewBooleanInput(attrs map[string]string) Widget`: Renders a checkbox (`*BooleanWidget`) treating `"true"`, `"on"`, and `"1"` (case-insensitive) as `true`.
- `NewDateInput(attrs map[string]string, t DateWidgetType) Widget`: Renders date/time inputs (`*DateWidget`) for `DateWidgetTypeDate` (`"date"`, `"2006-01-02"`) or `DateWidgetTypeDateTime` (`"datetime-local"`, `"2006-01-02T15:04"` / `"2006-01-02T15:04:05"`).
- `NewMultiWidget(attrs map[string]string) *MultiWidget`: Composes multiple named sub-widgets rendered under `<name>__<subname>` and backed by `map[string]interface{}`. Use `AddWidget(name string, widget Widget)` to register sub-widgets.

### `DecimalWidget` and `MultiWidget` Caveats

- **`DecimalWidget` formatting and `float32` panic**: `DecimalWidget.ValueToForm` formats values using `.StringFixed(2)` regardless of the configured `DecimalPlaces` field. In addition, both `ValueToGo` and `ValueToForm` use `case float32, float64: ... val.(float64)`, which panics with an interface conversion error if given a `float32`.
- **`MultiWidget` uninitialized map and ignored `attrs`**: `NewMultiWidget(attrs)` returns `&MultiWidget{}`, ignoring `attrs` and leaving `Widgets` as `nil`. Calling any widget method (`Media`, `Validate`, `ValueFromDataDict`, `GetContextData`, `Render`, etc.) before adding at least one widget via `AddWidget` panics on `b.Widgets.Front()`.

### Basic Rendering Example

```go
textWidget := widgets.NewTextInput(map[string]string{
	"placeholder": "Enter your text here",
})

passwordWidget := widgets.NewPasswordInput(map[string]string{
	"class": "secure-input",
})

widgetCtx := passwordWidget.GetContextData(
	ctx, "id_password", "password", "", map[string]string{},
)
err := passwordWidget.RenderWithErrors(
	ctx,
	writer,
	"id_password",
	"password",
	"",
	[]error{errors.New("Invalid password")},
	map[string]string{},
	widgetCtx,
)
if err != nil {
	log.Fatal(err)
}
```

---

## Supporting Types and Templ Helpers

### `FileObject`, `Option`, `FormOption`, and `WrappedOption`

```go
type FileObject struct {
	Name string
	File *bytes.Buffer
}

type FormOption struct {
	OptLabel any
	OptValue string
}

func NewOption(name, label, value string) Option

type WrappedOption struct {
	Option
	Selected bool
}

func WrapOptions(ctx context.Context, options []Option, selectedValues []string) []WrappedOption
```

`FormOption.Label()` resolves translatable labels via `trans.GetText(context.Background(), o.OptLabel)`. `WrapOptions` marks each option whose `Value()` appears in `selectedValues` as `Selected: true`.

### Templ Components

Defined in `src/forms/widgets/widgets.templ`, these [Templ](https://templ.guide) helpers render field labels, error lists, and help text:

```go
templ ErrorListComponent(forElem string, errors []error, attrs ...templ.Attributes)
templ LabelComponent(forElem, label, labelFor string, attrs ...templ.Attributes)
templ HelpTextComponent(forElem, helpText string, attrs ...templ.Attributes)
```

---

## `widgets/options` Subpackage

Import path: `github.com/Nigel2392/go-django/src/forms/widgets/options`.

The `options` subpackage provides choice-based widgets backed by a context-aware `choices` callback:

```go
type OptionsWidget struct {
	*widgets.BaseWidget
	Choices      func(ctx context.Context) []widgets.Option
	IncludeBlank bool
	BlankLabel   string
}

func NewOptionWidget(type_ string, templateName string, attrs map[string]string, choices func(ctx context.Context) []widgets.Option, options ...func(*OptionsWidget)) *OptionsWidget
func NewCheckboxInput(attrs map[string]string, choices func(ctx context.Context) []widgets.Option, opts ...func(*OptionsWidget)) *OptionsWidget
func NewRadioInput(attrs map[string]string, choices func(ctx context.Context) []widgets.Option, opts ...func(*OptionsWidget)) *OptionsWidget
func NewSelectInput(attrs map[string]string, choices func(ctx context.Context) []widgets.Option, opts ...func(*OptionsWidget)) *OptionsWidget

func IncludeBlank(b bool) func(*OptionsWidget)
func BlankLabel(label string) func(*OptionsWidget)
```

- `IncludeBlank(true)` prepends a blank option in `<select>` widgets (defaulting `BlankLabel` to `"---------"`) and allows empty submissions during `Validate`.
- `Validate` checks submitted `string` or `[]string` values against the options returned by `Choices(ctx)`.

### Example

```go
choices := func(ctx context.Context) []widgets.Option {
	return []widgets.Option{
		widgets.NewOption("option1", "Label 1", "1"),
		widgets.NewOption("option2", "Label 2", "2"),
	}
}

selectWidget := options.NewSelectInput(
	map[string]string{"class": "form-select"},
	choices,
	options.IncludeBlank(true),
	options.BlankLabel("Choose one..."),
)
```

### Checkbox Single-Value Caveat

`OptionsWidget` embeds `*widgets.BaseWidget` and inherits `BaseWidget.ValueFromDataDict`, which calls `data.Get(name)` and returns only the first submitted `string` value. Likewise, `forms/widgets/checkbox.html` checks `eq ($ctx.Get "value") $choice.Value`. For multi-value selection returning `[]string`, use [`multiselect.NewMultiSelectInput`](#widgetsmultiselect-subpackage).

---

## `widgets/multiselect` Subpackage

Import path: `github.com/Nigel2392/go-django/src/forms/widgets/multiselect`.

```go
type MultiSelectWidget struct {
	*options.OptionsWidget
}

func NewMultiSelectInput(attrs map[string]string, choices func(ctx context.Context) []widgets.Option) widgets.Widget
```

`MultiSelectWidget` overrides `ValueFromDataDict` to return `data[name]` (`[]string`) instead of a single value and bundles `forms/css/multiple-select.css` and `forms/js/index.js` via `Media()`.

---

## `widgets/filewidget` Subpackage

Import path: `github.com/Nigel2392/go-django/src/forms/widgets/filewidget`.

`FileWidget` handles file and image uploads alongside optional existing-path and clear inputs (`<name>__path` and `<name>__clear`):

```go
type FileWidget struct {
	*widgets.BaseWidget
	IsImageField    bool
	DisallowClear   bool
	ForceShowPath   bool
	ForceShowClear  bool
	PathPlaceholder string
	Extensions      []string
	Validators      []func(filename string, file io.Reader) error
}

func NewFileInput(attrs map[string]string, allowedMimeTypes []string, validators ...func(filename string, file io.Reader) error) *FileWidget
func NewImageFileInput(attrs map[string]string, allowedMimeTypes []string, validators ...func(filename string, file io.Reader) error) *FileWidget
```

- **Clearing and retaining existing files**:
  - When `<name>__clear` is `"on"`, `"true"`, or `"1"`, `ValueFromDataDict` returns `(nil, nil)`.
  - When no new file is uploaded and `<name>__path` is non-empty, it returns `&widgets.FileObject{Name: pathVal[0], File: nil}`.
  - When a file is uploaded, it reads the file into a `*bytes.Buffer` and returns `&widgets.FileObject{Name: cleanedName, File: buf}`.
- `Extensions` validates uploaded filenames by extension (case-insensitive) during `Validate`, and builds the HTML `accept` attribute (`extensionsAttr`) from MIME types.
- `Media()` adds `forms/css/file-input.css`.

### Buffer-Drain and Double-Validate Caveat

In `ValueFromDataDict`, `FileWidget` copies the uploaded file into a single `*bytes.Buffer` and passes `buf` directly to each `validator(fileName, buf)` without cloning or resetting the buffer offset. Later, `FileWidget.Validate` invokes all `f.Validators` a **second** time against the same `fileObj.File` buffer.

If a validator reads bytes from `file` without restoring them (or if multiple validators / `filefield.FileStorageField.Save` read after it), the buffer will already be drained (`EOF`) on subsequent reads. Validators that inspect file contents should buffer and restore `file` or avoid consuming it destructively if downstream saving depends on `fileObj.File`.

---

## `widgets/chooser` Subpackage

Import path: `github.com/Nigel2392/go-django/src/forms/widgets/chooser`.

The `chooser` subpackage connects form selection widgets to registered [`contenttypes`](../contenttypes.md) models:

```go
type BaseChooserOptions struct {
	TargetObject  interface{}
	GetPrimaryKey func(ctx context.Context, i interface{}) interface{}
	Queryset      func(ctx context.Context) ([]interface{}, error)
}

func BaseChooserWidget(opts BaseChooserOptions, attrs map[string]string) *BaseChooser
func SelectWidget(allowBlank bool, blankLabel string, opts BaseChooserOptions, attrs map[string]string) *Select
```

- `BaseChooserWidget` looks up `contenttypes.DefinitionForObject(opts.TargetObject)` (panicking if unregistered) and validates single IDs or slices of IDs via `Instance` / `InstancesByIDs`.
- `SelectWidget` extends `BaseChooser` to render a `<select>` populated from `opts.Queryset(ctx)` (falling back to `forModelDefinition.Instances(ctx, 1000, 0)`), labeling each item via `InstanceLabel` and keying values via `opts.GetPrimaryKey`.

---

## Custom Widgets

Implement `widgets.Widget` directly or embed `*widgets.BaseWidget` to inherit attribute merging, ID generation, and template rendering:

```go
type MyCustomWidget struct {
	*widgets.BaseWidget
}

func NewMyCustomWidget(attrs map[string]string) widgets.Widget {
	return &MyCustomWidget{
		BaseWidget: widgets.NewBaseWidget("text", "myapp/widgets/custom.html", attrs),
	}
}
```
