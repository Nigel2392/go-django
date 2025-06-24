# Interface Documentation

This document describes **all interfaces** used in the query system. The docstrings have been preserved where applicable. Interfaces are grouped by purpose.

---

## 🔌 Field Interfaces

### `AliasField`

Used to alias a field in SQL (e.g., annotations).

```go
type AliasField interface {
    attrs.Field
    Alias() string
}
```

### `VirtualField`

Field is rendered as SQL (e.g., expression fields, internal annotations).

```go
type VirtualField interface {
    SQL(inf *expr.ExpressionInfo) (string, []any)
}
```

### `RelatedField`

Marks a field as related to another model.

This allows for easily defining a reverse name or target field of the end-model.

```go
type RelatedField interface {
    attrs.Field
    GetTargetField() attrs.Field
    RelatedName() string
}
```

### `ForUseInQueriesField`

Controls inclusion in `SELECT *` queries.

The field will not be automatically included in `SELECT *` queries unless explicitly set to true.

```go
type ForUseInQueriesField interface {
    attrs.Field
    ForSelectAll() bool
}
```

### `SaveableField`

Field that can persist its value to the database.

This type of field will be saved before the model is saved, allowing it to persist its value  
and for the parent model to access it during the save operation.

```go
type SaveableField interface {
    attrs.Field
    Save(ctx context.Context) error
}
```

### `SaveableDependantField`

Field that saves with access to the parent model.

The parent model will be saved before this field, allowing it to access the parent model's state,
such as its primary key or other fields.

```go
type SaveableDependantField interface {
    attrs.Field
    Save(ctx context.Context, parent attrs.Definer) error
}
```

### `ProxyField`

Proxy to another field in queries.

If `IsProxy()` returns true, this field will be used to generate a JOIN clause in queries.

The join will always be automatically included in the query, even if the field is not selected.

Read more about [Proxy Models](./models/proxy_models.md).

```go
type ProxyField interface {
    attrs.FieldDefinition
    TargetClauseField
    IsProxy() bool
}
```

### `ProxyThroughField`

Proxy using a through model.

If `IsProxy()` returns true, this field will be used to generate JOIN clauses using a through model.

The join will always be automatically included in the query, even if the field is not selected.

Read more about [Proxy Models](./models/proxy_models.md).

```go
type ProxyThroughField interface {
    attrs.FieldDefinition
    TargetClauseThroughField
    IsProxy() bool
}
```

### `TargetClauseField`

A `TargetClauseField` is used to generate JOIN clauses for related fields in queries.

This allows for custom JOIN logic based on the field's definition and the target model, for example
if you need a content type to join to a model that has a polymorphic relation.

```go
type TargetClauseField interface {
    GenerateTargetClause(qs *QuerySet[attrs.Definer], internals *QuerySetInternals, lhs ClauseTarget, rhs ClauseTarget) JoinDef
}
```

### `TargetClauseThroughField`

Generates JOINs using a through table.

A `TargetClauseThroughField` is used to generate JOIN clauses for fields that use a through model.

This is useful for many-to-many relations or one-to-one relations with a through table, and allows
for custom JOIN logic based on the field's definition and the through model.

```go
type TargetClauseThroughField interface {
    GenerateTargetThroughClause(qs *QuerySet[attrs.Definer], internals *QuerySetInternals, lhs ClauseTarget, through ThroughClauseTarget, rhs ClauseTarget) (JoinDef, JoinDef)
}
```

---

## 📦 Model Interfaces

### `DataModel`

Stores annotations and relations.

```go
type DataModel interface {
    DataStore() ModelDataStore
}
```

### `ModelDataStore`

The underlying data store object itself for the model.

```go
type ModelDataStore interface {
    HasValue(key string) bool
    GetValue(key string) (any, bool)
    SetValue(key string, value any) error
    DeleteValue(key string) error
}
```

### `Annotator`

Receives annotations directly.

The annotations will be collected for the current row and then passed to the `Annotate` method.

