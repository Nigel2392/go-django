# The QuerySet API

`*queries.QuerySet[T]` is a generic struct pointer (`type QuerySet[T attrs.Definer] struct`) representing a composable database query for model type `T`.

Note that `*QuerySet[T]` is a **concrete struct pointer**, not an interface (functions accepting generic querysets or custom wrappers use `queries.AbstractQuerySet`, `queries.BaseReadQuerySet`, or `queries.BaseQuerySet` from [Interfaces](../interfaces.md)).

---

## Constructors and Database Helpers

```go
func GetQuerySet[T attrs.Definer](model T) *QuerySet[T]
func QS[T attrs.Definer](model T) *QuerySet[T]
func GetQuerySetWithContext[T attrs.Definer](ctx context.Context, model T) *QuerySet[T]
func QSC[T attrs.Definer](ctx context.Context, model T) *QuerySet[T]
func Objects[T attrs.Definer](model T, database ...string) *QuerySet[T]
func ChangeObjectsType[OldT, NewT attrs.Definer](qs *QuerySet[OldT]) *QuerySet[NewT]
func GetDatabaseName(model attrs.Definer) string
```

- `GetQuerySet(model)` (alias `QS(model)`): Respects `QuerySetDefiner` and `TypedQuerySetDefiner[T]` if implemented on `model` (cloning the returned queryset), otherwise falls back to `Objects(model)`.
- `GetQuerySetWithContext(ctx, model)` (alias `QSC(ctx, model)`): Equivalent to `GetQuerySet(model).WithContext(ctx)`. Panics if `ctx == nil`.
- `Objects(model, database...)`: Constructs the default `*QuerySet[T]` directly (used inside custom `GetQuerySet()` implementations to avoid recursion). Applies default primary-key ordering (or `OrderByDefiner`), sets default `Limit` to `MAX_DEFAULT_RESULTS` (`1000`), resolves database via `QuerySetDatabaseDefiner` or `APPVAR_DATABASE`, and runs `QuerySetChanger` if implemented.
- `ChangeObjectsType[OldT, NewT](qs)`: Re-casts `*QuerySet[OldT]` to `*QuerySet[NewT]` without cloning (panics with `errors.TypeMismatch` if underlying model does not implement `NewT`).

---

## Inspection, Context, and Transaction Methods

### `Meta() expr.ModelMeta`

Returns model metadata (`Model()`, `TableName()`, `PrimaryKey()`, `OrderBy()`). Use `qs.Meta().Model()` to access the underlying `attrs.Definer` prototype (there is no `Model()` method on `*QuerySet[T]`).

### `Compiler() QueryCompiler` and `DB() drivers.DB`

- `Compiler()` returns the `QueryCompiler` bound to the queryset.
- `DB()` returns the active `drivers.Transaction` if inside a transaction, or the underlying `drivers.Database`.

### `LatestQuery() QueryInfo` and `Peek() expr.QueryInformation`

- `LatestQuery()` returns the last compiled/executed `QueryInfo` (`SQL()`, `Args()`, `Model()`, `Compiler()`).
- `Peek()` returns a read-only `expr.QueryInformation` snapshot of selected fields, joins, `Where`, `Having`, `GroupBy`, `OrderBy`, `Unions`, `Limit`, `Offset`, `ForUpdate`, and `Distinct`.

### `GoString() string` and `String() string`

- `GoString()` formats the QuerySet configuration (`model`, `fields`, `where`, `orderBy`, `limit`, `offset`, `joins`, etc.) without hitting the database.
- `String()` **executes a database query** (clones `qs`, sets `Limit` to `MAX_GET_RESULTS` (`21`), calls `All()`, and formats the resulting models). Avoid calling `fmt.Println(qs)` or logging `%s` on a QuerySet unless a DB query is intended; use `%#v` (`GoString()`) for debug output.

### `Clone() *QuerySet[T]`

