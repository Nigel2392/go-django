# Model Implementation Details (`queries/src/models`)

The `models.Model` struct provides optional but recommended infrastructure for `attrs.Definer` models in `queries`. Embed `models.Model` by value directly inside your model struct or inside a value-embedded base struct (never embed `*models.Model` as a pointer).

Note that `models.Model` does not implement `attrs.Definer` itself; your outer struct implements `FieldDefs(ctx context.Context) attrs.Definitions` and delegates field setup to `m.Model.Define(ctx, m, ...)`.

---

## Extra Functionality Provided by `models.Model`

- `Define(ctx, def, flds...)`: Sets up `m.Model`, binds `attrs.ObjectDefinitions`, resolves `table:"..."` struct tags, injects reverse relation fields, and registers proxy fields.
- Annotation storage (`Annotations map[string]any`) and `DataStore() queries.ModelDataStore` (`MapDataStore`).
- Embedded proxy model discovery, automatic setup, and cascade saving (see [Proxy Models](./proxy_models.md)).
- Dirty-field state tracking via `*state.ModelState` (`queries/src/models/state`).
- Instance persistence methods (`Save`, `Create`, `Update`, `Delete`, `Load`, `SaveObject`, `DeleteObject`).
- Integration with `django/models.ContextSaver`, `ThroughModelSetter`, `attrs.CanSignalChanged`, and `attrs.CanCreateObject[attrs.Definer]`.

Note that `models.Model` does **not** provide a `GetQuerySet()` method; query your model using `queries.GetQuerySet(m)` or `queries.QS(m)` (or implement `QuerySetDefiner` / `TypedQuerySetDefiner[T]` on your own struct).

---

## Basic Embedding and `Define`

```go
package models

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type BaseModel struct {
	models.Model
}

type MyModel struct {
	BaseModel // Valid: value-embedded struct containing models.Model
	ID        int64
	Name      string
}

func (m *MyModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{
			Primary: true,
		}),
		attrs.NewField(m, "Name", nil),
	)
}
```

---

## Retrieving Annotated Values

When `models.Model` is embedded, annotations returned from queries are stored in `m.Annotations` (and pushed to matching model fields if present in `Defs()`):

```go
row, err := queries.GetQuerySet(&MyModel{}).
	WithContext(ctx).
	Filter("ID", 1).
	Select("ID", "Name").
	Annotate("NameLower", expr.LOWER("Name")).
	First()
if err != nil {
	return err
}

myModel := row.Object
fmt.Println(myModel.ID, myModel.Name, myModel.Annotations["NameLower"])
```

---

## Persistence: `Save`, `Create`, `Update`, `Delete`, `Load`, and `SaveConfig`

### `SaveableObject` and `DeleteableObject`

Implement `models.SaveableObject` or `models.DeleteableObject` on your outer struct to customize save or delete behavior, delegating back to `m.Model.SaveObject(ctx, cnf)` or `m.Model.DeleteObject(ctx)`:

```go
type SaveableObject interface {
	SaveObject(ctx context.Context, cnf SaveConfig) error
}

type DeleteableObject interface {
	DeleteObject(ctx context.Context) error
}
```

```go
func (m *MyModel) SaveObject(ctx context.Context, cnf models.SaveConfig) error {
	m.Name = strings.TrimSpace(m.Name)
	return m.Model.SaveObject(ctx, cnf)
}
```

### `SaveConfig` and Context Helpers

```go
type SaveConfig struct {
	QuerySet      *queries.QuerySet[attrs.Definer]
	Fields        []string
	IncludeFields []string
	ForceCreate   bool
	ForceUpdate   bool
}

func NewSaveConfigContext(SaveConfig SaveConfig) context.Context
func SaveConfigContext(ctx context.Context, cnf SaveConfig) context.Context
```

- `Fields`: Limits which fields are saved on DB-loaded models.
- `IncludeFields`: Forces listed fields to be written even if `*state.ModelState` marks them unchanged.
- `ForceCreate` / `ForceUpdate`: Forces `INSERT` or `UPDATE` regardless of `Saved()` / `flagFromDB`.
- `QuerySet`: Custom `*queries.QuerySet[attrs.Definer]` to execute the write.

