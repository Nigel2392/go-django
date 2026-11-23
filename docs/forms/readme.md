# Forms Package Documentation

The `forms` package (`github.com/Nigel2392/go-django/src/forms`) handles collecting request data, cleaning and validating input, managing CSS/JS media assets, and rendering HTML forms and error messages.

## Table of Contents

- [Fields](./fields.md)
- [Widgets](./widgets.md)
- [Media](./media.md)
- [Model Forms](./modelforms.md)
- [Formsets](./formsets.md)

---

## Core Interfaces

### `Cleaner` and `Validator`

Fields and custom processors implement `Cleaner` and `Validator` to parse raw input and check validation rules:

```go
type Cleaner interface {
	Clean(ctx context.Context, value interface{}) (interface{}, error)
}

type Validator interface {
	Validate(ctx context.Context, value interface{}) []error
}
```

### `BoundField` and `BoundForm`

`BoundForm` and `BoundField` represent a form and its fields prepared for template rendering:

```go
type BoundField interface {
	ID() string
	Name() string
	Widget() Widget
	Hidden() bool
	Input() Field
	Label() template.HTML
	HelpText() template.HTML
	Field() template.HTML
	HTML() template.HTML
	Context() context.Context
	Attrs() map[string]string
	Value() interface{}
	Errors() []error
}

type BoundForm interface {
	AsP() template.HTML
	AsUL() template.HTML
	AsTable() template.HTML
	Media() media.Media
	Fields() []BoundField
	FieldMap() map[string]BoundField
	ErrorList() []error
	UnpackErrors() []FieldError
	Errors() *orderedmap.OrderedMap[string, []error]
}
```

`BaseForm.EditContext(key, ctx)` inserts `f.BoundForm()` into the template [`ctx.Context`](../context.md) so templates can iterate `Fields`, inspect `ErrorList` / `UnpackErrors`, or render `AsP` / `AsUL` / `AsTable`.

### `FormRenderer`

`FormRenderer` controls HTML output for whole forms and individual fields:

```go
type FormRenderer interface {
	RenderAsP(w io.Writer, ctx context.Context, form BoundForm) error
	RenderAsUL(w io.Writer, ctx context.Context, form BoundForm) error
	RenderAsTable(w io.Writer, ctx context.Context, form BoundForm) error

	RenderFieldLabel(w io.Writer, ctx context.Context, field BoundField, id string, name string) error
	RenderFieldHelpText(w io.Writer, ctx context.Context, field BoundField, id string, name string) error
	RenderFieldWidget(w io.Writer, ctx context.Context, field BoundField, id string, name string, value interface{}, attrs map[string]string, errors []error, widgetCtx ctx.Context) error
	RenderField(w io.Writer, ctx context.Context, field BoundField, id string, name string, value interface{}, errors []error, attrs map[string]string, widgetCtx ctx.Context) error
}
```

Fields can also override label or help text rendering individually by implementing `FieldLabelRenderer` or `FieldHelpTextRenderer`.

### `ErrorAdder`, `ErrorDefiner`, and `FieldError`

```go
type ErrorAdder interface {
	AddFormError(errorList ...error)
	AddError(name string, errorList ...error)
}

type ErrorDefiner interface {
	ErrorList() []error
	BoundErrors() *orderedmap.OrderedMap[string, []error]
}

type FieldError interface {
	Name() string
	Field() string
	Errors() []error
}
```

- `AddFormError` adds form-wide errors (stored on `ErrorList()` and mapped under `"__all__"` in `BoundErrors()`).
- `AddError` attaches errors to a named field (or falls back to `ErrorList()` if the field does not exist).
- `FieldError.Name()` returns the raw field name (or `"__all__"`), while `FieldError.Field()` returns the translated field label when available.

---

## `Minimum`, `Form`, and `SaveableForm`

### `Minimum`

`Minimum` is the baseline interface shared by standalone forms, [`modelforms`](./modelforms.md), and child forms inside [`formsets`](./formsets.md):

