# Form Fields

Fields (`github.com/Nigel2392/go-django/src/forms/fields`) are the core input units of a form. They wrap a [`widgets.Widget`](./widgets.md), convert values between Go types and HTML form values, clean input, and run validators.

---

## The `Field` Interface

`fields.Field` is an alias for `internal/forms.Field`:

```go
type Field interface {
	FormValueConverter
	Attrs() map[string]string
	SetAttrs(attrs map[string]string)
	Hide(hidden bool)

	SetName(name string)
	SetLabel(label func(ctx context.Context) string)
	SetHelpText(helpText func(ctx context.Context) string)
	SetValidators(validators ...func(interface{}) error)
	SetDefault(defaultValue func() interface{})
	SetWidget(widget Widget)

	Name() string
	Label(ctx context.Context) string
	HelpText(ctx context.Context) string
	Validate(ctx context.Context, value interface{}) []error
	Widget() Widget
	Default() interface{}
	HasChanged(initial, data interface{}) bool

	Clean(ctx context.Context, value interface{}) (interface{}, error)
	Required() bool
	SetRequired(b bool)
	ReadOnly() bool
	SetReadOnly(b bool)
	IsEmpty(value interface{}) bool
}
```

- `ValueToForm` and `ValueToGo` (from `FormValueConverter`) convert between Go values and form/string representations.
- `Attrs` / `SetAttrs` manage HTML attributes rendered on the field's widget.
- `Hide` hides or unhides the widget.
- `SetLabel` and `SetHelpText` accept `func(ctx context.Context) string` so labels and help text support request-scoped [translations](../translations.md).
- `Clean` transforms raw converted values before `Validate` checks validators, widget validation, and `Required()`.
- `HasChanged` compares initial and submitted values (supporting `driver.Valuer` and convertible types).

### Saveable Fields

Fields that persist or transform data when a form is validated/saved can implement `SaveableField` or `SaveableCtxField`:

```go
type SaveableField interface {
	Field
	Save(value interface{}) (interface{}, error)
}

type SaveableCtxField interface {
	Field
	Save(ctx context.Context, value interface{}) (interface{}, error)
}
```