Pass a `SaveConfig` to `m.Save(ctx)` using `models.SaveConfigContext(ctx, cnf)` (or `models.NewSaveConfigContext(cnf)`). Convenience methods `m.Create(ctx)` and `m.Update(ctx)` automatically set `ForceCreate: true` and `ForceUpdate: true` respectively.

### `Delete(ctx)` and `DeleteObject(ctx)`

`m.Delete(ctx)` dispatches to your model's `DeleteObject(ctx)` if it implements `DeleteableObject`, otherwise calling `m.Model.DeleteObject(ctx)`, which executes `BeforeDelete` -> `QuerySet.Filter(where).Delete()` -> resets state and `flagFromDB` -> `AfterDelete`.

### `Load(ctx, loadOptions...)`

`m.Load(ctx, opts...)` refreshes a bound model from the database in place (targeting it via `queries.GenerateObjectsWhereClause`) and re-synchronizes embedded `Model` internals, proxies, annotations, and `flagFromDB`:

```go
type ModelLoadConfig struct {
	ExcludedFields []string
	Setter         func(attrs.Definer) error
	QuerySet       *queries.QuerySet[attrs.Definer]
}

func LoadSetter(setter func(attrs.Definer) error) func(*ModelLoadConfig)
func LoadQS(load func(qs *queries.QuerySet[attrs.Definer]) *queries.QuerySet[attrs.Definer]) func(*ModelLoadConfig)
func LoadExcl(fields ...string) func(*ModelLoadConfig)
```

```go
err := myModel.Load(ctx,
	models.LoadExcl("LocalOnlyField"),
	models.LoadQS(func(qs *queries.QuerySet[attrs.Definer]) *queries.QuerySet[attrs.Definer] {
		return qs.Select("*", "Profile.*")
	}),
)
```

---

## Dirty-State Tracking (`queries/src/models/state`)

`m.State()` returns `*state.ModelState` (`github.com/Nigel2392/go-django/queries/src/models/state`), initialized automatically after queries (`AfterQuery`) and reset after `SaveObject` / `DeleteObject`:

```go
type ModelState struct { ... }

func New(model *attrs.ObjectDefinitions) *ModelState
func (m *ModelState) Mark(fieldName string)
func (m *ModelState) Changed(checkState bool) bool
func (m *ModelState) FieldChanged(fieldName string) bool
func (m *ModelState) InitialValue(fieldName string) (interface{}, bool)
func (m *ModelState) Reset()
func (m *ModelState) ResetFieldState(fieldName string)
```

- `Changed(true)` re-scans field values with `reflect.DeepEqual` against captured initial values and updates the changed set.
- `Model.SignalChange(fa, value)` and `Model.SignalReset(fa)` emit `ModelChangeSignal` (`FlagFieldChanged`, `FlagFieldReset`, `FlagModelSetup`, `FlagModelReset`, `FlagProxySetup`, `FlagProxyChanged`) on the model's internal signal channel to keep `*state.ModelState` updated when values are modified via `attrs.Field`.
- A `nil` `*state.ModelState` treats `Changed` and `FieldChanged` as `true`.

---

## Proxy Helper Interfaces, `ExtractModel`, and Signals

- `CanTargetDefiner`: Implemented by polymorphic proxy targets to expose `TargetContentTypeField()` and `TargetPrimaryField()`.
- `CanControlSaving`: Implemented by polymorphic proxy models (`ControlsEmbedderSaving() bool`) to suppress automatic cascade saving of the proxy field in `SaveObject`.
- `models.ExtractModel(def attrs.Definer) (*Model, error)`: Walks the precomputed embedded chain on `def` and returns a pointer to its embedded `models.Model`.
- `models.SIGNAL_MODEL_SETUP`: Emitted during `Model.Setup(ctx, def)` (`FlagModelSetup`, `FlagProxySetup`, `FlagModelReset`) and forwarded to `queries.SignalModelInit`.

---

## Model Methods Reference

