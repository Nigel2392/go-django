# Getting Started with go-django-queries

This guide will help you get started with the `go-django-queries` package.

---

## 🏗 Installation

First we need to install the package.

We use Go's semantic versioning, so we can use the `@latest` tag to get the latest version.

If you want to use a specific version, you can use the `@vX.Y.Z` tag, where `X.Y.Z` is the version number.

```bash
go get github.com/Nigel2392/go-django/queries@latest
```

---

## ✅ Imports

We have divided most of the queries package into different smaller packages for separation of concerns.

The following imports will be used throughout the examples:

```go
import (
    "github.com/Nigel2392/go-django/queries/src"
    "github.com/Nigel2392/go-django/queries/src/expr"
    "github.com/Nigel2392/go-django/queries/src/fields"
    "github.com/Nigel2392/go-django/queries/src/models"
    "github.com/Nigel2392/go-django/queries/src/drivers/errors"
    "github.com/Nigel2392/go-django/src/core/attrs"
)
```

---

## Database Setup

Before we can start using the queries package, we need to setup the database.

For this example, we will use SQLite, but you can use any database that is supported by Go-Django.

To setup the database, we need to create a `sql.DB` object, and register it in Go-Django's settings.

```go
func main() {
    var db, err = drivers.Open("sqlite3", "./db.sqlite3")
    if err != nil {
        panic(err)
    }

    var app = django.App(
        django.Configure(map[string]interface{}{
            django.APPVAR_DATABASE: db,
            // ...
        }),
        // ...
    )
    // ...
}
```

---

The following step (if you haven't already) [is to define your models](./models/models.md)…