```go
type Minimum interface {
	ErrorDefiner
	WithDataDefiner
	FormFieldDefiner
	Prefix() string
	SetPrefix(prefix string)
	InitialData() map[string]any
	CleanedData() map[string]any
	Context() context.Context
	WithContext(ctx context.Context)
	AddFormError(errorList ...error)
	HasChanged() bool
}
```

### `Form`

The primary `Form` interface extends `Minimum`, `FullCleanMixin`, and `ErrorAdder`:

```go
type Form interface {
	Minimum
	FullCleanMixin
	ErrorAdder

	Media() media.Media
	SetInitial(initial map[string]interface{})
	FormValue(name string) interface{}
	Validators() []func(f Form, cleanedData map[string]interface{}) []error
	SetValidators(validators ...func(Form, map[string]interface{}) []error)
	Renderer() FormRenderer
	SetRenderer(renderer FormRenderer)

	Ordering([]string)
	FieldOrder() []string

	Field(name string) (Field, bool)
	Fields() []Field
	Widgets() []Widget
	AddField(name string, field Field)
	AddWidget(name string, widget Widget)
	DeleteField(name string) bool
	BoundForm() BoundForm
	BoundFields() *orderedmap.OrderedMap[string, BoundField]

	OnValid() signals.Signal[Form]
	OnInvalid() signals.Signal[Form]
	OnFinalize() signals.Signal[Form]

	WasCleaned() bool
	HasChanged() bool
	CleanedDataUnsafe() map[string]interface{}
}
```

