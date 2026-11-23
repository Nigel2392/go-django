# Interfaces Reference

This reference documents the interfaces, actors, signals, hooks, and compiler/resolver internals defined in `queries/src`, `queries/src/drivers`, `queries/src/resolver`, and `queries/src/alias`.

For core attribute and field interfaces (`attrs.Definer`, `attrs.Field`, `attrs.FieldDefinition`, `attrs.Binder`, `attrs.RelationValue`, etc.), see [Attrs Interfaces](../../docs/attrs/interfaces.md).

---

## Field Interfaces

### `AliasField`

Used to alias a field in SQL (for example, query annotations). Aliased from `expr.AliasField`.

```go
type AliasField interface {
	attrs.Field
	Alias() string
}
```

### `VirtualField`

Marks a field whose SQL expression is generated dynamically rather than mapping directly to a table column. Aliased from `expr.VirtualField`.

```go
type VirtualField interface {
	attrs.FieldDefinition
	SQL(chain []string, inf *expr.ExpressionInfo) (string, []any)
}
```

### `RelatedField`

Marks a field as pointing to another model, exposing its target field and optional reverse name.

```go
type RelatedField interface {
	attrs.Field
	GetTargetField() attrs.Field
	RelatedName() string
}
```

### `ForUseInQueriesField`

Controls whether a field is automatically included in `SELECT *` queries (`ForSelectAll(f)`). Fields without a column name (unless they implement `AliasField`) and embedded fields are already excluded by `ForSelectAll`.

```go
type ForUseInQueriesField interface {
	attrs.Field
	ForSelectAll() bool
}
```

### `ForDBEditableField`

Controls whether a field with a column name is included in `INSERT` and `UPDATE` column lists (`ForDBEdit(f)`).

```go
type ForDBEditableField interface {
	attrs.Field
	AllowDBEdit() bool
}
```

### `SaveableField`

Field whose value must be persisted before the parent model is saved (allowing the parent model to reference its primary key or updated state).

```go
type SaveableField interface {
	attrs.Field
	Save(ctx context.Context) error
}
```

### `SaveableDependantField`

Field that must be persisted after the parent model is saved so it can read the parent's primary key or state.

```go
type SaveableDependantField interface {
	attrs.Field
	Save(ctx context.Context, parent attrs.Definer) error
}
```

### `TargetClauseField` and `TargetClauseThroughField`

Custom JOIN clause generators used by relation and proxy fields:

```go
type TargetClauseField interface {
	GenerateTargetClause(qs expr.FieldResolver, lhs ClauseTarget, rhs ClauseTarget) JoinDef
}

type TargetClauseThroughField interface {
	GenerateTargetThroughClause(qs expr.FieldResolver, lhs ClauseTarget, through ThroughClauseTarget, rhs ClauseTarget) (JoinDef, JoinDef)
}
```

### `ProxyField` and `ProxyThroughField`

Marks a field as a proxy relation whose JOIN is automatically added to queries even when the field is not explicitly selected. See [Proxy Models](./models/proxy_models.md).

```go
type ProxyField interface {
	attrs.FieldDefinition
	TargetClauseField
	IsProxy() bool
}

type ProxyThroughField interface {
	attrs.FieldDefinition
	TargetClauseThroughField
	IsProxy() bool
}
```

---

## Model Interfaces

### `CanSetup`

Implemented by models (such as `models.Model`) that initialize embedded fields or internal state when constructed or prepared for queries.

```go
type CanSetup interface {
	Setup(ctx context.Context, value attrs.Definer) error
}
```

### `ContextValidator`

Implemented by models or fields that perform validation before database writes (`queries.Validate(ctx, validate)`).

```go
type ContextValidator interface {
	Validate(ctx context.Context) error
}
```

### `DataModel` and `ModelDataStore`

Stores non-struct annotations and cached relation objects on a model instance:

```go
type ModelDataStore interface {
	HasValue(key string) bool
	GetValue(key string) (any, bool)
	SetValue(key string, value any) error
	DeleteValue(key string) error
}

type DataModel interface {
	DataStore() ModelDataStore
}
```

### `Annotator`

Receives computed annotation columns scanned from a query row:

```go
type Annotator interface {
	Annotate(annotations map[string]any)
}
```

### `ThroughModelSetter`

Receives the through-table model instance when loading a `OneToOne` or `ManyToMany` relation with a through model:

```go
type ThroughModelSetter interface {
	SetThroughModel(throughModel attrs.Definer)
}
```

### `ForUseInQueries`

