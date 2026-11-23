# Queries Documentation

Welcome to the documentation for the `queries` package (`github.com/Nigel2392/go-django/queries`).

While [Go-Django](https://github.com/Nigel2392/go-django) does not force any database layer on you, `queries` provides a Django-style ORM for Go-Django when you want structured models, query composition, expressions, and migrations.

Do note that `queries` does not wrap `*sql.DB` directly; it defines its own `drivers.DB` interface and must be opened with `drivers.Open(ctx, driverName, dsn, opts...)`.

---

## Getting Started and Drivers

* [Getting Started](./getting_started.md) - Installing, configuring a database, wiring `migrator.NewAppConfig`, and running queries.
* [Interfaces](./interfaces.md) - Core ORM, query compiler, resolver, alias, actor, signal, and hook interfaces.
* [Drivers](./drivers.md) - Supported SQL drivers, ORM types, query logging, `dbtype`, and `securedb`.
* [Errors](./errors.md) - Driver-normalized sentinel errors (`NoRows`, `MultipleRows`, `UniqueViolation`, etc.) and error wrapping.

## Models, Fields, and Relations

* [Defining Models](./models/models.md) - Struct definitions, `FieldDefs(ctx)`, indexes, constraints, and validation.
* [The `models.Model` Base](./models/model.md) - Embedded base model, `Create`/`Update`/`Delete`/`Load`, dirty tracking, and lifecycle hooks.
* [Proxy Models](./models/proxy_models.md) - Extending or projecting models without creating new tables.
* [Specific Models](./models/specific_models.md) - Polymorphic base-to-concrete subtype loading (`queries/src/specific`).
* [Fields](./fields.md) - Using `fields` and `fields/formfields` for custom column types, relations, and forms.
* [Virtual Fields](./virtual_fields.md) - Computed SQL fields excluded from storage and writes.
* [Relations](./relations/relations.md) - Foreign keys, one-to-one, many-to-many, and dot-notation traversal.

## Querying the Database

* [Basic Querying](./querying.md) - Top-level helpers (`GetObject`, `CreateObject`, `CT_GetObject`, raw queries, compilers).
* [The QuerySet API](./queryset/queryset.md) - Concrete `*queries.QuerySet[T]`, `Filter`, `Select`, `SelectRelated`, `Preload`, transactions, subqueries, and raw queries.
* [Writing Complex Queries](./queryset/writing_queries.md) - End-to-end patterns for `expr.Q`, joins, preloading, subqueries, iterators, and transactions.

## Expressions, Lookups, and Constraints

* [Expressions](./expressions/expressions.md) - `Logical`, `F`/`Raw`, `As`, `Cast`, `OuterRef`, and SQL functions (`expr.COUNT`, `LOWER`, `COALESCE`, etc.).
* [Case Expressions](./expressions/cases.md) - Building SQL `CASE WHEN ... THEN ... ELSE ... END` expressions.
* [Lookups and Transforms](./expressions/lookups.md) - Filtering with `__lookup` suffixes, chained transforms (`__lower`), and custom registrations.
* [Constraints](./expressions/constraints.md) - Declarative `CheckConstraint`, `UniqueConstraint`, and `ForeignKeyConstraint` definitions.

## Migrations and HTTP Integration

* [Migrations](./migrations.md) - Schema snapshots, `.mig` generation, `migrator.AutoMigrate`, `migrator.NewAppConfig`, and `cmd/migrator`.
* [HTTP Integration](./http_integration.md) - Request transaction middleware (`TransactionMiddleware`) and generic detail views (`ObjectView[T]`).

---

## Quick Example

```go
// Query forward foreign key relation (Filter supports negation via .Not(true))
todos, err := queries.GetQuerySet(&Todo{}).
	Select("*", "User.*").
	Filter("Done", false).
	OrderBy("-ID").
	All()

// Negate a condition with .Not(true) (QuerySet has no Exclude method)
activeTodos, err := queries.GetQuerySet(&Todo{}).
	Filter(expr.Q("Done", true).Not(true)).
	All()

// Query reverse foreign key relation
users, err := queries.GetQuerySet(&User{}).
	Select("*", "TodoSet.*").
	Filter("TodoSet.Done", false).
	OrderBy("-ID").
	All()
```

### Tested Databases

* SQLite via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
* MySQL via [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) (tested against [dolthub/go-mysql-server](https://github.com/dolthub/go-mysql-server))
* MariaDB via [a custom driver](../src/drivers/drivers_mardiadb.go) wrapping `go-sql-driver/mysql` (use `"mariadb"` in `drivers.Open(...)`)
* PostgreSQL via [jackc/pgx](https://github.com/jackc/pgx) (use `"postgres"` in `drivers.Open(...)`)

#### Caveats

* MySQL and MariaDB require both `multiStatements=true` and `interpolateParams=true` in the DSN so batched statements and multi-row inserts can return multiple IDs. **Without these parameters, bulk inserts and bulk updates cannot work reliably.**

* The `"mariadb"` driver wraps [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) and adds `RETURNING` column support when creating records:

```go
db, err := drivers.Open(
	context.Background(),
	"mariadb",
	"user:password@tcp(localhost:3306)/dbname?multiStatements=true&interpolateParams=true",
)
if err != nil {
	log.Fatal(err)
}
```

### Supported Features

* Field selection, forward joins (`SelectRelated`), and batched relation preloading (`Preload`)
* Filtering with lookups, transforms, `Q` groups, `Logical` expressions, and `Subquery` / `EXISTS`
* Ordering, limiting, offsets, unions, distinct queries, and `SELECT ... FOR UPDATE`
* Annotations, aggregates, `CASE WHEN` expressions, and type casting
* Virtual fields, proxy models, and polymorphic specific models
* Table constraints (`CHECK`, `UNIQUE`, `FOREIGN KEY`) and schema migrations
* Transaction management, HTTP transaction middleware, and `ObjectView[T]`