Creates a shallow-cloned `*QuerySet[T]` with copied `QuerySetInternals` slices/maps and `AliasGen`, clearing any cached results (`qs.cached`).

### `Context() context.Context` and `WithContext(ctx context.Context) *QuerySet[T]`

- `Context()` returns the bound context (defaulting to `context.Background()`).
- `WithContext(ctx)` binds `ctx` on `qs` **in place**. If `ctx` carries an active transaction for the same database (`transactionFromContext`), `WithContext` binds it onto `qs.compiler` automatically. Panics if `ctx == nil`.

### `Prefix(prefix string) *QuerySet[T]`

Sets `qs.AliasGen.Prefix` in place to namespace table aliases (useful in subqueries or custom SQL joins).

### `StartTransaction`, `WithTransaction`, and `GetOrCreateTransaction`

```go
func (qs *QuerySet[T]) StartTransaction(ctx context.Context) (drivers.Transaction, error)
func (qs *QuerySet[T]) WithTransaction(tx drivers.Transaction) (drivers.Transaction, error)
func (qs *QuerySet[T]) GetOrCreateTransaction() (tx drivers.Transaction, err error)
```

- `StartTransaction(ctx)` begins a transaction via `qs.compiler.StartTransaction(ctx)`, stores it in `qs.context`, and returns `(drivers.Transaction, error)`.
- `WithTransaction(tx)` binds an existing `drivers.Transaction` to `qs.compiler` and `qs.context`.
- `GetOrCreateTransaction()` starts a transaction if `!InTransaction()` and `QUERYSET_CREATE_IMPLICIT_TRANSACTION` (`true` by default) is enabled; otherwise returns a no-op `NullTransaction`.

### Package-Level Transaction and Commit Context Helpers

```go
func StartTransaction(ctx context.Context, database ...string) (context.Context, drivers.DatabaseSpecificTransaction, error)
func RunInTransaction[T attrs.Definer](c context.Context, fn func(ctx context.Context, NewQuerySet ObjectsFunc[T]) (commit bool, err error), database ...string) error
func IsTransactionContext(ctx context.Context) bool

func CommitContext(ctx context.Context, canCommit bool) context.Context
func IsCommitContext(ctx context.Context) bool
```

- `queries.StartTransaction(ctx, db...)`: Starts a transaction stored on the returned `ctx` (or returns a no-op nested transaction if `ctx` already holds one for that database or `IsCommitContext(ctx)` is `false`). Any `QuerySet` receiving that `ctx` via `WithContext` / `QSC` automatically participates in the transaction.
- `queries.RunInTransaction[T](ctx, fn, db...)`: Starts a transaction, passes `ctx` and a `NewQuerySet(model)` factory (panicking with `errors.CrossDatabaseTransaction` if a model targets a different database), commits if `commit == true && err == nil`, and rolls back otherwise.
- `queries.CommitContext(ctx, false)`: Disables commits and database writes (`BulkCreate`, `BulkUpdate`, `Delete`, `StartTransaction`) while still running model setup, actors, and `Validate`.

---

## Query Builder Methods

Unless explicitly noted below, builder methods clone `qs` and return a new `*QuerySet[T]`:

### `Select(fields ...any) *QuerySet[T]`

Selects fields, relation paths, or expressions. Replaces previous field selections:

- No arguments or `"*"`: selects all `ForSelectAll` fields (including virtual `ExpressionField`s and embedded proxy chains).
- `nil` as single argument (`Select(nil)`): clears `Fields`, `Preload`, `Joins`, and `Annotations`.
- Field names & dot-notation relations: `Select("ID", "Title")`, `Select("*", "User.*")`, `Select("*", "User.Profile.Bio")`.
- `expr.Expression` / `expr.NamedExpression`: e.g. `Select("*", expr.As("TitleLower", expr.LOWER("Title")))`.

### `SelectRelated(fields ...string) *QuerySet[T]`