Allows a model to opt out of automatic `models.SaveObject` and `models.DeleteObject` goldcrest hooks when `ForUseInQueries()` returns `false`:

```go
type ForUseInQueries interface {
	attrs.Definer
	ForUseInQueries() bool
}
```

### `UniqueTogetherDefiner`

Aliased from `migrator.UniqueTogetherDefiner`. Declares multi-column unique constraints and allows models without a primary key to generate `WHERE` clauses and unique cache keys:

```go
type UniqueTogetherDefiner interface {
	attrs.Definer
	UniqueTogether() [][]string
}
```

### `OrderByDefiner`

Aliased from `resolver.OrderByDefiner`. Overrides default ordering (`[pk]` if a primary key exists) on new `QuerySet` instances:

```go
type OrderByDefiner interface {
	attrs.Definer
	OrderBy() []string
}
```

### `QuerySetDefiner`, `TypedQuerySetDefiner[T]`, `QuerySetChanger`, and `QuerySetDatabaseDefiner`

Customize which `*QuerySet` or database connection a model uses:

```go
type QuerySetDefiner interface {
	attrs.Definer
	GetQuerySet() *QuerySet[attrs.Definer]
}

type TypedQuerySetDefiner[T attrs.Definer] interface {
	attrs.Definer
	GetQuerySet() *QuerySet[T]
}

type QuerySetChanger interface {
	attrs.Definer
	ChangeQuerySet(qs *QuerySet[attrs.Definer]) *QuerySet[attrs.Definer]
}

type QuerySetDatabaseDefiner interface {
	attrs.Definer
	QuerySetDatabase() string
}
```

---

## Relation Interfaces

These interfaces bridge relation field values (`RelFK`, `RelRevFK`, `RelO2O`, `RelM2M`) and the `QuerySet` scanner/saver:

```go
type Relation = attrs.ValueRelation

type ParentInfo struct {
	Object attrs.Definer
	Field  attrs.Field
}

type RelationValue interface {
	attrs.Binder
	attrs.RelationValue
	ParentInfo() *ParentInfo
}

type MultiRelationValue interface {
	attrs.Binder
	attrs.MultiRelationValue
	ParentInfo() *ParentInfo
}

type ThroughRelationValue interface {
	attrs.Binder
	attrs.ThroughRelationValue
	ParentInfo() *ParentInfo
}

type MultiThroughRelationValue interface {
	attrs.Binder
	attrs.MultiThroughRelationValue
	ParentInfo() *ParentInfo
}
```

---

## QuerySet and Query Execution Interfaces

### `AbstractQuerySet` and Base QuerySet Interfaces

While `*queries.QuerySet[T]` is a concrete generic struct pointer, generic wrappers and custom QuerySets (such as `WrappedQuerySet` or `SpecificQuerySet`) compose the following interfaces:

```go
type AbstractQuerySet[MODEL any, QS AbstractQuerySet[MODEL, QS]] interface {
	expr.ExpressionBuilder
	expr.FieldResolver
	Peek() expr.QueryInformation

	Clone() QS
	Distinct() QS
	Select(fields ...any) QS
	Preload(fields ...any) QS
	Filter(key interface{}, vals ...interface{}) QS
	GroupBy(fields ...any) QS
	Limit(n int) QS
	Offset(n int) QS
	OrderBy(fields ...string) QS
	Reverse() QS
	ExplicitSave() QS
	Annotate(aliasOrAliasMap interface{}, exprs ...expr.Expression) QS

	GetOrCreateTransaction() (tx drivers.Transaction, err error)
	StartTransaction(ctx context.Context) (drivers.Transaction, error)
	WithTransaction(tx drivers.Transaction) (drivers.Transaction, error)

	ForUpdate() QS
	Prefix(prefix string) QS
	DB() drivers.DB
	Meta() expr.ModelMeta
	Compiler() QueryCompiler
	Having(key interface{}, vals ...interface{}) QS
	LatestQuery() QueryInfo
	WithContext(ctx context.Context) QS

	QueryAll(fields ...any) CompiledRowsQuery[[][]interface{}]
	QueryAggregate() CompiledRowsQuery[[][]interface{}]
	QueryCount() CompiledRowQuery[int64]
}
```

Specialized capability interfaces extend `AbstractQuerySet[T, QS]`:

