# Commands

Commands allow Go-Django applications to expose CLI subcommands through the application binary for tasks such as running checks, opening a SQL shell, creating users, or applying migrations.

## Built-in Commands and Global Flags

Every `command.Registry` includes `help`, and `django.Application.Initialize()` registers two additional built-in commands:

- `help` (`help [command]`): Lists all registered commands or prints flag usage for a specific command. Passing `help`, `--help`, or `-h` as the first argument to your application binary automatically resolves to the `help` command.
- `check`: Runs `TagSettings`, `TagSecurity`, `TagCommands`, and every app's `AppConfig.Check`, then returns `command.ErrShouldExit` (or an error if serious checks fail).
- `sql`: Opens an interactive SQL shell against `django.APPVAR_DATABASE`, or executes a single query with `-q "<query>"`. Exits the process directly via `os.Exit`.

Before dispatching the subcommand, `django.Initialize()` also parses a global `-v` boolean flag (`-v` before the command name) that sets `app.Log.SetLevel(logger.DBG)`.

---

## Core Interfaces and Types

The `command` package (`github.com/Nigel2392/go-django/src/core/command`) defines the interfaces for commands, flag binding, execution context, and registries:

```go
type Manager interface {
    Log(message string)
    Logf(format string, args ...interface{})
    Stdout() io.Writer
    Stderr() io.Writer
    Stdin() io.Reader
    Input(question string) (string, error)
    ProtectedInput(question string) (string, error)
    Registry() Registry
    Command() Command
}

type Command interface {
    Name() string
    Exec(m Manager, args []string) error
}

type CommandDescriptor interface {
    Command
    Description() string
}

type CommandAdder interface {
    AddFlags(m Manager, f *flag.FlagSet) error
}
```

- `Command`: Required interface; `args` receives any positional CLI arguments not consumed by flags.
- `CommandDescriptor`: Optional interface whose `Description()` text is displayed in `help` output.
- `CommandAdder`: Optional interface allowing the command to attach flags to a `*flag.FlagSet`.
- `Manager` (implemented by `*command.CommandManager` with `OUT`, `ERR`, `IN`, `Cmd`, `Reg`): Provides formatted output, interactive `Input(question)`, hidden terminal `ProtectedInput(question)` (via `golang.org/x/term.ReadPassword`), and access to the active `Registry`.

### Sentinel Errors

`src/core/command/errors.go` exports:

- `command.ErrUnknownCommand`: Unknown command name.
- `command.ErrTooManyArguments`: Excess arguments error.
- `command.ErrShouldExit`: Indicates the command succeeded and the process should exit rather than continuing to start the web server (also returned when `flag.ErrHelp` is returned from a command).
- `command.ErrNoCommand`: Returned when no command name was passed on the command line (so `Initialize()` proceeds normally with startup).

---

## Creating a Command with `command.Cmd[T]`

Use the generic `command.Cmd[T]` helper to implement `Command`, `CommandDescriptor`, and `CommandAdder` cleanly:

```go
package myapp

import (
    "errors"
    "flag"
    "fmt"
    "time"

    "github.com/Nigel2392/go-django/src/core/command"
)

type customCommandOpts struct {
    printTime bool
    printText string
}

var myCustomCommand = &command.Cmd[customCommandOpts]{
    ID:   "mycommand",
    Desc: "Prints the provided text with an optional timestamp",

    FlagFunc: func(m command.Manager, stored *customCommandOpts, f *flag.FlagSet) error {
        f.BoolVar(&stored.printTime, "t", false, "Print the current time")
        f.StringVar(&stored.printText, "text", "", "The text to print")
        return nil
    },

    Execute: func(m command.Manager, stored customCommandOpts, args []string) error {
        if stored.printText == "" {
            return errors.New("no text provided")
        }

        if stored.printTime {
            fmt.Println(time.Now().Format(time.RFC3339), stored.printText)
        } else {
            fmt.Println(stored.printText)
        }
        return nil
    },
}
```

### Repeatable Flags (`flags.List`)

The `flags` subpackage (`github.com/Nigel2392/go-django/src/core/command/flags`) provides `flags.List` (implementing `flag.Value`) for flags that may be specified multiple times (for example `-app blog -app auth`):

```go
var apps flags.List
f.Var(&apps, "app", "Apps to process (repeatable)")
```

`flags.List` preserves insertion order, rejects duplicates, and exposes `Len()`, `List() []string`, `Lookup(name) (int, bool)`, and `String()`.

### Command Checks (`checks.Checker`)

If a command implements `checks.Checker` (`Check(ctx context.Context) []checks.Message`), `django.Initialize()` runs those checks via `checks.TagCommands` right before executing CLI commands (unless `django.FlagSkipChecks` is set), and `check` includes them in its report.

---

## Registering a Command

Register commands on an `apps.AppConfig` via `AddCommand` (or the `Cmd` slice), or directly on `app.Commands` before `Initialize()` runs commands:

```go
myCustomApp.AddCommand(myCustomCommand)
```

You can also create standalone command registries with `command.NewRegistry(name, errorHandling)` and run them via `ExecCommand(args)` or `ExecCommandOpts(args, opts...)`.

---

## Execution Lifecycle and Configuration

During `app.Initialize()`, commands execute at **step 12**:

- **Already executed before commands run:** Logger setup, `TagSettings` & `TagSecurity` startup checks, `a.Mux` setup, global template helpers, app instantiation, model & `contenttypes` registration, App Loop 1 (`Dependencies()` & `AppConfig.Initialize(settings)`), and `TagCommands` checks.
- **Not yet executed when commands run:** App Loop 2 (`AppConfig.BuildRouting`, app `Processors()`, app `Templates()`, `AppConfig.Check`) and App Loop 3 (`AppConfig.OnReady()`).

> [!IMPORTANT]
> Because routing, app templates, and `OnReady()` run **after** CLI commands in `app.Initialize()`, custom commands should not assume `app.OnReady()` has run or app routes have been registered.

### Controlling Execution Behavior

- **Stop vs. continue (`APPVAR_CONTINUE_AFTER_COMMANDS`):** By default, once a command runs, `app.Initialize()` returns `command.ErrShouldExit` so the server does not start (note: returning `ErrShouldExit` from `Execute` also exits `Initialize()` early regardless of this setting). Set `django.APPVAR_CONTINUE_AFTER_COMMANDS: true` in settings if you want `Initialize()` to finish App Loops 2–3 and continue starting the server after a command succeeds.
- **Skip command parsing (`FlagSkipCmds`):** Pass `django.Flag(django.FlagSkipCmds)` to `django.App(...)` when embedding Go-Django in tests or tools where `os.Args` should not be parsed as management commands.

---

## Running a Command

```bash
go build -o mywebapp ./src
./mywebapp mycommand -text "Hello, World!"
# Output: Hello, World!

./mywebapp -v mycommand -text "Hello, World!" -t
# Enables logger.DBG and prints timestamp + text
```