Eagerly `JOIN`s and selects all `ForSelectAll` columns for single-valued relations (`ForeignKey` / `OneToOne`) along dot-separated paths. Note that `SelectRelated` **modifies `qs` in place** (does not clone) and **panics** if any hop is `RelOneToMany` or `RelManyToMany` (use `Preload` for collection relations).

### `Preload(fields ...any) *QuerySet[T]`

Schedules separate preload queries for related models (especially `OneToMany` and `ManyToMany`, or nested relations). Accepts:

- `string`: relation path (`"TodoSet"`, `"Tags"`)
- `queries.NoJoins("TodoSet")`: explicitly avoids adding parent `JOIN`s for the preload path
- `queries.Preload{Path: "TodoSet", NoJoins: true, QuerySet: customQS}`: attaches a custom `*QuerySet[attrs.Definer]` to filter, order, or select within the preloaded query.

### `Filter(key interface{}, vals ...interface{}) *QuerySet[T]`

Adds `WHERE` conditions (joined by `AND`). Accepts:

- Field/lookup string + value(s): `Filter("Done", false)`, `Filter("User.Name__startswith", "Alice")` (note **dot notation** for relations; double-underscore `__` is only for lookups/transforms)
- `expr.Expression` / `expr.ClauseExpression` / `map[string]any`
- `queries.PushFilterOR(clause)` or `queries.PushFilterAND(clause)` (`PushClauseExpression`, must pass no `vals`): combines `clause` with the existing `WHERE` conditions using `OR` or `AND`.

There is **no `Exclude` method** on `QuerySet`. Negate conditions using `.Not(true)` on a `Q` / `ClauseExpression`:

```go
qs.Filter(expr.Q("Done", true).Not(true))
```

### `Having(key interface{}, vals ...interface{}) *QuerySet[T]`

Adds `HAVING` conditions for aggregate queries (same key/value forms as `Filter`).

### `GroupBy(fields ...any) *QuerySet[T]` and `ClearGroupBy() *QuerySet[T]`

Sets `GROUP BY` columns. Note that `GroupBy` **resets `OrderBy` to `nil`**, so call `OrderBy(...)` after `GroupBy(...)` if ordering is required.

### `OrderBy(fields ...string) *QuerySet[T]`, `Reverse() *QuerySet[T]`, and `ClearOrderBy() *QuerySet[T]`

- `OrderBy("Field1", "-Field2")` sets ordering (`-` prefix for `DESC`), replacing the default PK ordering.
- `Reverse()` flips `ASC`/`DESC` on every existing `OrderBy` term.
- `ClearOrderBy()` clears all ordering clauses.

### `Limit(n int) *QuerySet[T]` and `Offset(n int) *QuerySet[T]`

Sets SQL `LIMIT` and `OFFSET` (and batch size for `BatchCreate` / `BatchUpdate`). Remember `Objects()` initializes `Limit` to `MAX_DEFAULT_RESULTS` (`1000`); pass `Limit(0)` to omit `LIMIT` or a custom `n`.

### `Distinct() *QuerySet[T]`

Adds `SELECT DISTINCT` (and `UNION DISTINCT` on union branches). Does not panic.

### `ForUpdate() *QuerySet[T]`

Appends `FOR UPDATE` to `SELECT` queries for row-level locking inside transactions.

### `Union[T2 attrs.Definer](other *QuerySet[T2]) *QuerySet[T]`

Combines two QuerySets with `UNION` (`UNION DISTINCT` if `other` uses `Distinct()`). Both QuerySets must select the same number of fields with compatible `dbtype.Type` or Go assignability/convertibility (panics otherwise). Strips `OrderBy`, `Limit`, and `Offset` from `other`, while subsequent `Filter`, `Having`, `GroupBy`, `OrderBy`, and `Reverse` calls apply across all union branches.

### `Annotate(aliasOrAliasMap interface{}, exprs ...expr.Expression) *QuerySet[T]`

