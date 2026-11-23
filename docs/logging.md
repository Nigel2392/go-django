# Logging

Logging in Go-Django is handled by the `logger` package (`github.com/Nigel2392/go-django/src/core/logger`), which provides a `Log` interface, a leveled `*logger.Logger` implementation, `io.Writer` adapters, namespaced sub-loggers, and ANSI color helpers.

## Log Levels

Available log levels (`logger.LogLevel`):

- `logger.DBG`: Debug messages (`"DEBUG"`).
- `logger.INF`: Informational messages (`"INFO"`, default log level).
- `logger.WRN`: Warning messages (`"WARN"`).
- `logger.ERR`: Error messages (`"ERROR"`).
- `logger.CRIT`: Critical messages (`100`, `"CRITICAL"`), used by `Fatal` / `Fatalf` before calling `os.Exit(errorcode)`.
- `logger.OutputAll`: Special level (`-1`) accepted by `SetOutput` to direct all standard log outputs (`DBG`, `INF`, `WRN`, `ERR`) to the same writer.

Parse a level name case-insensitively with `logger.ParseLogLevel(s string) (LogLevel, error)`.

---

## The `Log` Interface

Any logger passed to `logger.Setup(...)` or `django.AppLogger(...)` must implement `logger.Log`:

```go
type Log interface {
    // Getters
    Writer(level LogLevel) io.Writer
    PWriter(label string, level LogLevel) io.Writer
    NameSpace(label string) Log
    GetLevel() LogLevel

    // Setters
    SetOutput(level LogLevel, w io.Writer)
    SetLevel(level LogLevel)

    // Logging
    Debug(args ...interface{})
    Info(args ...interface{})
    Warn(args ...interface{})
    Error(args ...interface{})
    Fatal(errorcode int, args ...interface{})
    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})
    Fatalf(errorcode int, format string, args ...interface{})
    Log(level LogLevel, args ...interface{})
    Logf(level LogLevel, format string, args ...interface{})
    WriteString(s string) (n int, err error)
}
```

---

## Configuring the Logger (`Setup`, `Logger`, and `django.AppLogger`)

### The `Logger` Struct

```go
type Logger struct {
    Level       LogLevel
    Prefix      string
    Suffix      string
    OutputTime  bool
    OutputDebug io.Writer
    OutputInfo  io.Writer
    OutputWarn  io.Writer
    OutputError io.Writer
    WrapPrefix  func(context.Context, LogLevel, string) string
    Context     context.Context
}
```

- When an output writer is `nil`, `Logger` chains outputs upward from lowest to highest severity: `OutputDebug` defaults to `io.Discard`, `OutputInfo` defaults to `OutputDebug`, `OutputWarn` defaults to `OutputInfo`, and `OutputError` defaults to `OutputWarn`.
- `logger.Setup(logInstance)` binds the global package-level functions (`logger.Info`, `logger.Errorf`, `logger.Logf`, `logger.SetOutput`, `logger.SetLevel`, `logger.GetLevel`, `logger.Writer`, `logger.PWriter`, `logger.NameSpace`, etc.) to `logInstance`.

### Configuring with `django.AppLogger`

Pass `django.AppLogger(...)` when creating the application to configure the logger before `Initialize()` runs:

```go
app := django.App(
    django.AppLogger(&logger.Logger{
        Level:       logger.DBG,
        OutputTime:  true,
        Prefix:      "myapp",
        OutputDebug: os.Stdout,
        WrapPrefix:  logger.ColoredLogWrapper,
    }),
    // ...
)
```

If `django.AppLogger` is not used, `app.Initialize()` automatically creates a `*logger.Logger` at `logger.GetLevel()`, with `Prefix: "django"`, `OutputTime: true`, `WrapPrefix: logger.ColoredLogWrapper`, and `OutputAll` pointing to `os.Stdout`, then calls `logger.Setup(a.Log)`.

### Setting Per-Level Outputs

```go
logger.SetOutput(logger.OutputAll, os.Stdout)

file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
if err != nil {
    panic(err)
}
logger.SetOutput(logger.INF, file)
```

