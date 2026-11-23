# Model Forms

The `modelforms` package (`github.com/Nigel2392/go-django/src/forms/modelforms`) bridges [`attrs.Definer`](../attrs/interfaces.md#definer) models and [`forms.Form`](./readme.md#minimum-form-and-saveableform), automatically generating form fields from model definitions, populating initial values from a model instance, and saving validated changes back to the model.

---

## `ModelForm` Interface and `BaseModelForm`

### `ModelForm[T]`

```go
type ModelForm[T any] interface {
	forms.Form
	Load()
	Save() (map[string]interface{}, error)
	WithContext(ctx context.Context)
	Context() context.Context
	SetFields(fields ...string)
	SetExclude(exclude ...string)
	SetOnLoad(fn func(model T, initialData map[string]interface{}))
	Instance() T
	SetInstance(model T)
}
```

### `BaseModelForm[T]` and `NewBaseModelForm`

```go
type BaseModelForm[T attrs.Definer] struct {
	*forms.BaseForm
	Model          T
	Definition     attrs.Definitions
	InstanceFields []attrs.Field
	OnLoad         func(model T, initialData map[string]interface{})

	Initial      func() T
	SaveInstance func(context.Context, T) error
	ModelFields  []string
	ModelExclude []string
	// ...
}

func NewBaseModelForm[T attrs.Definer](ctx context.Context, model T, opts ...func(forms.Form)) *BaseModelForm[T]
```

If `model` is a nil pointer or zero value, `NewBaseModelForm` allocates a new model instance with `attrs.NewObject[attrs.Definer](ctx, reflect.TypeOf(model))` before calling `SetInstance`.

---

## Custom Field Savers: `ModelFieldSaver` vs. `AfterModelFieldSaver`

Form fields added to a `ModelForm` can implement one of two interfaces to control how validated values are written back to the model:

```go
type ModelFieldSaver interface {
	SaveField(ctx context.Context, field attrs.Field, value interface{}) error
}

type AfterModelFieldSaver interface {
	SaveField(ctx context.Context, field attrs.Field, value interface{}, model attrs.Definer) error
}
```

- **`ModelFieldSaver`**: Runs **before** the model is persisted (inside both `Instance()` and the pre-save pass of `Save()`). If a form field does not implement either interface, `field.SetValue(value, true)` is called instead.
- **`AfterModelFieldSaver`**: Runs **after** the model itself is saved in `Save()` (once the model has a primary key assigned). Use this for many-to-many or reverse relation widgets whose join rows require the parent model's saved ID.

If any saver or `field.SetValue` returns an error inside `SaveFields`, the error is attached to the field via `f.AddError(fieldname, err)` and returned immediately.

---

## Lifecycle: Configuration, Loading, Validation, and Saving

### 1. Construct and Configure (Before `Load()`)

`NewBaseModelForm` calls `SetInstance` automatically, which extracts `attrs.Define(ctx, model)`, skips any fields where `field.FormField() == nil`, and records the remaining field names in `ModelFields`.

Before calling `Load()`, you can customize which fields are included or override specific fields:

- `SetFields(fields ...string)`: Restricts `ModelFields` to the named fields (panics on duplicate field names; names absent from `f.Definition` are skipped).
- `SetExclude(exclude ...string)`: Excludes the named model fields from loading and saving.
- `AddField(name string, field fields.Field)`: Registers a custom form field (such as an `AfterModelFieldSaver`) before `Load()`; `Load()` preserves any field already present in `BaseForm`.
- `SetOnLoad(fn func(model T, initialData map[string]interface{}))`: Registers a callback invoked during `Load()` before `SetInitial(initialData)`.
- `Initial`: Optional `func() T` returning the model instance used to read initial values during `Load()`.
- `SaveInstance`: Optional `func(context.Context, T) error` overriding how `Save()` persists the model.

### 2. `Load()`

`Load()` builds the form fields on the embedded `*forms.BaseForm` (ignoring `ModelExclude` and keeping any custom field already registered via `AddField`), populates `initialData` from `field.GetValue()` (or `field.GetDefault()` if the model is nil/zero), runs `OnLoad`, calls `SetInitial(initialData)`, and locks configuration by setting the `formLoaded` flag. Subsequent calls to `Load()` are no-ops until `Reset()` is called.

### 3. Bind Data and Validate

Bind request or map data with `forms.WithRequestData` or `WithData` and validate using [`forms.IsValid(ctx, mf)`](./readme.md#validation-functions-and-mixins):

```go
mf := modelforms.NewBaseModelForm(r.Context(), article)
mf.SetExclude("ID", "CreatedAt")
mf.Load()

forms.WithRequestData(http.MethodPost, r)(mf)
if r.Method == http.MethodPost && forms.IsValid(r.Context(), mf) {
	// ...
}
```

### 4. Apply (`Instance()`) or Persist (`Save()`)

#### `Instance() T`

When `Cleaned != nil`, `Instance()` runs `SaveFields` to apply `ModelFieldSaver` fields and regular fields (`field.SetValue(value, true)`), skipping `AfterModelFieldSaver` fields, and returns `f.Model`. Calling `Instance()` does **not** persist the model to the database.

#### `Save() (map[string]interface{}, error)`

`Save()` performs the following steps:

1. Verifies there are no field errors and `Cleaned != nil` (failing with status 500 via `except.Assert` if `forms.IsValid` has not been run).
2. Calls `SaveFields` for `ModelFieldSaver` and regular fields (`field.SetValue(value, true)`).
3. Persists `f.Model`:
   - If `f.SaveInstance != nil`, calls `f.SaveInstance(ctx, f.Model)`.
   - Otherwise calls `models.SaveModel(ctx, f.Model)` (returning an error if `saved == false` or `models.SaveModel` fails).
4. Calls `SaveFields` for `AfterModelFieldSaver` fields.
5. Calls `f.Reset()` followed by `f.Load()` and returns the `cleaned` map.

#### `SaveFields` and `Reset`

```go
func (f *BaseModelForm[T]) SaveFields(
	ctx context.Context,
	cleaned map[string]any,
	saveField func(context.Context, forms.Field, attrs.Field, string, any) error,
) error
func (f *BaseModelForm[T]) Reset()
```

`SaveFields` iterates over `ModelFields` (excluding `ModelExclude`) for every field present in both `cleaned` and `f.Field(name)`. `Reset()` resets the underlying `BaseForm` state (`Raw`, `Initial`, `Errors`, `Cleaned`, etc.) and clears the `formLoaded` flag so `Load()` can re-populate from the model.

---

## Complete Example

```go
package views

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

type Post struct {
	ID    int64
	Title string
	Body  string
}

func (p *Post) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		attrs.NewField(p, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(p, "Title", &attrs.Config{MaxLength: 160}),
		attrs.NewField(p, "Body", &attrs.Config{Blank: true}),
	)
}

func (p *Post) Save(ctx context.Context) error {
	// Persist post to database...
	return nil
}

func EditPost(w http.ResponseWriter, r *http.Request, post *Post) error {
	mf := modelforms.NewBaseModelForm(r.Context(), post)
	mf.SetExclude("ID")
	mf.SetOnLoad(func(m *Post, initial map[string]interface{}) {
		if initial["Title"] == "" {
			initial["Title"] = "Draft Post"
		}
	})
	mf.Load()

	forms.WithRequestData(http.MethodPost, r)(mf)
	if r.Method == http.MethodPost && forms.IsValid(r.Context(), mf) {
		if _, err := mf.Save(); err != nil {
			return err
		}
	}

	_ = mf.BoundForm()
	return nil
}
```

---

## Caveats

- **Configuration panics after `Load()`**: Calling `SetInstance`, `SetFields`, or `SetExclude` after `Load()` has run triggers an assertion panic (`formLoaded` is already set). Call `Reset()` first if you need to reconfigure fields and reload.
- **`Instance()` skips `AfterModelFieldSaver`**: Calling `Instance()` updates regular and `ModelFieldSaver` fields on `f.Model` in memory, but deliberately skips `AfterModelFieldSaver` fields. Call `Save()` to persist the model and invoke `AfterModelFieldSaver.SaveField`.
- **`Save()` resets and reloads the form**: At the end of a successful `Save()`, `BaseModelForm` calls `f.Reset()` and `f.Load()`. Read `cleaned` from `Save()`'s return value (or inspect `f.Model`) rather than calling `f.CleanedData()` after `Save()`, because `f.Cleaned` has been cleared.
- **`SetFields` ignores non-model field names**: `SetFields` checks `f.Definition.Field(fieldName)` and skips names not present in the model's `attrs.Definitions`. Any custom field added with `AddField` that participates in `ModelFields` must match a field defined in `FieldDefs`.