```go
type Annotator interface {
    Annotate(annotations map[string]any)
}
```

### `ThroughModelSetter`

Sets a through model on the target model.

This is used for many-to-many relations or one-to-one relations with a through table, allowing the target
model to set the through model relation directly.

```go
type ThroughModelSetter interface {
    SetThroughModel(throughModel attrs.Definer)
}
```

### `ForUseInQueries`

A model can adhere to this interface to indicate that the queries package  
should not automatically save or delete the model to/from the database when  
`django/models.SaveObject()` or `django/models.DeleteObject()` is called.

```go
type ForUseInQueries interface {
    attrs.Definer
    ForUseInQueries() bool
}
```

### `UniqueTogetherDefiner`

Returns groups of fields that must be unique together.

These can in turn be used to generate a unique where clause in queries,
to target this specific model, might the primary key not be available.

```go
type UniqueTogetherDefiner interface {
    attrs.Definer
    UniqueTogether() [][]string
}
```

### `QuerySetDefiner`

Overrides the queryset returned when using `GetQuerySet()`.

```go
type QuerySetDefiner interface {
    attrs.Definer
    GetQuerySet() *QuerySet[attrs.Definer]
}
```

### `QuerySetChanger`

Dynamically alters queryset behavior when it is first set up.

Basically the `GetQuerySet`, but the queryset is already passed in and should be returned with modifications.

```go
type QuerySetChanger interface {
    attrs.Definer
    ChangeQuerySet(qs *QuerySet[attrs.Definer]) *QuerySet[attrs.Definer]
}
```

### `QuerySetDatabaseDefiner`

Specifies which DB to use for queries - this is the database that the compiler should target from `django.Global.settings`.

```go
type QuerySetDatabaseDefiner interface {
    attrs.Definer
    QuerySetDatabase() string
}
```

---

## 📈 Relation Interfaces

The following interfaces are used to represent relations between models, either directly or through a through model.

The interfaces are designed to work with the `QuerySet` to allow the `QuerySet` to easily set and retrieve related model values.

### `Relation`

Represents a model-to-model relation with a through model.

These will contain the instances of the models involved in the relation, and can be used to access the related models.

```go
type Relation interface {
    Model() attrs.Definer
    Through() attrs.Definer
}
```

### `RelationValue`

For fields that reference a single object (e.g., ForeignKey, OneToOne without a through table).

```go
type RelationValue interface {
    attrs.Binder
    ParentInfo() *ParentInfo
    GetValue() attrs.Definer
    SetValue(attrs.Definer)
}
```

### `MultiRelationValue`

For fields that reference multiple objects (e.g., OneToMany, ManyToMany where the through table is not important).

```go
type MultiRelationValue interface {
    attrs.Binder
    ParentInfo() *ParentInfo
    GetValues() []attrs.Definer
    SetValues([]attrs.Definer)
}
```

### `ThroughRelationValue`

Represents a one-to-one relationship with a through model.

```go
type ThroughRelationValue interface {
    attrs.Binder
    ParentInfo() *ParentInfo
    GetValue() (attrs.Definer, attrs.Definer)
    SetValue(attrs.Definer, attrs.Definer)
}
```

### `MultiThroughRelationValue`

Represents a many-to-many relationship with a through model.

```go
type MultiThroughRelationValue interface {
    attrs.Binder
    ParentInfo() *ParentInfo
    GetValues() []Relation
    SetValues([]Relation)
}
```

---

## 🧠 Query Interfaces

### `QueryInfo`

Describes a compiled query.

```go
type QueryInfo interface {
    SQL() string
    Args() []any
    Model() attrs.Definer
    Compiler() QueryCompiler
}
```

### `CompiledQuery[T]`

Executes a query.

```go
type CompiledQuery[T any] interface {
    QueryInfo
    Exec() (T, error)
}
```

#### Aliases

```go
type CompiledCountQuery      = CompiledQuery[int64]
type CompiledExistsQuery     = CompiledQuery[bool]
type CompiledValuesListQuery = CompiledQuery[[][]any]
```

