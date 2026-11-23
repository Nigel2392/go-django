# Server Errors Example

The `server_errors` example (`examples/server_errors`) demonstrates how to raise HTTP errors with `except.Fail` and register custom per-status-code HTTP error handlers in Go-Django.

## Table of Contents

- [Project Structure](#project-structure)
- [Running the Example](#running-the-example)
- [Error Handler Configuration (`main.go`)](#error-handler-configuration-maingo)
- [Raising and Handling Errors (`views.go`)](#raising-and-handling-errors-viewsgo)

---

## Project Structure

```text
examples/server_errors/
├── go.mod
├── main.go
└── views.go
```

Both `main.go` and `views.go` belong to `package main`.

---

## Running the Example

Because `package main` is split across `main.go` and `views.go`, run the entire package rather than `main.go` alone (`go run main.go` fails because the handlers in `views.go` will not be compiled):

From `examples/server_errors`:

```bash
go run .
```

Or from the repository root:

```bash
go run ./examples/server_errors
```

Once running on `http://127.0.0.1:8080`, test the routes in a browser or with `curl`:

- `http://127.0.0.1:8080/` — renders `"Welcome to the Index page!"`.
- `http://127.0.0.1:8080/raise/400` — raises `400` and invokes `Handle400`.
- `http://127.0.0.1:8080/raise/403` — raises `403` and invokes `Handle403`.
- `http://127.0.0.1:8080/raise/404` — raises `404` and invokes `Handle404`.
- `http://127.0.0.1:8080/raise/500` — raises `500` and invokes `Handle500`.
- `http://127.0.0.1:8080/raise/<code>` (for example `/raise/418`) — raises an arbitrary status code; if no custom `HandleErrorCode<code>` handler is registered, Go-Django falls back to `HOOK_SERVER_ERROR` hooks or `http.Error(w, message, code)`.

---

## Error Handler Configuration (`main.go`)

Error handlers are registered in `django.Configure(...)` using `django.APPVAR_ErrorCode(code)` (which formats `fmt.Sprintf("HandleErrorCode%d", code)`—or the equivalent literal string such as `"HandleErrorCode400"`):

```go
package main

import (
	"net/http"
	"os"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

func main() {
	app := django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         true,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",

			"HandleErrorCode400":                                    Handle400,
			django.APPVAR_ErrorCode(http.StatusForbidden):           Handle403,
			django.APPVAR_ErrorCode(http.StatusNotFound):            Handle404,
			django.APPVAR_ErrorCode(http.StatusInternalServerError): Handle500,
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.DBG,
			OutputTime:  true,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),
		django.Apps(&apps.AppConfig{
			AppName: "server_errors",
			Routing: func(m mux.Multiplexer) {
				m.Any("/", mux.NewHandler(Index))
				m.Any("/raise/<<code>>", mux.NewHandler(RaiseErrorCode))
			},
		}),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

Every handler registered under `APPVAR_ErrorCode(code)` must have the exact signature `func(http.ResponseWriter, *http.Request, except.ServerError)`.

See [Errors](../errors.md), [Configuring](../configuring.md), and [Routing](../routing.md).

---

## Raising and Handling Errors (`views.go`)

`RaiseErrorCode` extracts `code` from `/raise/<<code>>` (defaulting `0` to `500`) and calls `except.Fail(code, ...)`. Go-Django's error handling dispatches the resulting `except.ServerError` to the matching `Handle*` function:

```go
package main

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux"
)

func Index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the Index page!\n"))
}

func RaiseErrorCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars.GetInt("code")
	if code == 0 {
		code = http.StatusInternalServerError
	}

	except.Fail(code, "This is a test error: %d", code)
}

func Handle500(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("An internal server error occurred, please try again later.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}

func Handle404(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("The page you are looking for does not exist.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}

func Handle403(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("You do not have permission to access this resource.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}

func Handle400(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("The request was invalid or cannot be served.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}
```