Adds computed fields (`string` alias + expression, `map[string]expr.Expression`, or `map[string]any`). Values appear in `Row[T].Annotations` (and `model.Annotations` / `DataStore()` when `models.Model` is embedded).

### `ExplicitSave() *QuerySet[T]`

Sets `qs.explicitSave = true` **in place**, telling `Create` / `Update` to bypass `models.ContextSaver.Save(ctx)` and execute SQL directly (essential inside `SaveObject` or actor hooks to prevent infinite recursion).

### `Scope(scopes ...func(*QuerySet[T], *QuerySetInternals) *QuerySet[T]) *QuerySet[T]`

Runs reusable scope functions and applies any returned changes **in place** back onto `qs`:

```go
func ActiveScope(qs *queries.QuerySet[*Todo], internals *queries.QuerySetInternals) *queries.QuerySet[*Todo] {
	return qs.Filter("Done", false)
}
```

### `ForEachRow(rowFunc func(qs *QuerySet[T], row *Row[T]) error) *QuerySet[T]`

Registers a per-row callback executed during `IterAll()` (and `All()`).

---

## Query Execution Methods

### Read and Iteration (`Rows`, `All`, `IterAll`, `Get`, `First`, `Last`, `Exists`, `Count`)

- `All() (Rows[T], error)`: Executes the query, runs `AfterQuery` actors and `Preload`s, deduplicates joined multi-relations, and returns `Rows[T]` (`[]*Row[T]`).
- `IterAll() (int, iter.Seq2[*Row[T], error], error)`: Compiles rows and returns an `iter.Seq2[*Row[T], error]` iterator (plus total row count and error).
- `Get() (*Row[T], error)`: Sets `Limit = MAX_GET_RESULTS` (`21`), executes `All()`, and returns the single row, `errors.NoRows` if 0 match, or `errors.MultipleRows` if >1 match.
- `First() (*Row[T], error)` / `Last() (*Row[T], error)`: Sets `Limit = 1` (with `Reverse()` for `Last()`) and returns the row or `errors.NoRows`.
- `Exists() (bool, error)`: Sets `Limit = 1, Offset = 0` and checks `COUNT(*) > 0`.
- `Count() (int64, error)`: Runs `QueryCount()` (`SELECT COUNT(*)`) and returns the count.

#### `Row[T]` and `Rows[T]` Helpers

```go
type Row[T any] struct {
	Object          T
	ObjectFieldDefs attrs.Definitions
	Through         attrs.Definer
	Annotations     map[string]any
}

type Rows[T any] []*Row[T]

func (r Rows[T]) Len() int
func (rows Rows[T]) Objects() iter.Seq[T]
func (rows Rows[T]) Pluck(pathToField string) iter.Seq2[int, attrs.Field]
func PluckRowValues[ValueT any, ModelT attrs.Definer](rows Rows[ModelT], pathToField string) iter.Seq2[int, ValueT]
```

- `rows.Objects()` yields each `row.Object`.
- `rows.Pluck("User.Name")` / `queries.PluckRowValues[string](rows, "User.Name")` traverses dot-separated paths across loaded relations (panics on `FieldNotFound` or type mismatch).

### Values and Aggregation

- `Values(fields ...any) ([]map[string]any, error)`: Returns one `map[string]any` per row.
- `ValuesList(fields ...any) ([][]interface{}, error)`: Returns row values as `[][]interface{}`.
- `IterValuesList(fields ...any) (preAlloc int, loop iter.Seq2[[]interface{}, error], err error)`: Streams rows without building preloads or relation deduplication.
- `Aggregate(annotations map[string]expr.Expression) (map[string]any, error)`: Clears `OrderBy`, `Limit`, `Offset`, `ForUpdate`, `Distinct`, and returns aggregate values from the first row.

### Low-Level Compiled Queries

- `QueryAll(fields ...any) CompiledRowsQuery[[][]interface{}]`
- `QueryAggregate() CompiledRowsQuery[[][]interface{}]`
- `QueryCount() CompiledRowQuery[int64]`

