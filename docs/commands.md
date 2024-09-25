# Commands

## Introduction

Commands are a way to add custom commands to your app.

These commands can be run from the command line, and can be used to do things like database migrations, or other tasks that need to be run outside of the normal request/response cycle.

Some apps, like the `auth` app, will come with their own commands.

Go-Django only provides a `help` command by default.

A command is any object which implements the `command.Command` interface.

```go
type Command interface {
    // How the command should be called from the command line
    Name() string

    // Add optional flags to the flagset
    AddFlags(m command.Manager, f *flag.FlagSet) error

    // Execute the command
    // Any arguments not consumed by the flags will be passed here
    Exec(m command.Manager, args []string) error
}
```

Though if your command implements a `Description() string` method, it will be used in the help output.

## Creating a command

To create a command, you need to create a new struct that implements the `command.Command` interface.

A default interface implementation is already in place in the `command` package.

The following example shows how to create a simple command that prints a message to the console.

```go
package myapp

import (
    "github.com/Nigel2392/go-django/src/core/command"
    "flag"
    "fmt"
    "time"
    "errors"
)

type customCommandObj struct {
    printTime bool
    printText string
}

var myCustomCommand = &command.Cmd[customCommandObj]{
    ID:   "mycommand",
    Desc: "Prints the provided text with an optional timestamp",

    FlagFunc: func(m command.Manager, stored *customCommandObj, f *flag.FlagSet) error {
        f.BoolVar(&stored.printTime, "t", false, "Print the current time")
        f.StringVar(&stored.printText, "text", "", "The text to print")
        return nil
    },

    Execute: func(m command.Manager, stored customCommandObj, args []string) error {
        if stored.printText == "" {
            return errors.New("No text provided")
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

## Registering a command

Commands are required to be registered to an `apps.AppConfig` object for the command to be available when calling `Initialize()` on your django app.

Let's say the appconfig is stored in a variable called `myCustomApp`.

Ideally, you would register the command in the `NewCustomAppConfig` function as is directed in [the apps documentation](./apps.md#methods).

```go
myCustomApp.AddCommand(myCustomCommand)
```

## Running a command

After building your application, you can run the command from the command line.

```bash
go build -o mywebapp .
./mywebapp mycommand -text "Hello, World!"
# Output: Hello, World!

./mywebapp mycommand -text "Hello, World!" -t
# Output: <current-time> Hello, World!
```