- `BaseReadQuerySet[T, QS]`: `All()`, `IterAll()`, `Exists()`, `Count()`, `First()`, `Last()`, `Get()`
- `BaseValuesQuerySet[T, QS]`: `Values(fields ...any)`, `ValuesList(fields ...any)`, `Aggregate(annotations map[string]expr.Expression)`
- `BaseBulkQuerySet[T, QS]`: `BulkCreate(objects []T)`, `BatchCreate(objects []T)`, `BulkUpdate(params ...any)`, `BatchUpdate(params ...any)`
- `BaseWriteQuerySet[T, QS]`: `Create(value T)`, `Update(value T, expressions ...any)`, `GetOrCreate(value T)`, `Delete(objects ...T)`
- `BaseRawQuerySet[T, QS]`: `Row(sqlStr string, args ...interface{})`, `Rows(sqlStr string, args ...interface{})`, `Exec(sqlStr string, args ...interface{})`
- `BaseQuerySet[T, QS]`: Embeds all five interfaces above.

### `PushClauseExpression` and `BulkUpdateQuerySetParam`

```go
type PushClauseExpression interface {
	Operator() expr.ExprOp
	Unwrap() expr.ClauseExpression
}

type BulkUpdateQuerySetParam interface {
	BuildUpdateInfo() ([]UpdateInfo, error)
}
```

- `PushClauseExpression`: Passed as the sole key argument to `Filter(PushFilterOR(clause))` or `Filter(PushFilterAND(clause))` to combine `clause` with existing `WHERE` conditions using `OR` or `AND`.
- `BulkUpdateQuerySetParam`: Returned by `BulkUpdateQuerySet(qs, params...)` when combining multiple QuerySets into a single `BulkUpdate` or `BatchUpdate` call.

### `QueryInfo` and Compiled Query Interfaces

```go
type QueryInfo interface {
	SQL() string
	Args() []any
	Model() attrs.Definer
	Compiler() QueryCompiler
}

type CompiledQuery[T1 any] interface {
	QueryInfo
	Exec() (T1, error)
}

type CompiledRowQuery[T1 any] interface {
	CompiledQuery[T1]
	drivers.SQLRow
}

type CompiledRowsQuery[T1 any] interface {
	Reset()
	CompiledQuery[T1]
	drivers.SQLRows
}

type IterableQuery[RESULT any] interface {
	CompiledQuery[[]RESULT]
	Iter() iter.Seq2[RESULT, error]
}
```

---

## DB and Compiler Interfaces

### `drivers.DB`, `Unwrapper`, `Transaction`, and `DatabaseSpecificTransaction`

See [Drivers](./drivers.md) for full details on connection management. Summary:

```go
type Unwrapper interface {
	Unwrap() any
}

type DB interface {
	Unwrap() any
	Driver() driver.Driver
	QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) SQLRow
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Transaction interface {
	DB
	Finished() bool
	Commit(context.Context) error
	Rollback(context.Context) error
}

type DatabaseSpecificTransaction interface {
	Transaction
	DatabaseName() string
}
```

### `QueryCompiler` and `RebindCompiler`

`QueryCompiler` compiles `QuerySetInternals` and `expr.FieldResolver` state into database-specific SQL statements:

```go
type QueryCompiler interface {
	resolver.ExpressionCompiler

	DatabaseName() string
	DB() drivers.DB
	Driver() driver.Driver

	QuoteString(s string) string
	QuoteIdentifier(s string) string
	PrepForLikeQuery(any) string
	FormatLookupCol(string, string) string
	LogicalOpRHS() map[expr.LogicalOp]func(rhs string, value []any) (string, []any)
	LookupOperatorsRHS() map[string]string
	LookupPatternOperatorsRHS() map[string]string
	Quote() (front string, back string)
	PrepareValue(field attrs.Field, value any) any
	Placeholder() string
	FormatColumn(aliasGen *alias.Generator, tableColumn *expr.TableColumn) (string, []any)

	SupportsReturning() drivers.SupportsReturningType
	SupportsUnionOrderByTableAlias() bool

	StartTransaction(ctx context.Context) (drivers.Transaction, error)
	WithTransaction(tx drivers.Transaction) (drivers.Transaction, error)
	Transaction() drivers.Transaction
	InTransaction() bool

	BuildSelectQuery(ctx context.Context, resolver expr.FieldResolver, internals *QuerySetInternals) CompiledRowsQuery[[][]interface{}]
	BuildCountQuery(ctx context.Context, resolver expr.FieldResolver, internals *QuerySetInternals) CompiledRowQuery[int64]
	BuildCreateQuery(ctx context.Context, resolver expr.FieldResolver, internals *QuerySetInternals, objects []UpdateInfo) CompiledQuery[[][]interface{}]
	BuildUpdateQuery(ctx context.Context, resolver expr.FieldResolver, internals *QuerySetInternals, objects []UpdateInfo) CompiledQuery[int64]
	BuildDeleteQuery(ctx context.Context, resolver expr.FieldResolver, internals *QuerySetInternals) CompiledQuery[int64]
}

type RebindCompiler interface {
	QueryCompiler
	Rebind(ctx context.Context, s string) string
}
```

