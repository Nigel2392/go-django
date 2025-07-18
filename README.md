# go-django (v2)

![1719351174099](.github/image/README/1719351174099.png)

**Django** rewritten to **Golang**.

This is a rewrite of the Django framework in Golang.

The goal is to provide a similar experience to Django, but with the performance of Golang.

At the core this is meant to be a web framework, but it also includes sub-packages to create a CMS-like experience.

Any database logic can be implemented with the [`queries`](./queries/README.md) subpackage, providing an experience similar to
the Django ORM.

Latest version: `v1.7.0`

## Installation

The package is easily installed with `go get`.

```bash
go get github.com/Nigel2392/go-django@v1.7.0
```

The [CLI](./docs/cli.md) can optionally be installed with `go install`.

This will provide some useful utilities to help you get started, like creating a new project, app or Dockerfile.

```bash
go install github.com/Nigel2392/go-django/cmd/go-django@v1.7.0
```

Or to install the SQLC plugin to auto- generate SQL queries and [go-django definitions](./docs/sqlc.md) from your SQL database.

```bash
go install github.com/Nigel2392/go-django/cmd/go-django-definitions@v1.7.0
```

## Docs

- [Using the CLI](./docs/cli.md)
- [Configuring your server](./docs/configuring.md)
- [Creating an app](./docs/apps.md)
- [Setting up Routing](./docs/routing.md)
- [Working with context](./docs/context.md)
- [Working with the Filesystem](./docs/filesystem.md)
- [Rendering Your Templates](./docs/rendering.md)
- [Easily rendering Views](./docs/views.md) (WIP)
- [Creating Forms](./docs/forms/readme.md)
  - [Working with Fields](./docs/forms/fields.md)
  - [Working with Widgets](./docs/forms/widgets.md)
  - [Passing and creating Media](./docs/forms/media.md)
- [Information about models](./docs/models.md)
  - [Defining your models](./docs/attrs/attrs.md)
  - [Auto-generating GO-django models](./docs/sqlc.md)
  - [Usage of Contenttypes](./docs/contenttypes.md)
- [Paginating your data](./docs/pagination.md)
- [Cache Management](./docs/cache.md)
- [Caching your Views](./docs/caching_views.md)
- [Sending Emails](./docs/mail.md)
- [Setting up Logging](./docs/logging.md)
- [Setting up and calling Hooks](./docs/hooks.md)
- [Serving your Staticfiles](./docs/staticfiles.md)
- [Working with permissions](./docs/permissions.md)
- [Creating Management Commands](./docs/commands.md)

### Contrib apps

- [sessions](./docs/apps/sessions.md)
- [admin](./docs/apps/admin) (WIP)
- [auditlogs](./docs/apps/auditlogs.md) (WIP)
- [auth](./docs/apps/auth) (WIP)
- [oauth2](./docs/apps/oauth2.md)
- [messages](./docs/apps/messages.md)
- [pages](./docs/apps/pages/readme.md)
- [editorjs](./docs/apps/editor.md) (WIP)

## Tested Databases

GO-Django is tested to work on the following databases:

But more tests / databases will be added over time.

- SQLite through [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
- MySQL through [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
- Local MYSQL with [dolthub/go-mysql-server](https://github.com/dolthub/go-mysql-server) through [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) (used internally for testing)
- MariaDB through [a custom driver](https://github.com/Nigel2392/go-django/queries/blob/main/src/drivers/drivers.go#L38) (with returning support, custom driver - use "mariadb" in `drivers.Open(...)`)
- Postgres through [jackc/pgx](https://github.com/jackc/pgx)

### Examples

- [Todo App](./docs/examples/todos.md)
- [Blog App](./docs/examples/blog.md)

### How to work with models in the database

- [go-django-queries](./queries/README.md) - A library to help you create SQL queries specialized (and only useful) for go-django models.

## Help Needed

- [ ] Block application:
  - [ ] Javascript for structblock
  - [ ] Javascript for listblock
  - [ ] (maybe) Javascript for fieldblock