---

## 🔧 DB + Compiler Interfaces

### `DB`

Compatible with `*sql.DB`, `*sql.Tx`.

```go
type DB interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

### `Transaction`

Transaction wrapper for `DB`.

```go
type Transaction interface {
    DB
    Commit() error
    Rollback() error
}
```

### `DatabaseSpecificTransaction`

Transaction that tracks its DB name.

```go
type DatabaseSpecificTransaction interface {
    Transaction
    DatabaseName() string
}
```

### `QueryCompiler`

Compiles queries into SQL.

```go
// A QueryCompiler interface is used to compile a query.
//
// It should be able to generate SQL queries and execute them.
//
// It does not need to know about the model nor its field types.
type QueryCompiler interface {
    // DatabaseName returns the name of the database connection used by the query compiler.
    //
    // This is the name of the database connection as defined in the django.Global.Settings object.
    DatabaseName() string

    // DB returns the database connection used by the query compiler.
    //
    // If a transaction was started, it will return the transaction instead of the database connection.
    DB() DB

    // Quote returns the quotes used by the database.
    //
    // This is used to quote table and field names.
    // For example, MySQL uses backticks (`) and PostgreSQL uses double quotes (").
    Quote() (front string, back string)

    // Placeholder returns the placeholder used by the database for query parameters.
    // This is used to format query parameters in the SQL query.
    // For example, MySQL uses `?` and PostgreSQL uses `$1`, `$2` (but can support `?` as well).
    Placeholder() string

    // FormatColumn formats the given field column to be used in a query.
    // It should return the column name with the quotes applied.
    // Expressions should use this method to format the column name.
    FormatColumn(tableColumn *expr.TableColumn) (string, []any)

    // SupportsReturning returns the type of returning supported by the database.
    // It can be one of the following:
    //
    // - SupportsReturningNone: no returning supported
    // - SupportsReturningLastInsertId: last insert id supported
    // - SupportsReturningColumns: returning columns supported
    SupportsReturning() drivers.SupportsReturningType

    // StartTransaction starts a new transaction.
    StartTransaction(ctx context.Context) (Transaction, error)

    // WithTransaction wraps the transaction and binds it to the compiler.
    WithTransaction(tx Transaction) (Transaction, error)

    // Transaction returns the current transaction if one is active.
    Transaction() Transaction

    // InTransaction returns true if the current query compiler is in a transaction.
    InTransaction() bool

    // PrepForLikeQuery prepares a value for a LIKE query.
    //
    // It should return the value as a string, with values like `%` and `_` escaped
    // according to the database's LIKE syntax.
    PrepForLikeQuery(v any) string

    // BuildSelectQuery builds a select query with the given parameters.
    BuildSelectQuery(
        ctx context.Context,
        qs *QuerySet[attrs.Definer],
        internals *QuerySetInternals,
    ) CompiledQuery[[][]interface{}]

    // BuildCountQuery builds a count query with the given parameters.
    BuildCountQuery(
        ctx context.Context,
        qs *QuerySet[attrs.Definer],
        internals *QuerySetInternals,
    ) CompiledQuery[int64]

    // BuildCreateQuery builds a create query with the given parameters.
    BuildCreateQuery(
        ctx context.Context,
        qs *QuerySet[attrs.Definer],
        internals *QuerySetInternals,
        objects []*FieldInfo[attrs.Field],
        values []any,
    ) CompiledQuery[[][]interface{}]

    BuildUpdateQuery(
        ctx context.Context,
        qs *GenericQuerySet,
        internals *QuerySetInternals,
        objects []UpdateInfo,
    ) CompiledQuery[int64]

    // BuildUpdateQuery builds an update query with the given parameters.
    BuildDeleteQuery(
        ctx context.Context,
        qs *QuerySet[attrs.Definer],
        internals *QuerySetInternals,
    ) CompiledQuery[int64]
}
```
