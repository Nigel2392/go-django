# Querying, Inserting, Updating, and Deleting Objects

QuerySets are constructed with [`queries.GetQuerySet`](./queryset/queryset.md) (or `QS` / `QSC` / `GetQuerySetWithContext`), taking a model instance pointer as its argument.

For quick CRUD calls, `queries` also provides top-level helper functions, ContentTypes adapters (`CT_*`), raw statement execution helpers (`QueryRow`, `QueryRows`, `QueryExec`), validation/clause utilities, and compiler registration functions.

See [Defining Models](./models/models.md) for model setup and [Getting Started](./getting_started.md) for database initialization.

---

## Helper Functions

The following top-level functions are exported by `github.com/Nigel2392/go-django/queries/src`:

- `ListObjectsByIDs[T attrs.Definer, T2 any](object T, offset, limit uint64, ids []T2) ([]T, error)`
- `ListObjects[T attrs.Definer](object T, offset, limit uint64, ordering ...string) ([]T, error)`
- `GetObject[T attrs.Definer](object T, identifier any) (T, error)`
- `CountObjects[T attrs.Definer](obj T) (int64, error)`
- `SaveObject[T attrs.Definer](obj T) error`
- `CreateObject[T attrs.Definer](obj T) error`
- `UpdateObject[T attrs.Definer](obj T) (int64, error)`
- `DeleteObject[T attrs.Definer](obj T) (int64, error)`

### Retrieving Objects

```go
// Fetch by primary key
todo, err := queries.GetObject(&Todo{}, 1)

// List with offset, limit, and ordering
todos, err := queries.ListObjects(&Todo{}, 0, 100, "-ID")

// List by primary key slice
ids := []int{1, 2, 3}
subset, err := queries.ListObjectsByIDs(&Todo{}, 0, 1000, ids)

// Count all rows
count, err := queries.CountObjects(&Todo{})
```

### Saving, Creating, Updating, and Deleting

```go
// Create
todo := &Todo{Title: "New Task", Done: false}
err := queries.CreateObject(todo) // Populates todo.ID when driver supports returning

// Update
todo.Title = "Updated Title"
updatedCount, err := queries.UpdateObject(todo)

// Save (CreateObject if primary key is zero, otherwise UpdateObject)
err = queries.SaveObject(todo)

// Delete
deletedCount, err := queries.DeleteObject(todo)
```

`UpdateObject` and `DeleteObject` also invoke `django/models.SaveModel` / `django/models.DeleteModel` first if the object implements Go-Django's `models.Saver`/`models.Deleter` interfaces.

### Helper Function Caveats

- **`context.Background()` internally:** None of the top-level helpers (`GetObject`, `ListObjects`, `ListObjectsByIDs`, `CountObjects`, `SaveObject`, `CreateObject`, `UpdateObject`, `DeleteObject`) accept a `context.Context` argument — they internally construct `QuerySet`s bound to `context.Background()`. Whenever you need request cancellation, timeouts, query metrics (`drivers.ContextWithQueryInfo`), or transactions (`RunInTransaction` / `StartTransaction`), use `queries.GetQuerySetWithContext(ctx, obj)` (or `queries.QSC(ctx, obj)`) instead.
- **`SaveObject` zero-PK check:** `SaveObject` chooses between `CreateObject` and `UpdateObject` solely by checking `fields.IsZero(primaryValue)`. If a new model has a pre-generated non-zero primary key (such as `drivers.UUID` or `drivers.ULID`), `SaveObject` will attempt `UpdateObject` instead of inserting. Call `queries.CreateObject` (or `GetQuerySet(obj).Create(obj)`) for new rows with client-assigned PKs.

---

## ContentType Adapters (`CT_*`)

For integrating models with Go-Django's [ContentTypes](../../docs/contenttypes.md), `queries` provides three factory functions (automatically wired by `queries`'s `attrs.OnBeforeModelRegister` hook):

- `CT_GetObject(obj attrs.Definer) func(ctx context.Context, identifier any) (interface{}, error)`
- `CT_ListObjects(obj attrs.Definer) func(ctx context.Context, amount, offset uint) ([]interface{}, error)`
- `CT_ListObjectsByIDs(obj attrs.Definer) func(ctx context.Context, i []interface{}) ([]interface{}, error)`

Note that these adapters delegate to `GetObject`, `ListObjects`, and `ListObjectsByIDs`, so they also execute with `context.Background()` internally (and `CT_ListObjectsByIDs` defaults to `offset=0, limit=1000`).

---

## Raw Statement Helpers (`QueryRow`, `QueryRows`, `QueryExec`)

When you want to execute raw SQL while still using model/field placeholder substitution (`expr.ParseExprStatement`), connection routing, rebinding (`?` -> `$1`), and context transactions:

```go
func QueryRow(ctx context.Context, model attrs.Definer, sqlStr string, args ...interface{}) drivers.SQLRow
func QueryRows(ctx context.Context, model attrs.Definer, sqlStr string, args ...interface{}) (drivers.SQLRows, error)
func QueryExec(ctx context.Context, model attrs.Definer, sqlStr string, args ...interface{}) (sql.Result, error)
```

```go
row := queries.QueryRow(
	ctx,
	&Todo{},
	"SELECT ![ID], ![Title] FROM TABLE(SELF) WHERE ![Done] = ?[1]",
	false,
)
var id int64
var title string
if err := row.Scan(&id, &title); err != nil {
	return err
}
```

See [Expressions — Statement Parser](./expressions/expressions.md) for placeholder syntax (`![Field]`, `#[pkg.Model.Field]`, `TABLE(SELF)`, `?[1]`, `EXPR(...)`).

---

## Validation, Clause Generation, and Compilers

- `queries.Validate(ctx context.Context, validate any) error`: Calls `validate.(ContextValidator).Validate(ctx)` if implemented and returns any validation error.
- `queries.GenerateObjectsWhereClause[T attrs.Definer](objects ...T) ([]expr.ClauseExpression, error)`: Generates a `WHERE` clause matching `objects` by primary key (`pk = val` or `pk__in ids`), or falls back to unique / `UniqueTogether` fields registered on model metadata for PK-less models.
- `queries.GetUniqueKey(ctx context.Context, modelObject any) (any, error)`: Computes a deterministic key from primary key, through-model `(source, target)`, or unique fields (`errors.NoUniqueKey` if none are populated).
- `queries.Resolver(ctx context.Context, model attrs.Definer, database ...string) *resolver.Resolver`: Creates a standalone `resolver.Resolver` bound to the model's database compiler.
- `queries.RegisterCompiler(driver driver.Driver, compiler func(defaultDB string) QueryCompiler)` and `queries.Compiler(defaultDB string) QueryCompiler`: Registers or retrieves the SQL `QueryCompiler` for a database driver (`APPVAR_DATABASE` when `defaultDB == ""`). Importing `queries/src` automatically registers compilers for `DriverSQLite`, `DriverPostgres`, `DriverMariaDB`, and `DriverMySQL`.

---

## Next Steps

- Read [The QuerySet API](./queryset/queryset.md) for the complete method reference and runtime caveats.
- Read [Writing Queries](./queryset/writing_queries.md) for practical examples of relations, preloading, subqueries, transactions, and raw SQL.
