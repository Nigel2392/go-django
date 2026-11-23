# System Checks

The `github.com/Nigel2392/go-django/src/core/checks` package provides a lightweight validation framework for checking settings, security configuration, CLI commands, models, and fields during startup or on demand.

---

## Core Types and Constants

### Severity Levels and Tags

```go
package checks

type (
	Type    = logger.LogLevel
	Tag     string
	USELESS = struct{} // Allows registering checks in package-level var declarations
)

const (
	CRITICAL Type = logger.CRIT
	ERROR    Type = logger.ERR
	WARNING  Type = logger.WRN
	INFO     Type = logger.INF
	DEBUG    Type = logger.DBG

	SERIOUS_TYPE = ERROR
)

const (
	TagSettings Tag = "settings"
	TagSecurity Tag = "security"
	TagCommands Tag = "commands"
	TagModels   Tag = "models"
)
```

- **`SERIOUS_TYPE`**: Threshold severity (`ERROR`). Any unsilenced message with `Type >= SERIOUS_TYPE` (`ERROR` or `CRITICAL`) causes `Application.Initialize()` or `./app check` to fail.

### `Checker`, `TypeStringer`, and `Message`

```go
type Checker interface {
	Check(ctx context.Context) []Message
}

type TypeStringer interface {
	TypeString() string
}

type Message struct {
	ID     string
	Type   Type
	Object any
	Hint   string
	Text   string
}

func (m Message) New(object any, hint ...string) Message
func (m Message) Is(other Message) bool
func (m Message) IsSerious() bool
func (m Message) Silenced(ctx context.Context) bool
func (m Message) String(ctx context.Context) string
```

- **`Checker`**: Optional interface that models, model fields, and CLI commands can implement to emit check messages.
- **`TypeStringer`**: Optional interface on `Message.Object` used by `Message.String(ctx)` to render a custom object label instead of a truncated `%T` type name.
- **`Message.New`**: Clones an existing `Message` (keeping `ID`, `Type`, and `Text`) with a new `Object` and optional override `Hint`. Messages cloned this way continue to match the original via `Message.Is(other)`.
- **`Message.Is`**: Compares `ID` and `Type` equality.
- **`Message.IsSerious`**: Returns `m.Type >= SERIOUS_TYPE`.
- **`Message.Silenced`**: Returns `true` if `m.ID` was silenced via `Shutup`, or if `m.Type.LT(logger.GetLevel())`.

---

## Message Constructors

The package provides unformatted and `fmt.Sprintf`-style constructors for each severity level:

```go
func Critical(ID, text string, object any, hint ...string) Message
func Error(ID, text string, object any, hint ...string) Message
func Warning(ID, text string, object any, hint ...string) Message
func Info(ID, text string, object any, hint ...string) Message
func Debug(ID, text string, object any, hint ...string) Message

func Criticalf(ID, text string, object any, hint string, args ...any) Message
func Errorf(ID, text string, object any, hint string, args ...any) Message
func Warningf(ID, text string, object any, hint string, args ...any) Message
func Infof(ID, text string, object any, hint string, args ...any) Message
func Debugf(ID, text string, object any, hint string, args ...any) Message
```

---

## Registry API

Checks are registered against a `Tag` and executed via the package-level default registry (or an isolated instance created with `NewCheckRegistry()`):

```go
func NewCheckRegistry() *checkRegistry
func Register(tag Tag, check any) USELESS
func RunCheck(ctx context.Context, tag Tag, args ...any) []Message
func Shutup(id string, yes any)
func HasTag(tag Tag) bool
```

- **`Register`**: Registers a check function under `tag`. Returns `USELESS` (`struct{}`) so registrations can sit cleanly at package level (`var _ = checks.Register(...)`).
- **`RunCheck`**: Runs every check registered under `tag`, forwarding `ctx` followed by `args...`, and returns all emitted `Message` values (`nil` if `!HasTag(tag)`).
- **`Shutup`**: Silences or unsilences a check ID. `yes` must be a `bool` or a `func(m Message) bool`; passing any other type panics.
- **`HasTag`**: Reports whether at least one check is registered for `tag`.

### Registering Custom Checks

