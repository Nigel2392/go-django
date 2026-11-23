# Go-Django Command-Line Utility

The `go-django` CLI tool scaffolds new projects, creates individual apps, and generates Dockerfiles for building and running Go-Django applications.

## Installation

```bash
go install github.com/Nigel2392/go-django/cmd/go-django@latest
```

## Global Flags and Help

Run `go-django help` or `go-django help <command>` to inspect available subcommands and flags.

Global flags apply before the subcommand:

- `--verbose`, `-v`: Enable info-level logging (`logger.INF`, which is also the default log level).
- `--debug`, `-vv`: Enable debug-level logging (`logger.DBG`).

```bash
go-django --debug startproject -m github.com/example/myproject myproject
```

---

## Commands

### `dockerfile`

Generate a `Dockerfile` for building and running your project.

- `--dir`, `-d` (`string`): Directory where the `Dockerfile` will be written (defaults to the current working directory).
- `--port`, `-p` (`uint`, default `8080`): Port exposed via `EXPOSE` in the generated `Dockerfile`.
- `--executable`, `-e` (`string`, default `"main"`): Name of the compiled Go binary built from `./src`.
- `--scratch`, `-s` (`bool`): Generate a multi-stage `Dockerfile` using `golang:<version>-alpine` with `-ldflags="-extldflags=-static"` in the builder stage and `FROM scratch` (with `/etc/ssl/certs/ca-certificates.crt`) in the final stage.
- `--vendored`, `-v` (`bool`): Copy `vendor/` into the container instead of running `go mod download`.
- `--force`, `-f` (`bool`): Overwrite an existing `Dockerfile` instead of exiting with an error.

#### Go Version Resolution

When generating the `Dockerfile`, `dockerfile` reads `runtime/debug.ReadBuildInfo()` and strips the leading `"go"` prefix from `GoVersion` to pick the `golang:<version>-alpine` builder image. If build info is unavailable or empty, it falls back to `"1.24.5"`.

#### Example

```bash
go-django dockerfile -d . -p 8080 -e myapp -s
# Equivalent long form:
go-django dockerfile --dir . --port 8080 --executable myapp --scratch
```

#### Caveat with `--force`

When `--force` (`-f`) is specified, the `Dockerfile` is opened with `os.O_CREATE|os.O_WRONLY` without `os.O_TRUNC`. If the newly generated file is shorter than an existing `Dockerfile`, trailing bytes from the previous file can remain on disk; delete the old `Dockerfile` first or verify the resulting file when switching flags such as removing `--scratch`.

---

### `startproject`

Initialize a new Go-Django project module and starter application.

Usage:

```bash
go-django startproject -m <module-path> [-d <dir>] <projectname>
```

- `--module`, `-m` (`string`, **required**): Go module path passed to `go mod init`.
- `--dir`, `-d` (`string`, optional): Directory to create the project in (defaults to the current working directory).

#### Directory Resolution and Steps

If `--dir` does not start with `./` and is not `.` (for example an absolute path or omitted `--dir` resolving via `os.Getwd()`), `startproject` appends `<projectname>` as a subdirectory (`<dir>/<projectname>`). When `--dir` is `.` or starts with `./`, files are written directly into that directory without appending `<projectname>`.

After copying the template tree (`src/main.go`, `src/<projectname>/`, templates, and static assets), `startproject`:

1. Runs `go mod init <module>`.
2. Queries the GitHub API (`https://api.github.com/repos/Nigel2392/go-django/tags`) to look up the latest tag (**requires network access**).
3. Runs `go get -u github.com/Nigel2392/go-django@<latest-tag>`.
4. Runs `go mod tidy`.

#### Example

```bash
go-django startproject -m github.com/example/myproject -d ./my-project myproject
# Equivalent long form:
go-django startproject --module github.com/example/myproject --dir ./my-project myproject
```

---

### `startapp`

Scaffold a new Go-Django app package (`<appname>/<appname>.go`, `views.go`, static assets, and starter template).

Usage:

```bash
go-django startapp [-d <dir>] <appname>
```

- `--dir`, `-d` (`string`, optional): Target directory where `<appname>/` will be created. If omitted, `startapp` checks whether `./src` exists in the current working directory and uses `./src` if present, otherwise falling back to the current working directory.

#### Example

```bash
go-django startapp -d ./my-project/src myapp
# Equivalent long form:
go-django startapp --dir ./my-project/src myapp
```

---

## Note on `cmd/migrator`

The `cmd/migrator` binary inside the Go-Django repository is an internal repository development tool driven by `migrations.yml` for maintaining contrib app migrations, rather than a general-purpose user CLI.

For generating and applying migrations in your own project, use your application's `migrator` management commands as documented in [Migrations](../queries/docs/migrations.md) and [Commands](./commands.md).
