# Logging

Logging is done with the `logger` package.

The logger package provides a relatively extensive interface, and provides a default implementation of said interface.

## Loglevels

A default logger will be set with loglevel INFO.

Allowed log levels are:

 * `logger.DBG` - Loglevel DEBUG
 * `logger.INF` - Loglevel INFO
 * `logger.WRN` - Loglevel WARNING
 * `logger.ERR` - Loglevel ERROR

A special case exists for `logger.Fatal` and `logger.Fatalf`.

These will log the message and then call `os.Exit()` with the provided exit code.

## The `Log` interface

The `Log` interface is the main interface for the logger.

For a logger to be used with the global `logger.*` package functions, it must implement this interface.

```go
type Log interface { // size=16 (0x10)

    // Writer returns a new io.Writer that writes to the log output for the given log level.
    Writer(level LogLevel) io.Writer

    // PWriter returns a new io.Writer that writes to the log output for the given log level with a prefix.
    PWriter(label string, level LogLevel) io.Writer

    // NameSpace returns a new Log with the given label as the prefix.
    NameSpace(label string) Log

    // SetOutput sets the output for the given log level.
    SetOutput(level LogLevel, w io.Writer)

    // SetLevel sets the current log level.
    //
    // Log messages with a log level lower than the current log level will not be written.
    SetLevel(level LogLevel)

    // Log a debug message.
    Debug(args ...interface{})

    // Log an info message.
    Info(args ...interface{})

    // Log a warning message.
    Warn(args ...interface{})

    // Log an error message.
    Error(args ...interface{})

    // Log a message and exit the program with the given error code.
    Fatal(errorcode int, args ...interface{})

    // Log a format- and args-based debug message.
    Debugf(format string, args ...interface{})

    // Log a format- and args-based info message.
    Infof(format string, args ...interface{})

    // Log a format- and args-based warning message.
    Warnf(format string, args ...interface{})

    // Log a format- and args-based error message.
    Errorf(format string, args ...interface{})

    // Log a format- and args-based message and exit the program with the given error code.
    Fatalf(errorcode int, format string, args ...interface{})

    // Log a message at the given log level.
    Log(level LogLevel, args ...interface{})

    // Log a format string at the given log level.
    Logf(level LogLevel, format string, args ...interface{})

    // WriteString writes a string to the log output.
    WriteString(s string) (n int, err error)
}
```

## Outputs

The logger will output to `io.Discard` by default.

This can be changed by calling `logger.SetOutput(loglevel, io.Writer)`.

It is possible to set different outputs for different loglevels.

Example:

```go
logger.SetOutput(logger.OutputAll, os.Stdout)

var file, err = os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
if err != nil {
	panic(err)
}

logger.SetOutput(logger.INF, file)
```

In this example, all log messages will be written to `os.Stdout`, but INFO messages will instead be written to `log.txt`.

Using the logger is relatively simple.

### Basic usage of the Log interface

#### Logging a message

```go
logger.Log(logger.INF, "This is an info message")
```

#### Logging a message with formatting

```go
logger.Logf(logger.INF, "This is an info message with a number: %d", 42)
```

#### Logging a debug message

```go
logger.Debug("This is a debug message")
```

#### Logging a warning message

```go

logger.Warn("This is a warning message")
```

#### Logging an error message

```go
logger.Error("This is an error message")
```

#### Logging a fatal message

A fatal message will log the message and then call `os.Exit()` with the provided exit code.

```go
logger.Fatal(1, "This is a fatal message")
```

### Advanced usage of the Log interface

#### `Writer`

The logger provides a `Writer` function that returns an `io.Writer` that writes to the log output for the given log level.

This can be used to redirect the output of a function to the logger.

*Be mindful that your log messages can get scrambled, or split apart if say, an `io.Copy` is used and the copy buffer is too small.*

```go
var writer = logger.Writer(logger.INF)
writer.Write([]byte("This is an info message\n"))
```

#### `PWriter`

The logger provides some more advanced functions for logging.

For say, an `exec.Command`, you can use the `PWriter` function to prefix the log messages.

```go
var cmd = exec.Command("ls", "-l")
cmd.Stdout = logger.PWriter("ls", logger.INF)
cmd.Stderr = logger.PWriter("ls", logger.ERR)

if err := cmd.Run(); err != nil {
    logger.Error("Command failed:", err)
}
```

This will prefix all log messages with the logger's formatting and the provided prefix.

*Be mindful that your log messages can get scrambled, or split apart if say, an `io.Copy` is used and the copy buffer is too small.*

#### `NameSpace`

The logger provides a `NameSpace` function that returns a new `Log` interface with the given label as the prefix.

This can be used to create a new logger with a specific prefix.

```go
var myCustomNamespace = logger.NameSpace("myCustomNamespace")
myCustomNamespace.Info("This is an info message")
```

As a fun side- effect, namespaces can be nested.

```go
var myCustomNamespace = logger.NameSpace("myCustomNamespace")
var myCustomNamespace2 = myCustomNamespace.NameSpace("myCustomNamespace2")
myCustomNamespace2.Info("This is an info message")
```