Because `Register` wraps `check` with `versatile.NewFunc`, check functions may accept extra tag-specific parameters after `ctx context.Context`. Every registered function **must** take `context.Context` as its first parameter and return `[]checks.Message`:

```go
package myapp

import (
	"context"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/checks"
)

var _ = checks.Register(checks.TagSecurity, func(ctx context.Context, app *django.Application, settings django.Settings) []checks.Message {
	if django.ConfigGet(settings, django.APPVAR_DEBUG, false) {
		return []checks.Message{
			checks.Warning(
				"myapp.debug_enabled",
				"DEBUG is enabled",
				app,
				"Disable DEBUG in production deployments.",
			),
		}
	}
	return nil
})
```

You can silence noisy checks in startup code using `checks.Shutup`:

```go
// Silence unconditionally
checks.Shutup("model.cant_check", true)

// Or silence conditionally based on message contents
checks.Shutup("myapp.debug_enabled", func(m checks.Message) bool {
	return isLocalDev()
})
```

---

## Django Startup and CLI Integration

During `Application.Initialize()`, Go-Django runs checks in three stages unless `django.FlagSkipChecks` is set:

1. **Startup checks**: Runs `checks.RunCheck(ctx, checks.TagSettings, a, a.Settings)` and `checks.RunCheck(ctx, checks.TagSecurity, a, a.Settings)`.
2. **Command checks**: Runs `checks.RunCheck(ctx, checks.TagCommands, a, a.Settings, a.Commands.Commands())`.
3. **Application checks**: Calls `app.Check(ctx, a.Settings)` on every registered `django.AppConfig`.

Any unsilenced serious message (`ERROR` or `CRITICAL`) causes `Application.Initialize()` to log the count of failures and return an error (`"Startup checks failed"`, `"Command checks failed"`, or `"Application checks failed"`).

You can also run all checks explicitly from the CLI without starting the HTTP server using the built-in `check` command (`src/commands.go`):

```bash
go run . check
```

To skip checks programmatically (for example, in tests), pass `django.FlagSkipChecks`:

```go
app := django.App(
	django.Flag(django.FlagSkipChecks),
)
```

---

## Built-In Checks

Go-Django registers several built-in checks out of the box:

- **`settings.matches.<KEY>`** (`TagSettings`, `src/django_checks.go`): Validates the Go types of core `APPVAR_*` settings (such as `DEBUG`, `ALLOWED_HOSTS`, `DATABASE`, `STATIC_URL`, `TLS_CONFIG`, `TRANSLATIONS_DEFAULT_LOCALE`, etc.). In addition, `ALLOWED_HOSTS` must be present and non-empty (`len(setting) > 0`), or startup fails with a `CRITICAL` check message.
- **`secrets.missing.key`** (`TagSettings`, `src/core/secrets/vars.go`): Emits a `CRITICAL` message when `APPVAR_SECRET_KEY` is missing from settings, including a freshly generated 32-character random key in `Hint`.
- **`TagCommands`** (`src/django_checks.go`): Checks any registered `command.Command` that implements `checks.Checker`.
- **`AppConfig.Check`** (`src/apps/apps.go`): Iterates every model in `AppConfig.ModelObjects`. If the model implements `checks.Checker`, it delegates directly to `model.Check(ctx)`. Otherwise it emits `model.cant_check` (`WARNING`), checks field DB types (`field.invalid_db_type`), runs `field.Check(ctx)` on fields implementing `checks.Checker` (such as `FieldDef` length/value bounds), checks for multiple primary keys (`model.multiple_primary_keys`), and warns if a non-through model has no primary key (`model.no_primary_key`).

---

## Caveats

- **Exact function signature requirement**: Functions passed to `checks.Register` must accept `context.Context` as their first parameter and return `[]checks.Message`. `TagSettings` and `TagSecurity` pass `(ctx, app, settings)`, while `TagCommands` passes `(ctx, app, settings, commands)`.
- **Logger level suppresses lower-severity messages**: `Message.Silenced(ctx)` returns `m.Type.LT(logger.GetLevel())` when not explicitly silenced via `Shutup`. If your global logger level (see [Logging](./logging.md)) is set above `WARNING` or `ERROR`, check messages below that level are treated as silenced—even during `SERIOUS_TYPE` counting.