---

## Using the Global Logger

Package `logger` exports package-level functions (`Debug`, `Debugf`, `Info`, `Infof`, `Warn`, `Warnf`, `Error`, `Errorf`, `Fatal`, `Fatalf`, `Logf`, `Writer`, `PWriter`, `NameSpace`, `SetOutput`, `SetLevel`, `GetLevel`).

> [!NOTE]
> There is **no** package-level `logger.Log(...)` function because `Log` is the interface name. Use `logger.Logf(level, format, args...)`, level-specific helpers such as `logger.Info(...)`, or call `.Log(level, args...)` on a `logger.Log` instance (such as `app.Log` or a namespace).

```go
logger.Info("Application started")
logger.Infof("Listening on port %d", 8080)
logger.Debug("Debug details")
logger.Warn("Warning message")
logger.Error("Something went wrong")
logger.Logf(logger.INF, "Meaning of life: %d", 42)
logger.Fatal(1, "Unrecoverable error") // logs at CRIT and calls os.Exit(1)
```

---

## `LogWriter`, `PWriter`, and `NameSpace`

### `Writer` and `PWriter`

`Writer(level)` and `PWriter(label, level)` return a `*logger.LogWriter` (`io.Writer`) that forwards writes to the logger if `level >= Logger.Level`:

```go
writer := logger.Writer(logger.INF)
writer.Write([]byte("This is an info message\n"))
```

Use `PWriter` to capture `exec.Command` output with a namespace prefix:

```go
cmd := exec.Command("ls", "-l")
cmd.Stdout = logger.PWriter("ls", logger.INF)
cmd.Stderr = logger.PWriter("ls", logger.ERR)

if err := cmd.Run(); err != nil {
    logger.Error("Command failed:", err)
}
```

*Note: Writing arbitrary byte chunks (such as via `io.Copy` with small buffers) can split a single line across multiple prefixed writes.*

### `NameSpace`

`NameSpace(label)` returns a copy of the logger with `label` appended to its prefix (nested namespaces join with `" / "`):

```go
ns := logger.NameSpace("auth")
sub := ns.NameSpace("oauth")
sub.Info("Token refreshed")
```

---

## Color Helpers

`src/core/logger/wrappers.go` provides ANSI terminal coloring utilities:

- Constants: `CMD_Green`, `CMD_Cyan`, `CMD_Yellow`, `CMD_Red`, `CMD_Bold`, `CMD_Reset`
- `ColoredLogWrapper(ctx context.Context, l LogLevel, s string) string`: Default `WrapPrefix` coloring `DBG` green, `INF` cyan, `WRN` yellow, and `ERR`/`CRIT` bold red.
- `Colorize(ctx context.Context, color, s any) string` and `FColorize(ctx context.Context, w io.Writer, color, s any) (int, error)`
- `WithSupportsColors(ctx context.Context, supports bool) context.Context` and `SupportsColors(ctx context.Context) bool` (defaults to `logger.SUPPORTS_COLORS_DEFAULT`, which is `true`).

---

## Caveats

1. **No-op before `logger.Setup`:** Before `logger.Setup(...)` is called (either directly, via `django.AppLogger(...)`, or by `app.Initialize()`), the default package-level `Debug`/`Info`/`Warn`/`Error`/`Fatal`/`Logf` functions do nothing (`printArgs` has an empty body), `SetOutput` is a no-op, and `logger.NameSpace` returns `nil`.
2. **`(*Logger).Debugf` level check:** `Debugf` checks `DBG.LT(l.Level)` (`DBG <= l.Level`, always true for `DBG`..`CRIT`) instead of `l.Level.LT(DBG)`. If `OutputDebug` is set to a non-discard writer (for example via `SetOutput(OutputAll, os.Stdout)`), `Debugf` will emit output even when `Level` is `INF` or higher.
3. **`(*Logger).Log` and `(*Logger).Logf` bypass level filtering:** Calling `Log` or `Logf` writes straight to `l.Output(level)` without checking `l.Level`.
