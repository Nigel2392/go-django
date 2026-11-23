# Messages

The `messages` app provides flash messaging akin to Django's `contrib.messages`, allowing you to store messages in the user's session or HTTP cookies and display them on a subsequent request.

## Installation

Include `messages.NewAppConfig` in `django.Apps(...)` to install the messages middleware and template context processor:

```go
package main

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/session"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/mux"
)

func main() {
	settings := map[string]any{
		django.APPVAR_ALLOWED_HOSTS: []string{"*"},
		django.APPVAR_DEBUG:         true,
	}

	// Optional: if omitted, messages uses NewSessionBackend when the
	// "session" app is installed, or NewCookieBackend otherwise.
	messages.ConfigureBackend(
		messages.NewSessionBackend,
	)

	app := django.App(
		django.Configure(settings),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			apps.NewAppConfigForHandler(
				"myapp",
				mux.GET, "/myapp",
				mux.NewHandler(myView),
			),
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

During initialization:

- If `messages.ConfigureBackend` was not called, `messages` selects `NewSessionBackend` when the [`session`](./sessions.md) app is installed, or falls back to `NewCookieBackend` otherwise.
- `MessagesMiddleware` is registered globally wrapped with `django.NonStaticMiddleware`, so static routes are skipped.
- `RequestMessageContext` is registered as a template context processor and runs whenever templates are rendered using a context implementing [`ctx.ContextWithRequest`](../context.md) (such as `ctx.RequestContext(r)`).

## Message Interface and `BaseMessage`

In `contrib/messages`, `messages.Message` is an **interface**, and `*messages.BaseMessage` is the concrete struct stored by the built-in backends (pre-registered with `encoding/gob` in `init()`):

```go
type Message interface {
	Tag() MessageTag
	Message() string
	ExtraTags() []MessageTag
	Prepare() error
}

type BaseMessage struct {
	Level       MessageTag   `json:"tag"`
	Text        string       `json:"message"`
	ExtraLevels []MessageTag `json:"extra"`
}
```

`BaseMessage` also implements `String() string`, returning `m.Text`.

## Levels, Tags, and Configuration

Message levels use the `MessageTag` string type and numeric severity levels:

```go
type MessageTag string

const (
	DEBUG   MessageTag = "debug"
	INFO    MessageTag = "info"
	SUCCESS MessageTag = "success"
	WARNING MessageTag = "warning"
	ERROR   MessageTag = "error"

	LEVEL_DEBUG   uint = 10
	LEVEL_INFO    uint = 20
	LEVEL_SUCCESS uint = 30
	LEVEL_WARNING uint = 40
	LEVEL_ERROR   uint = 50
)

type MessageTags struct {
	Debug   MessageTag
	Info    MessageTag
	Success MessageTag
	Warning MessageTag
	Error   MessageTag
}
```

The package also exports `DefaultTags`, `TagLevels`, `LevelTags`, and the following helpers and sentinel errors:

- `messages.Tags() *MessageTags`: Returns the configured `MessageTags`.
- `(*MessageAppConfig).ConfigureTags(tags MessageTags)`: Customizes the tags exposed via `messages.Tags()` and the `DEFAULT_MESSAGE_LEVELS` template context variable.
- `messages.GetLevel(r *http.Request) (MessageTag, error)`: Returns the minimum `MessageTag` level currently active for the request's backend.
- `messages.SetLevel(r *http.Request, level MessageTag) error`: Sets the minimum `MessageTag` level stored by the backend; messages below this level are discarded by `Store`.
- Sentinel errors: `ErrRequestRequired`, `ErrBackendNotConfigured`, `ErrMessagesNotInstalled`.

## Adding and Reading Messages in Views

Use the helper functions in `contrib/messages` to queue messages or inspect the backend for a request:

```go
func AddMessage(r *http.Request, tag MessageTag, message string, extraTags ...MessageTag) error
func Debug(r *http.Request, message string, extraTags ...MessageTag) error
func Info(r *http.Request, message string, extraTags ...MessageTag) error
func Success(r *http.Request, message string, extraTags ...MessageTag) error
func Warning(r *http.Request, message string, extraTags ...MessageTag) error
func Error(r *http.Request, message string, extraTags ...MessageTag) error

func Debugf(r *http.Request, message string, argsAndTags ...any) error
func Infof(r *http.Request, message string, argsAndTags ...any) error
func Successf(r *http.Request, message string, argsAndTags ...any) error
func Warningf(r *http.Request, message string, argsAndTags ...any) error
func Errorf(r *http.Request, message string, argsAndTags ...any) error

func Messages(r *http.Request) (messages []Message, AllRetrieved bool)
func Backend(r *http.Request) MessageBackend
func BackendFromContext(r *http.Request) MessageBackend
```

Do note a quirk in `Debugf`, `Infof`, `Successf`, `Warningf`, and `Errorf`: `tagsFromArgs` consumes any leading `MessageTag` or `[]MessageTag` values from `argsAndTags` and passes everything starting at the first non-tag argument to `fmt.Sprintf`. Extra tags must therefore come **before** format arguments (e.g., `messages.Infof(r, "Saved %s", messages.MessageTag("bold"), name)`), and passing only tags with no format arguments leaves `lastIdx` at `0`, forwarding those tags into `fmt.Sprintf`.

Example view rendering a template with queued messages:

```go
package main

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

