# Go Django Queries Documentation

Welcome to the documentation for the `go-django-queries` package.

While [Go-Django](https://github.com/Nigel2392/go-django) tries to do as little as possible with the database, sometimes helper functions make working with models easier.

This library brings Django-style ORM queries to Go-Django, allowing you to:

* Define models with relationships
* Compose queries with filters, ordering, limits
* Use select expressions and annotations

Do note that the queries package is not compatible with the default `*sql.DB` struct, but rather implements it's own `Database` interface.
Databases must always be opened using `drivers.Open(ctx, driverName, dsn)`.

---

## 📁 Documentation Structure

* [Getting Started](./docs/getting_started.md)
* [Defining Models](./docs/models/models.md)
* [Interfaces](./docs/interfaces.md)
* [Querying Objects](./docs/querying.md)
  * [QuerySet](./docs/queryset/queryset.md)
  * [Writing Queries](./docs/queryset/writing_queries.md) (WIP)
* [Relations & Joins](./docs/relations/relations.md) (WIP)
* [Expressions](./docs/expressions/expressions.md)
  * [Lookups](./docs/expressions/lookups.md)
  * [Case Expressions](./docs/expressions/cases.md)
* [Advanced: Virtual Fields](./docs/virtual_fields.md) (WIP)

---

## 🔧 Quick Example

```go
// Query forward foreign key relation
var todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    Filter("Done", false).
    OrderBy("-ID").
    All()

// Query reverse foreign key relation
var todos, err := queries.GetQuerySet(&User{}).
    Select("*", "TodoSet.*").
    Filter("TodoSet.Done", false).
    OrderBy("-ID").
    All()
```

Continue with [Getting Started](./docs/getting_started.md)…

## ✅ Supported Features

We try to support as many features as possible, but some stuff is either not supported, implemented or tested yet.

### Tested Databases

But more tests / databases will be added over time.

* SQLite through [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
* MySQL through [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
* MariaDB through [a custom driver](https://github.com/Nigel2392/go-django/queries/blob/main/src/drivers/drivers.go#L38) (with returning support, custom driver - use "mariadb" in `drivers.Open(...)`)
* [dolthub/go-mysql-server](https://github.com/dolthub/go-mysql-server) through [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)

#### Caveats

* MySQL and MariaDB requires both `multiStatements` and `interpolateParams` to be `true`. This is because
  the driver cannot [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) otherwise return multiple result id's in a single query.
  **Without this it is impossible to bulk update or bulk insert.**

* The `mariadb` driver is a custom driver that supports returning all columns upon creation of an object.
  It is a wrapper around the [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) driver.
  It can be used by passing `mariadb` as the driver name to `drivers.Open(...)`, example:

  ```go
  db, err := drivers.Open("mariadb", "user:password@tcp(localhost:3306)/dbname?multiStatements=true&interpolateParams=true")
  if err != nil {
      log.Fatal(err)
  }
  ```

### The following features are currently supported

* Selecting fields
* Selecting forward and reverse relations
* Filtering
* Lookups
* Ordering
* Limiting
* Expressions
* Annotations
* Aggregates
* Virtual fields