`RebindCompiler` adapts `?` placeholders to backend-specific bind markers (such as `$1`, `$2` for Postgres).

---

## Lifecycle Actors, Signals, and Hooks

### Actor Interfaces (`Acts*`)

Models can implement any of the 9 lifecycle actor interfaces:

```go
type ActsAfterQuery interface {
	AfterQuery(ctx context.Context) error
}

type ActsBeforeSave interface {
	BeforeSave(ctx context.Context) error
}

type ActsAfterSave interface {
	AfterSave(ctx context.Context) error
}

type ActsBeforeCreate interface {
	BeforeCreate(ctx context.Context) error
}

type ActsAfterCreate interface {
	AfterCreate(ctx context.Context) error
}

type ActsBeforeUpdate interface {
	BeforeUpdate(ctx context.Context) error
}

type ActsAfterUpdate interface {
	AfterUpdate(ctx context.Context) error
}

type ActsBeforeDelete interface {
	BeforeDelete(ctx context.Context) error
}

type ActsAfterDelete interface {
	AfterDelete(ctx context.Context) error
}
```

Return `errors.NotImplemented` from an actor method to log a warning and skip it without aborting the operation.

### `ObjectActor` and `Actor`

`queries.Actor(obj attrs.Definer) *ObjectActor` inspects which actor interfaces `obj` implements and exposes helper methods:

- `BeforeSave(ctx)`, `AfterSave(ctx)`
- `BeforeCreate(ctx)`, `AfterCreate(ctx)`
- `BeforeUpdate(ctx)`, `AfterUpdate(ctx)`
- `BeforeDelete(ctx)`, `AfterDelete(ctx)`
- `AfterQuery(ctx)`
- `Fake(ctx, flags ...actorFlag)`: Marks actor flags (`FlagActsBeforeSave`, `FlagActsAfterSave`, `FlagActsBeforeCreate`, `FlagActsAfterCreate`, `FlagActsBeforeUpdate`, `FlagActsAfterUpdate`, `FlagActsBeforeDelete`, `FlagActsAfterDelete`, `FlagActsAfterQuery`) as already seen in `ctx` so duplicate calls are skipped.

Running `BeforeCreate`/`BeforeUpdate` automatically chains to `BeforeSave`, and `AfterCreate`/`AfterUpdate` automatically chains to `AfterSave`.

### Signals

`queries` exposes seven signals using `ModelSignal` (`type SignalSave = ModelSignal` is a deprecated alias):

```go
type ModelSignal struct {
	Context  context.Context
	Instance attrs.Definer
	Using    QueryCompiler
}

var (
	SignalModelInit       = signals.Get[ModelSignal]("queries.model.init")
	SignalPreModelSave    = signals.Get[ModelSignal]("queries.model.pre_save")
	SignalPostModelSave   = signals.Get[ModelSignal]("queries.model.post_save")
	SignalPreModelCreate  = signals.Get[ModelSignal]("queries.model.pre_create")
	SignalPostModelCreate = signals.Get[ModelSignal]("queries.model.post_create")
	SignalPreModelDelete  = signals.Get[ModelSignal]("queries.model.pre_delete")
	SignalPostModelDelete = signals.Get[ModelSignal]("queries.model.post_delete")
)
```

`queries` also connects `SignalPreModelSave` and `SignalPostModelSave` to `embedder.PreSave.Bind` and `embedder.PostSave.Bind`, and `models.SIGNAL_MODEL_SETUP` forwards model setup events to `SignalModelInit`.

### Actor and Signal Caveats

- **Signals are skipped when the corresponding `Acts*` interface is implemented:** In `ObjectActor.execute` (`queries/src/queryset_actors.go`), each save/create/delete case checks `if _, ok := s.obj.(Acts...); ok { return ... }` and only calls `s.sendSig(...)` if the model does **not** implement that actor interface.
- **`actsAfterSave` currently emits `SignalPreModelSave` instead of `SignalPostModelSave`:** In `queryset_actors.go`, the fallback branch for `actsAfterSave` calls `s.sendSig(SignalPreModelSave, ctx)` rather than `SignalPostModelSave`. Consequently, `SignalPostModelSave` is currently never emitted by `ObjectActor` during creates or updates.
- **No signals for `Update` or `AfterQuery`:** `actsBeforeUpdate`, `actsAfterUpdate`, and `actsAfterQuery` invoke only `ActsBeforeUpdate`, `ActsAfterUpdate`, and `ActsAfterQuery` respectively and do not emit any signal (though update still chains into `actsBeforeSave` / `actsAfterSave`).