func myView(w http.ResponseWriter, r *http.Request) {
	_ = messages.Debug(r, "This is a debug message")
	_ = messages.Info(r, "This is an info message")
	_ = messages.Warning(r, "This is a warning message")
	_ = messages.Error(r, "This is an error message")
	_ = messages.Success(r, "This is a success message")

	context := ctx.RequestContext(r)
	if err := tpl.FRender(w, context, "myapp", "myapp/template.tmpl"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
```

See the [rendering](../rendering.md) and [context](../context.md) documentation for template registration details.

## Showing Messages in Templates

When rendering a template with a [`ctx.ContextWithRequest`](../context.md) context, `RequestMessageContext` populates two keys and immediately calls `backend.Clear()` so messages are consumed once:

- `"messages"`: `[]messages.Message` retrieved from the backend.
- `"DEFAULT_MESSAGE_LEVELS"`: `*messages.MessageTags` containing the configured tag constants.

```html
<section class="messages">
    {{ $levels := (.Get "DEFAULT_MESSAGE_LEVELS") }}
    {{ range $msg := (.Get "messages") }}
        {{ if eq $msg.Tag $levels.Success }}
            <div class="message bg-success">
                <div class="message-text">{{ $msg.Message }}</div>
            </div>
        {{ else if eq $msg.Tag $levels.Error }}
            <div class="message bg-danger">
                <div class="message-text">{{ $msg.Message }}</div>
            </div>
        {{ else if eq $msg.Tag $levels.Warning }}
            <div class="message bg-warning">
                <div class="message-text">{{ $msg.Message }}</div>
            </div>
        {{ else if eq $msg.Tag $levels.Info }}
            <div class="message bg-info">
                <div class="message-text">{{ $msg.Message }}</div>
            </div>
        {{ else if eq $msg.Tag $levels.Debug }}
            <div class="message bg-debug">
                <div class="message-text">{{ $msg.Message }}</div>
            </div>
        {{ end }}
    {{ end }}
</section>
```

## Backends

All storage backends implement the `MessageBackend` interface:

```go
type MessageBackend interface {
	Get() (messages []Message, AllRetrieved bool)
	Store(message Message) error
	Level() MessageTag
	SetLevel(level MessageTag) error
	Finalize(w http.ResponseWriter, r *http.Request) error
	Clear() error
}
```

### The Cookie Backend

`NewCookieBackend` encodes messages with `encoding/gob` and base64 into an HTTP cookie (subject to the ~4KB browser cookie size limit). By default, it uses cookie name `"messages.cookieBackendKey"` with `MaxAge` set to 7 days (`60 * 60 * 24 * 7`), plus a `<Name>.level` cookie for `SetLevel`.

> [!IMPORTANT]
> `messages.APPVAR_COOKIE_KEY` MUST be set to an `http.Cookie{...}` **value**, NOT a `*http.Cookie` pointer (`&http.Cookie{...}`). `NewCookieBackend` calls `django.ConfigGet` with an `http.Cookie` value default; passing a pointer causes a runtime panic due to type mismatch.

```go
settings := map[string]any{
	django.APPVAR_ALLOWED_HOSTS: []string{"*"},
	django.APPVAR_DEBUG:         true,
	messages.APPVAR_COOKIE_KEY: http.Cookie{
		Name:     "my_cookie_name",
		MaxAge:   60 * 60 * 24 * 30,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	},
}
```

### The Session Backend

`NewSessionBackend` stores messages inside the user's session via [`sessions.Retrieve`](./sessions.md). It requires the `session` app to be installed (otherwise `NewSessionBackend` returns `ErrBackendNotConfigured`), supports larger payloads, and avoids exposing raw message payloads in client cookies.

## Caveats

- **`Finalize` and flushed headers in `CookieBackend`**: `MessagesMiddleware` calls `backend.Finalize(w, r)` *after* `next.ServeHTTP(w, r)` returns. `django.Application` wraps non-static routes in `middleware.BufferMiddleware` so writes are buffered until middleware unwinds, but if you bypass `BufferMiddleware` or flush the response body early (for example via `http.Flusher`) on a non-redirect response, headers are sent before `Finalize` runs and `Set-Cookie` updates are lost.
- **`CookieBackend` `Secure` and clear-cookie attributes**: `CookieBackend.Finalize` copies `Path`, `Domain`, `Expires`, `MaxAge`, `HttpOnly`, and `SameSite` from `APPVAR_COOKIE_KEY`, **but does not copy `Secure`**. Additionally, when `Clear()` triggers cookie deletion, `Finalize` writes `&http.Cookie{Name: d.BaseCookie.Name, Value: "", MaxAge: -1}` **without** `Path`, `Domain`, `HttpOnly`, `SameSite`, or `Secure`—which can prevent browsers from deleting cookies scoped to a custom `Path` or `Domain`.
- **`ConfigureTags` scope**: Calling `ConfigureTags` changes `app.Tags` (and `DEFAULT_MESSAGE_LEVELS`), while `TagLevels` and `LevelTags` remain keyed by the built-in `DEBUG`, `INFO`, `SUCCESS`, `WARNING`, and `ERROR` constants.
