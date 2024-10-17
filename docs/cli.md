# GO-Django command line utility

The `go-django` command line utility is a tool that helps you easily initialize new projects, apps or a dockerfile.

## Installation

```bash
go install github.com/Nigel2392/go-django/cmd/go-django@latest
```

## Usage

```bash
go-django help
```

This will show you all the available commands and flags.

## Commands

- `dockerfile`: Create a Dockerfile for your project.
- `startproject`: Create a new project.
- `startapp`: Create a new app.

Each command has its own help page, which you can access by running `go-django help <command>`.

## Arguments

### `dockerfile`

Initialize a new Dockerfile for your project.

- dir:
    *Alias*: d
    The directory to write the Dockerfile to
- port:
    *Alias*: p
    The port the application listens on
- executable:
    *Alias*: e
    The name of the executable
- scratch:
    *Alias*: s
    Use a scratch image
- vendored:
    *Alias*: v
    Use a vendored build
- force:
    *Alias*: f
    Force overwrite of existing Dockerfile

#### Example

```bash
go-django dockerfile -d . -p 8080 -e myapp -s
# This translates to:
go-django dockerfile --dir . --port 8080 --executable myapp --scratch
```

### `startproject`

Initialize a new project with the given go module name.

- module:
    *Alias*: m
    The name of the go module
- dir:
    *Alias*: d
    The directory to create the project in

#### Example

```bash
go-django startproject -m github.com/Nigel2392/myproject -d ./my-project
# This translates to:
go-django startproject --module github.com/Nigel2392/myproject --dir ./my-project
```

### `startapp`

Start a new app in the given project directory.

If no directory is provided, we will look for a `src` directory in the current working directory.

- dir:
    *Alias*: d
    The directory to create the app in

#### Example

```bash
go-django startapp -d ./my-project
# This translates to:
go-django startapp --dir ./my-project
```
