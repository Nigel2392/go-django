# Formsets

The `formsets` package (`github.com/Nigel2392/go-django/src/forms/formsets`) manages collections of repeated forms under indexed prefixes (`0-<field>`, `1-<field>`, or `<prefix>-0-<field>`), supporting dynamic addition, deletion, ordering, per-form validation, and batch saving.

---

## Interfaces, Options, and Constants

### Field Name Constants

```go
const (
	TOTAL_FORM_COUNT    = "TOTAL_FORMS"
	INITIAL_FORM_COUNT  = "INITIAL_FORMS"
	MIN_NUM_FORM_COUNT  = "MIN_NUM_FORMS"
	MAX_NUM_FORM_COUNT  = "MAX_NUM_FORMS"
	ORDERING_FIELD_NAME = "__ORDER__"
	DELETION_FIELD_NAME = "__DELETE__"
)
```

### `ManagementForm`

`ManagementForm` tracks the submitted form count and min/max constraints in hidden fields prefixed `<prefix>-management` (when a prefix is set on the formset):

```go
type ManagementForm struct {
	forms.Form

	TotalFormsValue int
	InitialForms    int
	MinNumForms     int
	MaxNumForms     int
}

func NewManagementForm(ctx context.Context, opts ...func(*ManagementForm)) *ManagementForm
```

`NewManagementForm` adds hidden `TOTAL_FORMS` (required integer), `INITIAL_FORMS`, `MIN_NUM_FORMS`, and `MAX_NUM_FORMS` fields and validates that `TOTAL_FORMS` is an integer within `[MinNumForms, MaxNumForms]` (when `MaxNumForms > 0`).

### `ListFormSet[T]`

```go
type BaseFormSetForm = forms.Minimum

type ListFormSet[T BaseFormSetForm] interface {
	forms.WithDataDefiner
	forms.ErrorDefiner
	forms.FormFieldDefiner
	Load()
	WithContext(ctx context.Context)
	Context() context.Context
	AddFormError(errorList ...error)
	HasChanged() bool
	Media() media.Media
	Prefix() string
	SetPrefix(prefix string)
	PrefixForm(fld any) string
	DeletedForms() []T
	ManagementForm() *ManagementForm
	Forms() ([]T, error)
	SetForms(forms []T)
	SetDeletedForms(forms []T)
	NewForm(ctx context.Context) T
	Initial(ctx context.Context, totalForms int) (base map[string]interface{}, list []map[string]interface{})
	Form(index int) (form T, ok bool)
	InitialData() map[string]interface{}
	CleanedData() map[string]any // Always returns nil on BaseFormSet; use CleanedDataList
	CleanedDataList() []map[string]any
	ErrorLists() [][]error
	BoundErrorsList() []*orderedmap.OrderedMap[string, []error]
	Save() ([]any, error)
}
```

### `FormsetOptions[FORM]` and `NewBaseFormSet`

```go
type FormsetOptions[FORM BaseFormSetForm] struct {
	NewForm          func(c context.Context) FORM
	BeforeCheckValid func(ctx context.Context, formObj any) error
	MinNum           int
	Extra            int
	MaxNum           int
	CanOrder         bool
	CanDelete        bool
	CanAdd           bool
	SkipPrefix       bool
	HideDelete       bool
	DefaultForms     func(ctx context.Context, max, min int) ([]FORM, error)
	DeleteForms      func(ctx context.Context, forms []FORM) error
	GetDefaults      func(ctx context.Context, totalForms int) []map[string]interface{}
	BaseDefaults     func(ctx context.Context) map[string]interface{}
}

func NewBaseFormSet[FORM BaseFormSetForm](ctx context.Context, opts FormsetOptions[FORM]) *BaseFormSet[FORM]
```

- `NewForm`: Factory creating a new blank child form. Required when adding/initializing forms dynamically or looking up `Field`/`Widget`.
- `BeforeCheckValid`: Optional hook executed at the start of `CheckIsValid`.
- `MinNum`, `Extra`, `MaxNum`: Control the initial number of forms (`min(MinNum+Extra, max(MaxNum, MinNum+Extra))`) and management-form bounds.
- `CanAdd`: Allows submitted `TOTAL_FORMS` to exceed the initial `Forms()` count and instantiates additional child forms via `NewForm`. When `false`, submitting more forms than initial forms adds a validation error.
- `CanOrder`: Adds a hidden integer `__ORDER__` field to each child form and sorts kept and deleted forms stably by their submitted `__ORDER__` value.
- `CanDelete` / `HideDelete`: Adds a boolean `__DELETE__` field to each child form (hidden when `HideDelete: true`). Checked forms are moved to `DeletedForms()` and excluded from `Forms()`, `CleanedDataList()`, and per-form `Save()`.
- `SkipPrefix`: When `true`, leaves each child form's existing prefix untouched instead of assigning `<formsetPrefix>-<i>` (or `<i>`).
- `DefaultForms`: Provides pre-populated child forms (for example, existing model forms loaded from a queryset).
- `DeleteForms`: Callback invoked by `Save()` when `len(DeletedForms()) > 0` to delete removed items.
- `GetDefaults` / `BaseDefaults`: Supply per-index or shared fallback initial data maps.

---

## Lifecycle and Validation

### 1. Initialize and Load

Create the formset with `formsets.NewBaseFormSet(ctx, opts)` (optionally calling `SetPrefix("items")` and `Load()` for GET rendering):