- `FormValue(name)` resolves a field's display value in order: `Cleaned` -> `InvalidDefaults` -> `Defaults` -> `field.Default()`.
- `CleanedDataUnsafe()` returns the raw `Cleaned` map without triggering `CleanedData()`'s validation/error assertions.
- `OnValid()`, `OnInvalid()`, and `OnFinalize()` return [`go-signals`](https://github.com/Nigel2392/go-signals) `signals.Signal[Form]` channels emitted by `forms.IsValid`.

### `SaveableForm`

```go
type SaveableForm interface {
	Form
	Save() (map[string]interface{}, error)
}
```

`BaseForm` implements `SaveableForm`. Its `Save()` method checks that validation ran without errors, copies `Initial` values for read-only fields, calls `Save(value)` on any [`fields.SaveableField`](./fields.md#saveable-fields), and returns the resulting data map.

---

## Validation Functions and Mixins

`BaseForm` does **not** expose `FullClean()`, `Validate()`, or `IsValid()` methods. Instead, validation and data extraction run through package-level functions in `forms`:

```go
func IsValid[T any](ctx context.Context, f T) bool
func FullClean(ctx context.Context, f Form) (invalid, defaults, cleaned map[string]any)
func ValueFromDataDict[T any](
	ctx context.Context,
	form FormFieldDefiner,
	name string,
	data map[string][]string,
	files map[string][]filesystem.FileHeader,
) (T, bool, []error)
func HasErrors(f ErrorDefiner) bool
```

### How `forms.IsValid` Runs

1. If `f` implements `FormWrapper[T]`, `IsValid` validates unwrapped child forms first.
2. Any embedded mixins implementing `PrevalidatorMixin` run `Prevalidate(ctx, formObj, rawData, files)`.
3. `FullClean` (or `fullClean`) iterates over `FieldMap()`, skips read-only fields, extracts values via `Widget.ValueOmittedFromData` / `ValueFromDataDict`, checks `Required` + `IsEmpty`, converts via `Field.ValueToGo`, cleans via `Field.Clean`, validates via `Field.Validate`, runs any field `Save` method, and binds `(invalid, defaults, cleaned)`.
4. If no field errors occurred and the form implements `CleanableForm` (`Clean(ctx, cleaned) (map[string]any, []error)`), form-wide cleaning runs.
5. Form-level `Validators()` and embedded `ValidatorMixin` validators run.
6. Emits `OnValid()` (or clears `Cleaned` and emits `OnInvalid()`), followed by `OnFinalize()`.
7. If the form implements `IsValidDefiner`, calls `f.IsValid(ctx)`.

For custom form types that do not implement `Form`, `forms.IsValid` checks for a `CheckIsValid(context.Context, T) bool` method (used by [`formsets.BaseFormSet`](./formsets.md)).

### Validation Mixin Interfaces

```go
type PrevalidatorMixin interface {
	Prevalidate(ctx context.Context, root any, data url.Values, files map[string][]filesystem.FileHeader) []error
}

type ValidatorMixin interface {
	Validators() []func(ctx context.Context, root Form) []error
}

type FullCleanMixin interface {
	Widget(name string) (Widget, bool)
	PrefixName(fieldName string) string
	FieldMap() *orderedmap.OrderedMap[string, Field]
	BindCleanedData(invalid, defaults, cleaned map[string]interface{})
}

type IsValidDefiner interface {
	IsValid(ctx context.Context) bool
}

type CleanableForm interface {
	Clean(ctx context.Context, cleaned map[string]interface{}) (map[string]interface{}, []error)
}
```

---

## Initialization and Option Functions

Create and configure forms with `forms.NewBaseForm(ctx, opts...)` and `forms.Initialize(f, opts...)`:

```go
func NewBaseForm(ctx context.Context, opts ...func(Form)) *BaseForm
func Initialize[T Form](f T, initfuncs ...func(Form)) T
```

Available option functions:

- `WithData[T WithDataDefiner](data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) func(T)`: Binds explicit form values and uploaded files.
- `WithRequestData(method string, r *http.Request) func(Form)`: When `r.Method == method`, parses `r.ParseMultipartForm` (respecting `APPVAR_MAX_FORM_MEMORY`) and passes `r.Form` and uploaded files to `WithData`; otherwise binds empty values/files.
- `WithFields(fields ...Field) func(Form)`: Adds fields to the form using each field's `Name()`.
- `WithPrefix(prefix string) func(Form)`: Sets the form field prefix (`<prefix>-<name>`).
- `WithInitial(initial map[string]interface{}) func(Form)`: Sets initial data.
- `OnValid(funcs ...func(Form)) func(Form)`: Connects listeners to `f.OnValid()`.
- `OnInvalid(funcs ...func(Form)) func(Form)`: Connects listeners to `f.OnInvalid()`.
- `OnFinalize(funcs ...func(Form)) func(Form)`: Connects listeners to `f.OnFinalize()`.

---

## Template and Static Asset Hooks

`src/forms/renderer.go` embeds the default widget templates and static assets (`assets/templates/forms/widgets/*.html` and `assets/static/forms/...`). Before registering them on `tpl.Add` and `staticfiles.AddFS` during first template render, it runs two [`goldcrest`](https://github.com/Nigel2392/goldcrest) hooks:

```go
type (
	FormTemplateFSHook     func(fSys fs.FS, cnf *tpl.Config) fs.FS
	FormTemplateStaticHook func(fSys fs.FS) fs.FS
)

const (
	FormTemplateFSHookName     = "forms.TemplateFSHook"
	FormTemplateStaticHookName = "forms.TemplateStaticHook"
)
```

Register against these hook names in [`goldcrest`](../hooks.md) to override or wrap the widget template filesystem or static asset filesystem.

---

## Complete Example

### 1. Initialize and Validate a Form

```go
package views

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func ContactHandler(w http.ResponseWriter, r *http.Request) {
	form := forms.Initialize(
		forms.NewBaseForm(r.Context()),
		forms.WithRequestData(http.MethodPost, r),
		forms.WithFields(
			fields.EmailField(
				fields.Label("Email"),
				fields.HelpText("Enter your email"),
				fields.Name("email"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(250),
				fields.Widget(widgets.NewEmailInput(nil)),
			),
			fields.CharField(
				fields.Label("Name"),
				fields.HelpText("Enter your name"),
				fields.Name("name"),
				fields.Required(true),
				fields.Regex(`^[a-zA-Z]+$`),
				fields.MinLength(2),
				fields.MaxLength(50),
			),
		),
		forms.OnValid(func(f forms.Form) {
			data := f.CleanedData()
			fmt.Println("Valid submission:", data["email"], data["name"])
		}),
		forms.OnInvalid(func(f forms.Form) {
			fmt.Println("Invalid submission:", f.BoundErrors())
		}),
	)

	if r.Method == http.MethodPost && forms.IsValid(r.Context(), form) {
		validData := form.CleanedData()
		_ = validData
	}

	// Pass form.BoundForm() into template context under "Form"
	_ = form.BoundForm()
}
```

### 2. Render in an HTML Template

When `form.BoundForm()` is stored in template context under `"Form"`:

```html
<div class="form-media">
    {{ range $File := (.Get "Form").Media.JS }}
        {{ $File }}
    {{ end }}
    {{ range $File := (.Get "Form").Media.CSS }}
        {{ $File }}
    {{ end }}
</div>

<form method="post">
    <input type="hidden" name="csrf_token" value="{{ .CsrfToken }}">
    {{ $errorList := (.Get "Form").ErrorList }}
    {{ if (gt (len $errorList) 0) }}
        <ul class="error-list">
            {{ range $Error := $errorList }}
                <li class="error"><p>{{ $Error }}</p></li>
            {{ end }}
        </ul>
    {{ end }}

    {{ range $Field := (.Get "Form").Fields }}
        <div class="form-field-wrapper">
            <div class="form-field">
                {{ $Field.Label }}
                {{ if (gt (len $Field.Errors) 0) }}
                    <ul class="error-list">
                        {{ range $Error := $Field.Errors }}
                            <li class="error"><p>{{ $Field.Label }}: {{ $Error }}</p></li>
                        {{ end }}
                    </ul>
                {{ end }}

                {{ if (ne $Field.HelpText "") }}
                    <p class="help-text">{{ $Field.HelpText }}</p>
                {{ end }}

                {{ $Field.Field }}
            </div>
        </div>
    {{ end }}

    <input type="submit" value="Submit">
</form>
```

### 3. Using Model Forms

See [Model Forms](./modelforms.md) for binding models directly with `modelforms.NewBaseModelForm`:

```go
modelForm := modelforms.NewBaseModelForm(r.Context(), existingModel)
modelForm.Load()

forms.WithRequestData(http.MethodPost, r)(modelForm)
if r.Method == http.MethodPost && forms.IsValid(r.Context(), modelForm) {
	if _, err := modelForm.Save(); err != nil {
		// handle error
	}
	instance := modelForm.Instance()
	_ = instance
}
```

---

## Caveats

- **`CleanedData()` assertions**: `BaseForm.CleanedData()` panics (with HTTP status 500 via `except.Assert`) if called before `forms.IsValid` populates `Cleaned`, or if any field errors or form errors exist. Always check `forms.IsValid(ctx, form)` first, or use `CleanedDataUnsafe()` when inspecting partially cleaned state.
- **`WithData` is required before `forms.IsValid`**: `forms.IsValid` asserts that `rawData != nil`, so always bind request or `url.Values` data (even an empty `url.Values{}`) before validating.
- **`BaseForm.Save()` double-run on `SaveableField`**: During `forms.IsValid`, `fullClean` already invokes any `Save` method on fields (handling both `SaveableField` and `SaveableCtxField`). Calling `BaseForm.Save()` afterward invokes `Save(value)` a second time on any `fields.SaveableField`. For fields whose first `Save` transforms the value type (such as `filefield.FileStorageField` returning `mediafiles.StoredObject` instead of `*widgets.FileObject`), calling `BaseForm.Save()` afterward will fail type checks unless handled. Note also that `modelforms.BaseModelForm` overrides `Save()` and uses `ModelFieldSaver` / `AfterModelFieldSaver` instead.