### Package Hooks (`queries_init.go`)

Importing `queries/src` installs four global hooks (see [Hooks](../../docs/hooks.md)):

- `models.MODEL_SAVE_HOOK` (priority `0`): When `django/models.SaveObject(ctx, m)` is called (and `ForUseInQueries()` is not `false`), decides whether to `Create` or `Update` by checking `Saved() bool` (if implemented) or whether the primary key is nil/zero, then executes `GetQuerySetWithContext(ctx, m).ExplicitSave()...`.
- `models.MODEL_DELETE_HOOK` (priority `0`): When `django/models.DeleteObject(ctx, m)` is called (and `ForUseInQueries()` is not `false`), deletes by primary key if the primary key is non-nil and non-zero.
- `attrs.OnBeforeModelRegister`: Automatically populates `contenttypes.ContentTypeDefinition` (`GetInstance`, `GetInstances`, `GetInstancesByIDs`) using `CT_GetObject`, `CT_ListObjects`, and `CT_ListObjectsByIDs`.
- `attrs.OnModelRegister`: Stores `UniqueTogetherDefiner.UniqueTogether()` under `MetaUniqueTogetherKey`. If the model has no primary key, it verifies that at least one unique field or `UniqueTogether` group exists (returning `errors.FieldNotFound` otherwise) and stores the generated `WHERE`-clause builder on model metadata. (`attrs.OnThroughModelRegister` performs an analogous check and composite key builder for through models without primary keys.)

---

## Compiler and Resolver Internals

Two internal helper packages power query compilation and field path resolution:

### `queries/src/resolver`

```go
type OrderByDefiner interface {
	attrs.Definer
	OrderBy() []string
}

type ExpressionCompiler interface {
	ExpressionInfo(resolver expr.FieldResolver) *expr.ExpressionInfo
}

type ModelInfo struct {
	Primary  attrs.FieldDefinition
	Object   attrs.Definer
	Fields   []attrs.FieldDefinition
	Table    string
	Ordering []string
}

func NewModelInfo(model attrs.Definer) (ModelInfo, error)
func New(ctx context.Context, model attrs.Definer, compiler ExpressionCompiler) *Resolver
```

- `ModelInfo` implements `expr.ModelMeta` (`Model()`, `TableName()`, `PrimaryKey()`, `OrderBy()`). `NewModelInfo` extracts table/field metadata, reads `OrderByDefiner` (defaulting to `[primary.Name()]` if `OrderBy` is `nil` and a primary key exists), and calls `attrs.Setup`.
- `Resolver` implements `expr.FieldResolver`. `New(ctx, model, compiler)` constructs a `Resolver` bound to a model, context, `ExpressionCompiler`, and `*alias.Generator`.
- `(*Resolver).Resolve(fieldName string, inf *expr.ExpressionInfo)` splits dot-separated relation paths (for example, `"Author.Profile.Bio"`), calls `attrs.WalkRelationChain`, generates table aliases per hop via `alias.Generator`, and returns `(finalModel, finalField, *expr.TableColumn, err)`.
- `(*Resolver).ResolveExpression(e expr.Expression)` and `(*Resolver).ExpressionSQL(w io.Writer, e expr.Expression)` resolve and render expressions outside a full `QuerySet` (for example, during constraint compilation in `migrator`).

### `queries/src/alias`

```go
type Generator struct {
	Prefix string
}

func NewGenerator() *Generator
func (a *Generator) Clone() *Generator
func (a *Generator) GetTableAlias(currentTable string, chainorKey any) string
func (a *Generator) GetFieldAlias(tableAlias, alias string) string
func (a *Generator) Set(chainOrKey any, val string)
func (a *Generator) Mapping() map[string]string
```

- `GetTableAlias(currentTable, chainOrKey)` returns the unprefixed table name when `chainOrKey` is empty (the root model) and assigns deterministic `T_<tableName>`, `T1_<tableName>`, `T2_<tableName>`, ... aliases (prefixed with `Prefix + "_"` for subqueries/prefixed QuerySets) for joined relation chains.
- `GetFieldAlias(tableAlias, alias)` prevents column alias collisions across tables by qualifying duplicate aliases with `<tableAlias>_<alias>`.