### Create, Update, Bulk, and Delete

- `Create(value T) (T, error)`: If `value` implements `models.ContextSaver` and `!explicitSave`, runs `BeforeCreate` -> `Save(...)` -> `AfterCreate`; otherwise delegates to `BulkCreate([]T{value})`.
- `GetOrCreate(value T) (T, bool, error)`: Requires at least one `WHERE` condition (panics with `errors.NoWhereClause` otherwise); attempts `Get()` and creates `value` if `errors.Is(err, errors.NoRows)`.
- `Update(value T, expressions ...any) (int64, error)`: If `value != nil`, implements `models.ContextSaver`, `len(Where) == 0`, and `!explicitSave`, delegates to `Save(...)`. Otherwise delegates to `BulkUpdate`. Note `Update` wraps errors with `errors.NoChanges`.
- `BulkCreate(objects []T) ([]T, error)`: Inserts `objects` in a single multi-row `INSERT`, running `Setup`, `ActsBeforeCreate`, `Validate`, `AutoNowAdd`/`AutoNow`, and `ActsAfterCreate`.
- `BatchCreate(objects []T) ([]T, error)`: Chunks `objects` using `qs.internals.Limit` (default `1000`, so there is **no `BatchSize` method** — call `.Limit(n)`) and calls `BulkCreate` per batch inside a transaction.
- `BulkUpdate(params ...any) (int64, error)`: Accepts any mix of `T`, `[]T`, `expr.NamedExpression` (e.g. `expr.As("Done", expr.Value(true))`), `map[string]expr.Expression`, `BulkUpdateQuerySet(qs, params...)`, `UpdateInfo`, `[]UpdateInfo`, `[]attrs.Definer`, and `[]any`.
- `BatchUpdate(params ...any) (int64, error)`: Chunks `params` by `qs.internals.Limit` and runs `BulkUpdate` per batch, returning `errors.NoChanges` if any batch updates 0 rows.
- `Delete(objects ...T) (int64, error)`: Deletes by `GenerateObjectsWhereClause(objects...)` if `objects` are given (running `BeforeDelete`/`AfterDelete`), or deletes rows matching the QuerySet's `WHERE` clause.

### Raw SQL on QuerySet

`Row`, `Rows`, `Exec`, and `ParseExpression` resolve statement placeholders (`![Field]`, `#[pkg.Model.Field]`, `TABLE(SELF)`, `?[1]`, `EXPR(...)`), rebind `?` placeholders for the active driver, and run against `qs.compiler.DB()` (so they respect active transactions):

```go
func (qs *QuerySet[T]) ParseExpression(sqlStr string, args ...interface{}) (string, []interface{})
func (qs *QuerySet[T]) Rows(sqlStr string, args ...interface{}) (drivers.SQLRows, error)
func (qs *QuerySet[T]) Row(sqlStr string, args ...interface{}) drivers.SQLRow
func (qs *QuerySet[T]) Exec(sqlStr string, args ...interface{}) (sql.Result, error)
```

### Subquery Expression (`queries.Subquery`)

```go
func Subquery[T attrs.Definer, QS BaseQuerySet[T, QS]](qs QS) expr.Expression
```

Wraps any `BaseQuerySet` in a subquery expression (marking subquery context and capturing parent `ExpressionInfo` so `expr.OuterRef` resolves outer columns). Note `QuerySet[T]` also implements `expr.ExpressionBuilder` (`BuildExpression()`), which strips `Limit`/`Offset`.

---

## Wrapping QuerySets (`WrappedQuerySet` and `CTEQuerySet`)

Use `WrapQuerySet` and `WrappedQuerySet[T, CONV, ORIG]` to build typed custom QuerySet wrappers (such as [`specific.SpecificQuerySet`](../models/specific_models.md)) that preserve immutability via `QuerySetCanClone`, `QuerySetCanBeforeExec`, `QuerySetCanAfterExec`, and `attrs.CanSetup`.