Note that `forms.IsValid` invokes `Save` on fields during `fullClean`, and `BaseForm.Save()` invokes `SaveableField.Save` again (see the [caveat in Forms](./readme.md#caveats)).

---

## Standard Field Constructors

Fields are normally constructed using functional option parameters (`opts ...func(Field)`).

### `NewField` and `CharField`

```go
func NewField(opts ...func(Field)) *BaseField
func CharField(opts ...func(Field)) Field
```

`NewField` creates a `*BaseField` (defaulting to a `widgets.NewTextInput(nil)` widget and English title-cased field name fallback for labels). `CharField` wraps `NewField(opts...)` and returns `Field`.

### `EmailField`

```go
func EmailField(opts ...func(Field)) Field
```

Defaults to `widgets.NewEmailInput(nil)`, which parses and validates email addresses using `net/mail.ParseAddress` and converts values to `*mail.Address`.

### `BooleanField`

```go
func BooleanField(opts ...func(Field)) Field
```

Defaults to `widgets.NewBooleanInput(nil)` (checkbox input). Avoid setting `Required(true)` on optional boolean checkboxes because unchecked HTML checkboxes are omitted from form submissions and evaluate as `false`/empty.

### `DateField`

```go
func DateField(typ widgets.DateWidgetType, opts ...func(Field)) Field
```

Defaults to `widgets.NewDateInput(nil, typ)`, where `typ` is either `widgets.DateWidgetTypeDate` (`"date"`) or `widgets.DateWidgetTypeDateTime` (`"datetime-local"`).

### `NumberField`

```go
func NumberField[T widgets.NumberType](opts ...func(Field)) Field
```

Generic numeric input for any integer or floating-point constraint (`widgets.NumberType`), defaulting to `widgets.NewNumberInput[T](nil)` with a custom `IsEmptyFunc` treating `nil` and `""` (not `0`) as empty.

### `DecimalField`

```go
func DecimalField(opts ...func(Field)) Field
```

Defaults to `widgets.NewDecimalInput(nil, 0)` (2 decimal places step, parsing to `github.com/shopspring/decimal.Decimal`).

### `SQLNullField` (`NullableSQLField`)

```go
type NullableSQLField[SQLType any] struct {
	*BaseField
}

func SQLNullField[SQLType any](opts ...func(Field)) *NullableSQLField[SQLType]
```

Wraps `database/sql` nullable types (`sql.NullBool`, `sql.NullInt16`, `sql.NullInt32`, `sql.NullInt64`, `sql.NullFloat64`, `sql.NullString`, `sql.NullTime`), unwrapping them in `ValueToForm`, converting submitted strings via the appropriate default widget in `ValueToGo`, and turning `nil` into the corresponding zero `sql.Null*` struct.

**Caveat for `byte` (`sql.NullByte`)**: `NullableSQLField.ValueToGo` passes through an already-constructed `sql.NullByte`, but has no `byte` type case when converting raw `byte` or `nil` values (returning `errs.ErrInvalidType`), and `Widget()` falls back to a generic text input.

### `MultiField`

```go
type MultiField struct {
	*BaseField
	Fields []Field
}

func NewMultiField(fields []Field, opts ...func(Field)) *MultiField
```

Groups multiple subfields into a single field backed by `map[string]interface{}` keyed by each child field's `Name()`. `MultiField` propagates `SetRequired`, `SetReadOnly`, `Hide`, `Clean`, `Validate`, and `HasChanged` across all child fields and defaults to `widgets.NewMultiWidget`.

### `MarshallerField` and `JSONField`

```go
type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

func EncFunc[T Encoder](f func(io.Writer) T) func(io.Writer) Encoder
func DecFunc[T Decoder](f func(io.Reader) T) func(io.Reader) Decoder

func MarshallerField[T any](
	encoder func(io.Writer) Encoder,
	decoder func(io.Reader) Decoder,
	opts ...func(Field),
) *MarshallerFormField[T]

func JSONField[T any](opts ...func(Field)) *JSONFormField[T]
```

- `MarshallerField` encodes Go values to strings via `encoder` in `ValueToForm` and decodes submitted strings into `T` via `decoder` in `ValueToGo`. Use `EncFunc` / `DecFunc` to adapt typed factory functions such as `json.NewEncoder` and `json.NewDecoder`.
- `JSONField` wraps `MarshallerField[T]` with `json.NewEncoder`/`json.NewDecoder`, defaults to `widgets.NewTextarea(nil)`, and translates `encoding/json` errors into user-friendly validation messages.

### `Protect` (`ProtectedFormField`)

```go
type ProtectedFormField struct {
	Field
	ErrorMessage func(err error) error
}

func Protect(w Field, errFn func(err error) error) *ProtectedFormField
```

Wraps any `Field` and intercepts errors returned by `ValueToGo`, `Clean`, and `Validate`, passing them through `errFn` (for example, to avoid leaking sensitive details on credential fields). Note that in `ProtectedFormField.Validate`, `errFn` is called twice (`merged := pw.ErrorMessage(...)` and `pw.ErrorMessage(merged)`).

---

## Field Option Functions

All option functions live in `github.com/Nigel2392/go-django/src/forms/fields` (`src/forms/fields/funcs.go`):

- `Label(label any) func(Field)`: Accepts a `string`, `func(context.Context) string`, or `trans.Translatable` (resolved via `trans.GetTextFunc`).
- `HelpText(helpText any) func(Field)`: Same translatable text types as `Label`.
- `Name(name string) func(Field)`: Sets the field name.
- `Required(b bool) func(Field)`: Adds HTML `required=""` and sets `Required` to `true` when `b` is `true`.
- `ReadOnly(b bool) func(Field)`: Adds HTML `readonly=""` and sets `ReadOnly` to `true` when `b` is `true`.
- `Placeholder(placeholder string) func(Field)`: Sets the HTML `placeholder` attribute.
- `Attributes(attrs map[string]string) func(Field)`: Merges extra HTML attributes onto the field.
- `Default(value interface{}) func(Field)`: Sets a static default value or a `func() interface{}` generator.
- `Regex(regex string) func(Field)`: Compiles `regex` and validates `fmt.Sprintf("%v", value)` against it.
- `MinLength(min int) func(Field)`: Sets HTML `minlength` and validates string length (`len(fmt.Sprintf("%v", value)) >= min`).
- `MaxLength(max int) func(Field)`: Sets HTML `maxlength` and validates string length (`len(fmt.Sprintf("%v", value)) <= max`).
- `MinValue(min int) func(Field)`: Sets HTML `min` and validates integer lower bound.
- `MaxValue(max int) func(Field)`: Sets HTML `max` and validates integer upper bound.
- `Widget(w widgets.Widget) func(Field)`: Overrides the default widget.
- `Hide(b bool) func(Field)`: Hides or shows the field's widget.
- `Validators(validators ...func(interface{}) error) func(Field)`: Appends custom value validators.

### Option Caveats

- **`Required(false)` and `ReadOnly(false)` are no-ops**: Both option helpers check `if b` / `if !b` and do nothing when `false` is passed. They will not clear an already-set `Required` or `ReadOnly` flag or remove `"required"`/`"readonly"` attributes; call `field.SetRequired(false)` or `field.SetReadOnly(false)` directly if unsetting.
- **`MinLength` / `MaxLength` vs. `MinValue` / `MaxValue`**: `MinLength` and `MaxLength` always measure the string length of `fmt.Sprintf("%v", value)`, even on numeric fields. Use `MinValue` and `MaxValue` for numeric bounds.
- **`MinValue` and `MaxValue` only support integers**: Both convert the value with `strconv.Atoi(fmt.Sprintf("%v", value))`, so they fail with a conversion error on floating-point or decimal strings such as `"3.14"`.

### Field Configuration Example

```go
ageField := fields.NumberField[int](
	fields.Name("age"),
	fields.Label("Age"),
	fields.HelpText("Must be between 18 and 120"),
	fields.Placeholder("18"),
	fields.Default(18),
	fields.Required(true),
	fields.MinValue(18),
	fields.MaxValue(120),
	fields.Attributes(map[string]string{"class": "number-input"}),
)
```

---

## `fields/filefield` Subpackage

Import path: `github.com/Nigel2392/go-django/src/forms/fields/filefield`.

The `filefield` subpackage provides file and image upload fields integrated with `mediafiles.Backend` storage engines:

```go
type FileStorageField struct {
	*fields.BaseField
	IsImageField   bool
	DisallowClear  bool
	StorageBackend mediafiles.Backend
	StorageEngine  string
	Extensions     []string
	UploadTo       func(fileObject *widgets.FileObject) string
	Validators     []func(filename string, file io.Reader) error
}

func FileField(engine string, opts ...func(fields.Field)) *FileStorageField
func ImageFileField(engine string, opts ...func(fields.Field)) *FileStorageField
func FileFieldWithExtensions(allowedExtensions []string) func(fields.Field)
```

- `FileField` and `ImageFileField` default to [`filewidget.NewFileInput`](./widgets.md#widgetsfilewidget-subpackage) and `filewidget.NewImageFileInput` respectively.
- `Save(value interface{}) (interface{}, error)` takes a `*widgets.FileObject`, resolves `StorageBackend` (or `mediafiles.RetrieveBackend(StorageEngine)`), saves the uploaded buffer under `UploadTo(fileObject)` (defaulting to `images/<name>` for image fields), and returns `mediafiles.Open(path)`.

### `mediafiles.SimpleStoredObject` Auto-Registration

Importing `filefield` registers an `attrs.RegisterFormFieldGetter` for `mediafiles.SimpleStoredObject{}` in its `init()` function. Any `attrs` model field of type `mediafiles.SimpleStoredObject` automatically produces a `*FileStorageField` configured from `attrs.AttrUploadToKey` (`string`, `func() string`, or `func(*widgets.FileObject) string`).

### `FileFieldWithExtensions` Caveat

`FileField` and `ImageFileField` pass `opts` directly to `fields.NewField(opts...)`, which executes each option against `*fields.BaseField` rather than `*FileStorageField`. Passing `filefield.FileFieldWithExtensions(...)` inside `FileField(...)` or `ImageFileField(...)` panics with `"FileFieldWithExtensions requires a FileStorageField, got *fields.BaseField"`.

Set `Extensions` on the returned field instead, or invoke `FileFieldWithExtensions` after creation:

```go
avatar := filefield.ImageFileField("default",
	fields.Name("avatar"),
	fields.Label("Avatar"),
)
filefield.FileFieldWithExtensions([]string{".png", ".jpg"})(avatar)
```