| Method | Signature | Description |
|---|---|---|
| `Setup` | `Setup(ctx context.Context, def attrs.Definer) error` | Initializes model internals, proxies, and signals |
| `Define` | `Define(ctx context.Context, def attrs.Definer, flds ...any) *attrs.ObjectDefinitions` | Defines fields, `table` tag, reverse relations, and proxy fields |
| `Defs` | `Defs() *attrs.ObjectDefinitions` | Returns cached `*attrs.ObjectDefinitions` (panics if not set up) |
| `PK` | `PK() attrs.Field` | Returns the primary key `attrs.Field` (or `nil`) |
| `Object` | `Object() attrs.Definer` | Returns the outer `attrs.Definer` instance |
| `DataStore` | `DataStore() queries.ModelDataStore` | Returns the `MapDataStore` for annotations/relations |
| `State` | `State() *state.ModelState` | Returns (and initializes if needed) the dirty-state tracker |
| `Saved` | `Saved() bool` | Returns `true` if `flagFromDB` is set or `PK()` is non-zero |
| `Save` | `Save(ctx context.Context) error` | Saves via `SaveableObject` or `SaveObject` |
| `Create` | `Create(ctx context.Context) error` | Saves with `ForceCreate: true` |
| `Update` | `Update(ctx context.Context) error` | Saves with `ForceUpdate: true` |
| `Delete` | `Delete(ctx context.Context) error` | Deletes via `DeleteableObject` or `DeleteObject` |
| `Load` | `Load(ctx context.Context, loadOptions ...func(*ModelLoadConfig)) error` | Reloads the model in place from the database |
| `AfterQuery` | `AfterQuery(ctx context.Context) error` | Initializes state and marks `flagFromDB` |
| `AfterSave` | `AfterSave(ctx context.Context) error` | Marks `flagFromDB` |

---

## Lifecycle Hooks (Actor Interfaces)

`Model.SaveObject` and `Model.DeleteObject` execute the `queries` actor hooks (`BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeSave`, `AfterSave`, `BeforeDelete`, `AfterDelete`, `AfterQuery`). If your outer struct overrides `AfterQuery` or `AfterSave`, remember to call `m.Model.AfterQuery(ctx)` / `m.Model.AfterSave(ctx)` so `flagFromDB` and state tracking remain accurate.

If you call `QuerySet.Create` or `QuerySet.Update` from inside an actor hook, call `.ExplicitSave()` on the QuerySet to avoid recursive hook loops (see [Writing Queries](../queryset/writing_queries.md)).

---

## Caveats

- **Dirty-check no-op on unchanged DB models:** In `SaveObject`, if the model was loaded or previously saved (`flagFromDB` is set), `SaveConfig.Force()` is `false`, `Fields` / `IncludeFields` are empty, and `State.Changed(true)` is `false` (or `State == nil`), `SaveObject` returns `nil` immediately without issuing an `UPDATE`. Pass `SaveConfig{ForceUpdate: true}` (or call `m.Update(ctx)` / specify `IncludeFields`) when you want an `UPDATE` regardless.
- **Pre-generated primary keys (`UUID`, `ULID`, etc.):** `m.Saved()` returns `true` whenever `PK()` is non-zero, even if `flagFromDB` is `false`. If you instantiate a new struct with a pre-generated UUID/ULID primary key and call `m.Save(ctx)` (or `django/models.SaveObject`), `SaveObject` (and `MODEL_SAVE_HOOK`) will attempt an `UPDATE` instead of an `INSERT` and return `errors.NoChanges`. Use `m.Create(ctx)`, `SaveConfigContext(ctx, models.SaveConfig{ForceCreate: true})`, or `queries.CreateObject` / `QuerySet.Create` for new records with client-generated PKs.
- **Value-embed order before proxy fields:** `models.Model` (or a value-embedded struct containing `models.Model`) **must** appear in struct field order *before* any proxy pointer fields. `buildModelChain` (`queries/src/models/chain.go`) only discovers proxies once `base != nil` (and panics inside embedded base structs if a proxy field precedes `models.Model`).
- **`models.Saver` field panic:** Inside `SaveObject`, any field implementing the legacy non-context `django/models.Saver` interface causes a panic (`"Saver is not supported in Save(), a ContextSaver is required to maintain transaction integrity"`). Use `queries.SaveableField` or `queries.SaveableDependantField` instead.
- **Always call `models.Setup(ctx, obj)` on struct literals:** Constructing `&MyModel{}` directly without `models.Setup(ctx, ...)` or `attrs.NewObject` leaves `Model.internals` nil; `Save`, `Create`, `Update`, `Delete`, and `Load` return `ErrModelInitialized`, and `Defs`/`PK`/`Object`/`State` panic.