`CTEQuerySet[T]` / `NewCTEQuerySet[T]` (`queries/src/queryset_wrapped_cte.go`) also exists as an **experimental / incomplete** wrapper for SQL `WITH` Common Table Expressions (`CTEQueryCompiler` and `Join`), and should be treated as experimental (note its `CloneQuerySet` drops `compiler` and `internals`).

---

## Caveats

- **Default `LIMIT 1000`:** `Objects()` initializes `Limit` to `MAX_DEFAULT_RESULTS` (`1000`). Call `.Limit(0)` if you want an unbounded query, or `.Limit(n)` to change page/batch sizes.
- **In-place mutation methods:** While `Filter`, `Select`, `OrderBy`, `Limit`, `Offset`, `Preload`, `GroupBy`, `Having`, `Annotate`, `Reverse`, `Distinct`, `ForUpdate`, `Union`, and `ForEachRow` clone the `QuerySet`, **`Scope`, `SelectRelated`, `WithContext`, `Prefix`, `ExplicitSave`, `Get`, `First`, `Last`, `Exists`, `QueryAll`, `QueryAggregate`, and `QueryCount` mutate `qs` in place**. Call `.Clone()` before branching or reusing a shared base `QuerySet`.
- **`qs.cached` reuse hazard:** `QUERYSET_USE_CACHE_DEFAULT` is `true` by default, and `Get()`, `Values()`, `ValuesList()`, and `Aggregate()` store heterogeneous concrete values (`*Row[T]`, `[]map[string]any`, `[][]any`, `map[string]any`) into the same `qs.cached` field. Calling e.g. `qs.Get()` and then `qs.All()` (or `qs.Values()` and then `qs.All()`) on the exact same uncloned `qs` pointer will panic on a type assertion (`*Row[T]` -> `[]*Row[T]`). Always clone or create a fresh `QuerySet` per execution.
- **`GroupBy` resets `OrderBy`:** `GroupBy()` explicitly clears `OrderBy = nil` (removing even default PK ordering). Always chain `.OrderBy(...)` **after** `.GroupBy(...)`.
- **Zero-value `Update` skip:** When `.Select(...)` was not called before `qs.Update(obj)` / `BulkUpdate(obj)`, `updateFields` skips any field whose Go value is `reflect.Value.IsZero()` (so `false`, `0`, or `""` will NOT be updated!). Pass `.Select("FieldToZero")` or `expr.As("FieldToZero", expr.Value(false))` to update fields to zero values.
- **`BulkCreate` MySQL returning caveat:** On MySQL (`SupportsReturningLastInsertId`), if the model has no primary key or its primary key is not auto-incrementing (`!attrs.CanAutoIncrement`), `BulkCreate` returns early before running `ActsAfterCreate` and does not populate DB-generated defaults. For auto-increment PKs, MySQL multi-row `BulkCreate` requires `multiStatements=true&interpolateParams=true` in the DSN and checks that `AllLastInsertIds()` matches `len(objects)` (failing with `errors.LastInsertId` otherwise).
- **`String()` executes a DB query:** Calling `qs.String()` runs `SELECT ... LIMIT 21` against the database. Use `qs.GoString()` (`%#v`) for zero-DB debugging.
- **Postgres `BuildUpdateQuery` caveats:** `postgresQueryBuilder.BuildUpdateQuery` (`queries/src/queryset_compiler_generic.go`) sends `pgx.Batch` directly through `g.queryInfo.DB` rather than `g.DB()` (which bypasses an active transaction on `g.transaction`), and calls `br.Exec()` only once instead of looping over all queued statements (so `RowsAffected()` and error checking reflect only the first statement in a multi-object batch update).
- **`RunInTransaction` panic swallowing:** `queries.RunInTransaction` recovers any panic inside `fn`, logs it, rolls back the transaction, and returns `nil` (the recovered panic is **not** propagated or returned as an error).