```go
fs := formsets.NewBaseFormSet(r.Context(), formsets.FormsetOptions[forms.Form]{
	NewForm: func(c context.Context) forms.Form {
		return forms.Initialize(
			forms.NewBaseForm(c),
			forms.WithFields(fields.CharField(fields.Name("title"), fields.Required(true))),
		)
	},
	MinNum:    1,
	Extra:     1,
	MaxNum:    5,
	CanAdd:    true,
	CanDelete: true,
	CanOrder:  true,
})
fs.SetPrefix("items")
fs.Load()
```

`Load()` initializes `Forms()`, populates `ManagementForm` initial counts (`TOTAL_FORMS`, `INITIAL_FORMS`, `MIN_NUM_FORMS`, `MAX_NUM_FORMS`), and calls `Load()` on any child form that implements `Load()` (such as [`modelforms.BaseModelForm`](./modelforms.md)).

### 2. Validate with `forms.IsValid`

`BaseFormSet` implements `CheckIsValid(ctx context.Context, formObj any) bool`, so pass the formset directly to [`forms.IsValid(ctx, fs)`](./readme.md#validation-functions-and-mixins):

1. Runs `BeforeCheckValid` and any embedded `PrevalidatorMixin` hooks.
2. If `ManagementForm() != nil`, binds data to the management form and validates it to obtain `totalForms`; otherwise uses `len(formList)`.
3. Rejects submissions where `!CanAdd && totalForms > len(formList)`.
4. Iterates child slots, applies prefixes (`PrefixForm(i)`), binds submitted data, reads `__ORDER__` and `__DELETE__`, marks deleted slots, applies `GetDefaults`/`BaseDefaults`, validates each non-deleted child form via `forms.IsValid`, and runs validators added via `AddValidator`.
5. Sorts child forms by `__ORDER__`, updates `SetForms(finalForms)` and `SetDeletedForms(deletedForms)`, and returns `isValid && !forms.HasErrors(fs)`.

### 3. Inspect Results and Errors

- `AddValidator(validator ...func(FORM, map[string]any) []error)`: Attaches cross-field validators executed for each valid child form during `CheckIsValid`.
- `ForEach(fn func(form FORM, index int) error) error`: Iterates `Forms()`.
- `CleanedDataList() []map[string]any`: Returns `CleanedData()` from each active child form in order.
- `ErrorLists() [][]error`: Returns each active child form's `ErrorList()`.
- `BoundErrorsList() []*orderedmap.OrderedMap[string, []error]`: Returns each active child form's `BoundErrors()`.
- `ErrorList() []error` and `BoundErrors()`: Aggregate form-level errors and bound field errors across all active forms (plus formset-level errors on `ErrorList()`).
- `DeletedForms() []FORM`: Returns forms marked with `__DELETE__`.

### 4. Save

```go
results, err := fs.Save()
```

For each form in `FormList` (skipping any form whose cleaned `__DELETE__` value is `true`), `Save()` reflects on the child form's `Save` method via `versatile.Method` (passing `b.ctx` if accepted) supporting both `func() error` and `func() (any, error)` signatures, collecting returned results into `results`. Afterward, if `opts.DeleteForms != nil` and `len(DeletedForms()) > 0`, it calls `opts.DeleteForms(b.ctx, fs.DeletedForms())`.

---

## Complete Example

```go
package views

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/formsets"
)

func TagFormSetHandler(w http.ResponseWriter, r *http.Request) {
	fs := formsets.NewBaseFormSet(r.Context(), formsets.FormsetOptions[forms.Form]{
		NewForm: func(c context.Context) forms.Form {
			return forms.Initialize(
				forms.NewBaseForm(c),
				forms.WithFields(
					fields.CharField(fields.Name("name"), fields.Required(true)),
				),
			)
		},
		MinNum:    1,
		Extra:     1,
		MaxNum:    10,
		CanAdd:    true,
		CanDelete: true,
		DeleteForms: func(ctx context.Context, deleted []forms.Form) error {
			// Delete removed items...
			return nil
		},
	})
	fs.SetPrefix("tags")

	if r.Method == http.MethodPost {
		forms.WithData(r.PostForm, nil, r)(fs)
		if forms.IsValid(r.Context(), fs) {
			cleanedItems := fs.CleanedDataList()
			_, err := fs.Save()
			_ = cleanedItems
			_ = err
		}
	} else {
		fs.Load()
	}
}
```

---

## Caveats

- **`ManagementForm()` can be `nil`**: `NewBaseFormSet` only instantiates a `ManagementForm` when `!((opts.MinNum+opts.Extra) == opts.MaxNum && (opts.MinNum > 0 || !opts.CanAdd))`. For fixed-size formsets where `MinNum + Extra == MaxNum` and either `MinNum > 0` or `CanAdd == false`, `fs.ManagementForm()` returns `nil` — always nil-check `fs.ManagementForm()` before rendering it.
- **`CleanedData()` always returns `nil`**: Calling `fs.CleanedData()` on `BaseFormSet` always returns `nil`. Always call `fs.CleanedDataList()` to inspect cleaned per-form data.
- **`Load()` skip condition**: `Load()` returns early (logging a warning) if `Request != nil` alongside `Values != nil` or `Files != nil`, or if `forms.HasErrors(fs)` is true.
- **`formsets/cluster` is experimental**: The `github.com/Nigel2392/go-django/src/forms/formsets/cluster` subpackage (`ClusterForm`, `ClusteredForm`, `SAVE_ORDERING_PRE`/`NONE`/`POST`) is currently incomplete/experimental and does not yet have public configuration helpers for its internal `formsFunc` or a complete test suite.
