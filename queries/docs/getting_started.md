# Getting Started with the Queries Package

This guide walks through installing the `queries` package, configuring a database connection, and registering the migrator so you can define models and run queries in Go-Django.

---

## Installation

Install the `queries` module using `go get`:

```bash
go get github.com/Nigel2392/go-django/queries@latest
```

---

## Imports and Side-Effect Registration

The `queries` package is split into focused subpackages for drivers, errors, expressions, fields, models, and migrations:

```go
import (
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)
```

Two imports carry important automatic `init()` registrations:

- Importing `github.com/Nigel2392/go-django/queries/src/drivers` automatically registers the four built-in database drivers: `"sqlite3"`, `"mysql"`, `"mariadb"`, and `"postgres"`.
- Importing `github.com/Nigel2392/go-django/queries/src` automatically blank-imports the migrator SQL schema editors (`migrator/sql/mysql`, `migrator/sql/postgres`, and `migrator/sql/sqlite`), sets `migrator.CompilerForModel`, registers query compilers for all four drivers, and hooks `models.MODEL_SAVE_HOOK` / `models.MODEL_DELETE_HOOK`.

---

## Database and Migrator Setup

Go-Django operates on `drivers.Database` rather than `*sql.DB`. Open a connection with `drivers.Open(ctx, driverName, dsn, opts...)`, pass it to `django.APPVAR_DATABASE`, and include `migrator.NewAppConfig` in `django.Apps(...)` if you want schema migrations and CLI commands (`makemigrations`, `migrate`).

```go
package main

import (
	"context"
	"log"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
)

var _ = queries.Compiler // ensures queries/src init() runs if no symbols are referenced yet

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "./db.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE: db,
			// Optional: override the default "./migrations" directory
			migrator.APPVAR_MIGRATION_DIR: "./migrations",
		}),
		django.Apps(
			migrator.NewAppConfig,
			// Your application configs here...
		),
	)

	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}
```

Do note that if `queries/src` is only needed in an entrypoint for its hooks/compilers without calling a symbol directly, use a blank import `_ "github.com/Nigel2392/go-django/queries/src"` so its `init()` functions run before `app.Initialize()` executes.

---

## Next Steps

- Configure database drivers, custom types, and query logging in [Drivers](./drivers.md).
- Inspect typed ORM and SQL error sentinels in [Errors](./errors.md).
- Define your first struct models in [Defining Models](./models/models.md).
- Set up schema migrations in [Migrations](./migrations.md).
- Run queries with [Basic Querying](./querying.md) and [The QuerySet API](./queryset/queryset.md), or see [Configuring Go-Django](../../docs/configuring.md) for application-wide settings.
