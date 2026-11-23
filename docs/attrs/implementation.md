# `attrs` Package Implementation Details

Models participate in `attrs` by implementing the [`Definer`](./interfaces.md#definer) interface. You can define fields automatically via `attrs` struct tags, manually with `attrs.NewField` and `attrs.Config`, or without struct reflection via [`fattrs`](#fattrs-subpackage) and [`embedder`](#embedder-subpackage).

---

## Implementing the `Definer` Interface

Implement `FieldDefs(ctx context.Context) attrs.Definitions` on a pointer receiver for your model struct:

```go
type Article struct {
	ID    int64
	Title string
}

func (a *Article) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, a,
		attrs.NewField(a, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(a, "Title", &attrs.Config{MaxLength: 200}),
	)
}
```

Do not call `a.FieldDefs(context.Background())` directly in application code; use [`attrs.Define(ctx, a)`](./attrs.md#attrsdefine-and-attributecontext) so `ctx` is guaranteed to carry an attribute context.

---

## Automatic Field Definitions with `attrs` Struct Tags

When using [`attrs.AutoDefinitions(ctx, instance, include...)`](./functions.md#autodefinitions) or `attrs.NewField(instance, name)` (without an explicit `*attrs.Config`), `attrs` parses semicolon-separated options inside the `attrs:"..."` struct tag.

### Recognized Struct Tags

- `attrs:"-"`: Skips the struct field completely during `AutoFieldList`. Anonymous (embedded) and unexported fields are also skipped automatically.
- `attrs:"null"`: Sets `Config.Null = true`.
- `attrs:"blank"`: Sets `Config.Blank = true`.
- `attrs:"readonly"`: Sets `Config.ReadOnly = true`.
- `attrs:"primary"`: Sets `Config.Primary = true`.
- `attrs:"label=<text>"`: Sets `Config.Label`.
- `attrs:"helptext=<text>"`: Sets `Config.HelpText`.
- `attrs:"column=<name>"`: Sets `Config.Column`.
- `attrs:"min_length=<int>"`: Sets `Config.MinLength`.
- `attrs:"max_length=<int>"`: Sets `Config.MaxLength`.
- `attrs:"min_value=<float>"`: Sets `Config.MinValue`.
- `attrs:"max_value=<float>"`: Sets `Config.MaxValue`.
- `attrs:"fk:<modelKey>"` or `attrs:"fk:<modelKey>,<targetField>"`: Configures `Config.RelForeignKey` (`RelManyToOne`) using a deferred relation resolved via `contenttypes.LoadModel`.
- `attrs:"o2o:<modelKey>"` or `attrs:"o2o:<modelKey>,<targetField>"`: Configures `Config.RelOneToOne` using a deferred relation resolved via `contenttypes.LoadModel`.
- `attrs:"default=<value>"`: Parses a default literal for signed/unsigned integers, floats, `string`, `bool`, and nullable SQL types (`sql.NullBool`, `sql.NullInt16`, `sql.NullInt32`, `sql.NullInt64`, `sql.NullFloat64`, `sql.NullString`, `sql.NullTime` in RFC 3339 format), wrapping into pointers when the field type is a pointer.

### Model Key Format for `fk` and `o2o` Tags

Relation tags pass `<modelKey>` to `contenttypes.LoadModel`, which checks lazy registries first and then registered content type names in `<appName>.<ModelName>` format (for example `"users.User"` or `"blog.Post"`), not Go import paths.

**Caveat on `o2o` through-table tags**: Although `autoDefinitionStructTag` checks `len(v) == 5` for `o2o:<target>,<target_field>,<through>,<through_source>,<through_target>`, it never assigns the 3rd, 4th, or 5th elements into `throughModel`, `throughSource`, and `throughTarget`, leaving `deferredThroughModel` with empty strings. To define a one-to-one relation with a through model, configure `RelOneToOne` explicitly in `FieldDefs` with `attrs.Relate(..., attrs.ThroughDeferred(...))` or `&attrs.ThroughModel{...}` instead of using a struct tag.

### Example Using `AutoDefinitions`

```go
package models

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Author struct {
	ID   int64  `attrs:"primary;readonly;label=ID;helptext=Primary identifier"`
	Name string `attrs:"max_length=120;label=Name;helptext=Author display name"`
}

func (a *Author) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, a)
}

type Book struct {
	ID     int64   `attrs:"primary;readonly"`
	Title  string  `attrs:"max_length=200;default=Untitled"`
	Author *Author `attrs:"fk:books.Author,ID"`
	Notes  string  `attrs:"-"`
}

func (b *Book) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, b)
}
```

---

## Manually Defining Fields with `attrs.Make` and `attrs.Config`

For full control over form fields, validators, options, relations, and defaults, pass `attrs.NewField` or `attrs.Unbound` calls with `*attrs.Config` into `attrs.Make(ctx, d, ...)`.

### `attrs.Config` Reference

```go
type Config struct {
	AutoInit             bool                                          `json:"auto_init,omitempty"`
	Null                 bool                                          `json:"allow_null"`
	Blank                bool                                          `json:"allow_blank"`
	ReadOnly             bool                                          `json:"read_only,omitempty"`
	Primary              bool                                          `json:"is_primary,omitempty"`
	IsEmbedded           bool                                          `json:"is_embedded,omitempty"`
	AutoNowAdd           bool                                          `json:"auto_now_add,omitempty"`
	AutoNow              bool                                          `json:"auto_now,omitempty"`
	Precision            int                                           `json:"precision,omitempty"`
	Scale                int                                           `json:"scale,omitempty"`
	NameOverride         string                                        `json:"name_override,omitempty"`
	Label                any                                           `json:"-"`
	HelpText             any                                           `json:"-"`
	Column               string                                        `json:"column,omitempty"`
	MinLength            int64                                         `json:"min_length,omitempty"`
	MaxLength            int64                                         `json:"max_length,omitempty"`
	MinValue             float64                                       `json:"min_value,omitempty"`
	MaxValue             float64                                       `json:"max_value,omitempty"`
	Attributes           map[string]interface{}                        `json:"attributes,omitempty"`
	ReverseAlias         string                                        `json:"reverse_alias,omitempty"`
	RelForeignKey        Relation                                      `json:"-"`
	RelManyToMany        Relation                                      `json:"-"`
	RelOneToOne          Relation                                      `json:"-"`
	RelForeignKeyReverse Relation                                      `json:"-"`
	Default              any                                           `json:"default,omitempty"`
	Options              OptionsType                                   `json:"-"`
	WidgetAttrs          map[string]string                             `json:"widgetAttrs,omitempty"`
	Validators           []func(interface{}) error                     `json:"-"`
	FormField            func(opts ...func(fields.Field)) fields.Field `json:"-"`
	FormWidget           func(Config) widgets.Widget                   `json:"-"`
	OnInit               func(Definer, *FieldDef, *Config) *Config     `json:"-"`
}
```

- `AutoInit`: When accessing a nested struct field via pointer indirection in `NewField`, automatically allocates nil intermediate structs/maps/slices (and initializes `Definer` types via `NewObject`) instead of panicking.
- `Null`: Permits `nil` / invalid values in `SetValue`.
- `Blank`: Permits zero/blank values (such as `""`, empty slices/maps, nil pointers, or zero structs) in `SetValue`, and controls whether form fields are marked `Required(true)` (`!field.AllowBlank()`).
- `ReadOnly`: Marks the field read-only (`AllowEdit()` returns `!ReadOnly`).
- `Primary`: Marks the field as a primary key.
- `IsEmbedded`: Marks the field as belonging to an embedded or proxy struct (`Embedder.Embedded()`).
- `AutoNowAdd` / `AutoNow`: Metadata flags indicating timestamps populated on creation or update (used by ORM backends).
- `Precision` / `Scale`: Decimal precision and scale stored in `Attrs()` under `AttrPrecisionKey` and `AttrScaleKey`.
- `NameOverride`: Overrides `Field.Name()`.
- `Label` / `HelpText`: Accepted as `any` and resolved through `trans.GetText(ctx, ...)` (supporting `string`, `func(context.Context) string`, and `trans.Translatable`). `HelpText` panics if a non-nil value is not supported by `trans.GetText`.
- `Column`: Overrides the database column name returned by `ColumnName()`.
- `MinLength` / `MaxLength` / `MinValue` / `MaxValue`: Bounds validated by `FieldDef.Validate()` and forwarded to generated form field options.
- `Attributes`: Arbitrary extra attributes merged into the map returned by `Attrs()`.
- `ReverseAlias`: Custom field name used when registering the reverse relation on the target model.
- `RelForeignKey`, `RelManyToMany`, `RelOneToOne`, `RelForeignKeyReverse`: Relationship definitions created with [`attrs.Relate`](./model-meta.md#relate-and-reverserelation) or [`attrs.RelatedDeferred`](./model-meta.md#throughdeferred-and-relateddeferred).
- `Default`: A default value or a function (`func() T`, `func(Model) T`, `func() (T, error)`, `func(Model) (T, error)`).
- `Options`: Choice definitions for the field (see below).
- `WidgetAttrs`: HTML attributes copied onto the generated `fields.Field`.
- `Validators`: Custom validation callbacks invoked in `FieldDef.Validate()`.
- `FormField`: Custom constructor returning a `fields.Field` with auto-generated `opts` already populated from `Config`.
- `FormWidget`: Custom constructor returning a `widgets.Widget` for the generated form field.
- `OnInit`: Hook executed at the end of `NewField(instance, name, conf...)` that can inspect `Definer` / `*FieldDef` and return a modified `*Config`.

Note that `Config` does not have `Setter` or `Getter` function fields. Method-based field hooks (`Set<FieldName>`, `Get<FieldName>`, `GetDefault<FieldName>`, `<FieldName>ToString`) only run when the package-level `attrs.ALLOW_METHOD_CHECKS` variable is set to `true` (it defaults to `false`).

### `Options` and Automatic Select Widgets

`Config.Options` (`OptionsType`) accepts any of the following types:

- `[]attrs.Option` (where `Option` is `struct { Label trans.Translatable; Val any }`)
- `func() []attrs.Option` or `func() ([]attrs.Option, error)`
- `func(attrs.Definer) []attrs.Option` or `func(attrs.Definer) ([]attrs.Option, error)`
- `func(context.Context) []attrs.Option` or `func(context.Context) ([]attrs.Option, error)`
- `func(context.Context, attrs.Definer) []attrs.Option` or `func(context.Context, attrs.Definer) ([]attrs.Option, error)`

When `Config.FormWidget` is `nil` and `Config.Options` matches one of those types, `Config.GetFormField` automatically replaces the form field's widget with `options.NewSelectInput` backed by `TryOptions`. If `Config.Blank` is `true`, it also passes `options.IncludeBlank(true)`.

### Attribute Keys (`Attr*Key` Constants)

The `attrs` package defines standard keys for `FieldDefinition.Attrs()` (populated automatically by `Config.Attrs()` and `FieldDef.Attrs()`):

```go
const (
	AttrNameKey          = "field.name"
	AttrMaxLengthKey     = "field.max_length"
	AttrMinLengthKey     = "field.min_length"
	AttrMinValueKey      = "field.min_value"
	AttrMaxValueKey      = "field.max_value"
	AttrAllowNullKey     = "field.allow_null"
	AttrAllowBlankKey    = "field.allow_blank"
	AttrAllowEditKey     = "field.read_only"
	AttrIsPrimaryKey     = "field.primary"
	AttrAutoIncrementKey = "field.auto_increment"
	AttrUniqueKey        = "field.unique"
	AttrReverseAliasKey  = "field.reverse_alias"
	AttrPrecisionKey     = "field.precision"
	AttrScaleKey         = "field.scale"
	AttrUploadToKey      = "field.upload_to"
	AttrAutoNowAddKey    = "field.auto_now_add"
	AttrAutoNowKey       = "field.auto_now"
)
```

Note that `AttrAllowEditKey` has the underlying string value `"field.read_only"`, while `Config.Attrs()` stores `!f.ReadOnly` (whether editing is allowed) under that key.

### Manual Definition Example (with `attrs.Unbound`)

`attrs.Unbound` lets you declare reusable field definitions without repeating the receiver variable:

```go
type Profile struct {
	ID      int64
	Bio     string
	Website string
	Role    string
	User    *User
}

var sharedBioField = attrs.Unbound("Bio", &attrs.Config{
	Blank:    true,
	Label:    "Biography",
	HelpText: "Short public bio",
})

func (p *Profile) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		attrs.NewField(p, "ID", &attrs.Config{
			Primary:  true,
			ReadOnly: true,
			Label:    "ID",
		}),
		sharedBioField,
		attrs.NewField(p, "Role", &attrs.Config{
			Default: "member",
			Options: []attrs.Option{
				{Label: "Member", Val: "member"},
				{Label: "Editor", Val: "editor"},
			},
		}),
		attrs.NewField(p, "User", &attrs.Config{
			RelForeignKey: attrs.Relate(&User{}, "", nil),
			Column:        "user_id",
		}),
	)
}
```

### Caveats for `FieldDef.GetDefault` and `FieldDef.Validate`

- **`FieldDef.GetDefault` return value and dead-code path**:
  - When `Config.Default` is a function, `FieldDef.GetDefault()` currently returns `outVal` (a `reflect.Value`) instead of `outVal.Interface()`.
  - When `Config.Default` is `nil`, `FieldDef.GetDefault()` returns `nil` early (right after any `ALLOW_METHOD_CHECKS` lookup), leaving the `DefaultForType` goldcrest hook and struct zero-value fallback unreachable. Set `Config.Default` directly on fields when you need defaults.
- **`FieldDef.Validate` numeric bounds**:
  - `MinValue` and `MaxValue` checks in `FieldDef.Validate()` only run on signed and unsigned integer kinds (`int*` and `uint*`), skipping `float32`/`float64` fields.
  - Negative `MinValue` comparisons in `FieldDef.Validate()` check `num > f.Config.MinValue`, which reverses the comparison for negative minimums.

---

## `fattrs` Subpackage

Import path: `github.com/Nigel2392/go-django/src/core/attrs/fattrs`.

The `fattrs` package provides high-performance field definitions that avoid reflection on field reads/writes and cache computed field options per model type.

### `fattrs.Field` (Pointer-Backed Fields)

`fattrs.Field` binds directly to a pointer to a struct field (`&m.FieldName`) and wraps `PtrFieldConfig[MODEL, VALUE]`:

```go
type PtrFieldConfig[MODEL attrs.Definer, VALUE any] struct {
	attrs.Config
	Flags    fieldFlag
	Default  interface{} // func(MODEL) *VALUE | func(MODEL) VALUE | *VALUE | VALUE | fattrs.NULL
	Validate func(obj MODEL, value VALUE) error
	New      func() VALUE
	IsZero   func(VALUE) bool
}

func Field[MODEL attrs.Definer, VALUE any](
	obj MODEL,
	name string,
	ptr *VALUE,
	cnf func() PtrFieldConfig[MODEL, VALUE],
	optionsKey ...string,
) attrs.Field
```

The `cnf` closure is evaluated lazily (or during `OnModelRegister`) and cached globally per `(reflect.TypeOf(obj), name, optionsKey...)`. Pass `fattrs.NULL{}` (or `nil`) for `Default` when a zero default is desired.

```go
func (p *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		fattrs.Field(p, "ID", &p.ID, func() fattrs.PtrFieldConfig[*Product, int64] {
			return fattrs.PtrFieldConfig[*Product, int64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(p, "Title", &p.Title, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config:  attrs.Config{MaxLength: 150},
				Default: "Untitled",
			}
		}),
	)
}
```

### `fattrs.NewField` (`ReflectlessField`)

`fattrs.NewField` works with custom getter/setter/driver-value closures (`FieldConfig[T]`) rather than requiring a direct field pointer:

```go
type FieldConfig[T attrs.Definer] struct {
	attrs.Config
	GetValue       func(obj T) interface{}
	GetDriverValue func(obj T) (interface{}, error)
	SetValue       func(obj T, value any) error
	SetDriverValue func(obj T, value any) error
	Validate       func(obj T, value any) error
	Default        interface{} // must be non-nil; use fattrs.NULL{} for nil default
}

func NewField[T attrs.Definer](obj T, name string, cnf func() FieldConfig[T]) attrs.Field
```

In `FieldConfig`, `GetValue`, `SetValue`, and `Default` are mandatory (passing `nil` panics; use `fattrs.NULL{}` for a `nil` default).

### `fattrs.RegisterModel`, `AddOptions`, and `GetOptions`

You can pre-register field configurations for a model type via `RegisterModel` or manage cached option entries directly:

```go
func RegisterModel[T attrs.Definer](register func(addField func(string, FieldConfig[T])))
func AddOptions(typ reflect.Type, name string, opts any, extra ...string)
func GetOptions[OPTIONS any](typ reflect.Type, name string, extra ...string) (opts OPTIONS, ok bool)
```

### `fattrs` Caveats

- **`SetValue` bypasses constraints**: Neither `ReflectlessField.SetValue` nor `ptrField.SetValue` checks `force`, `AllowEdit()`, `AllowBlank()`, or `AllowNull()`, nor do they automatically call `Validate()` or `SignalChange` on `Definitions`.
- **`ReflectlessField.OnModelRegister` signature mismatch**: `ReflectlessField.OnModelRegister` takes `(model attrs.Definer, outer attrs.FieldDef) error` instead of `(model attrs.Definer, top attrs.FieldDefinition) error`, so `ReflectlessField` does not implement `attrs.CanOnModelRegister` (whereas `ptrField` does).
- **`ptrField` lazy-options nil panics**: `ForSelectAll()`, `CanMigrate()`, `AllowDBEdit()`, and `IsReverse()` on `ptrField` read `r.opts.flags` directly without calling `r.useOpts()`, which will panic if accessed before `OnModelRegister` or another method triggers `useOpts()`. Additionally, if a `Binder` value returns an error in `ptrField.GetValue()`/`GetDefault()`, `ptrField.get()` invokes an uninitialized `r.opts.bindError` callback, causing a nil-pointer panic; and if `useOpts()` returns `nil`, metadata methods dereference `nil`.

---

## `embedder` Subpackage

Import path: `github.com/Nigel2392/go-django/src/core/attrs/embedder`.

The `embedder` package automatically discovers embedded structs (value or pointer, including nested pointer-to-struct hierarchies) and runs setup or save lifecycle hooks on them.

### Registries and Hook Methods

`embedder` exposes three pre-configured registries and a factory for custom method names:

```go
type Registry interface {
	Bind(ctx context.Context, obj any) error
	Count(obj any) int
	Register(obj any)
	Walk(obj any, walkFn func(depth int, field reflect.StructField) (stop bool)) bool
}

var (
	Init     Registry = NewRegistry("SetupEmbedded")
	PreSave  Registry = NewRegistry("PreSaveEmbedded")
	PostSave Registry = NewRegistry("PostSaveEmbedded")
)
```

`attrs.RegisterModel` automatically registers models with `embedder.Init`, `embedder.PreSave`, and `embedder.PostSave` and calls `embedder.Init.Bind`. `attrs.NewObject` and `attrs.Setup` also invoke `embedder.Init.Bind`.

### Supported Method Signatures

Embedded types can declare `SetupEmbedded`, `PreSaveEmbedded`, or `PostSaveEmbedded` (with either value or pointer receiver) matching any of these signatures:

```go
func (e *MyEmbedded) SetupEmbedded(parent any)
func (e *MyEmbedded) SetupEmbedded(parent *ParentModel) error
func (e *MyEmbedded) SetupEmbedded(parent reflect.Value, field reflect.StructField) error
func (e *MyEmbedded) SetupEmbedded(ctx context.Context, parent any) error
func (e *MyEmbedded) SetupEmbedded(ctx context.Context, parent *ParentModel, field reflect.StructField) error
```

During `Bind`, `embedder` allocates any `nil` pointer fields along the embedding path and passes the immediate parent struct (or `reflect.Value`), plus the `reflect.StructField` if requested. Use `embedder:"-"` on a struct field tag to opt a field out of `embedder` discovery.

### `RootValueContext`

Inside a hook receiving `ctx context.Context`, call `embedder.RootContext(ctx)` to obtain the `*embedder.RootValueContext` containing the top-level `Root` object passed to `Bind`:

```go
func (e *MyEmbedded) SetupEmbedded(ctx context.Context, parent any) error {
	if rc := embedder.RootContext(ctx); rc != nil {
		// rc.Root is the top-level object passed to Bind
		_ = rc.Root
	}
	return nil
}
```

### `embedder` Caveats

- **Context lifetime**: `Registry.Bind` allocates `RootValueContext` on the stack (guarded with `runtime.KeepAlive`). Hooks must **never** store `ctx` or `*embedder.RootValueContext` beyond the return of `Bind`.
- **Duplicate embedded types**: `Registry.Register` tracks visited `reflect.Type` entries across the entire struct in a `seen` map. If a model embeds two fields of the exact same type, only the first field is registered and bound; subsequent fields of that type are skipped.
- **Value vs. pointer recursion**: `embedder` only recurses into subfields of pointer-to-struct fields (`*Embedded`). Non-pointer embedded structs are checked for the hook method directly without recursing into their own fields.
