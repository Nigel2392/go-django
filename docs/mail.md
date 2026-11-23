# Mail

The `github.com/Nigel2392/go-django/src/core/mail` package provides a configurable email sending layer built on top of [`github.com/jordan-wright/email`](https://github.com/jordan-wright/email). It includes a named backend registry, a console backend for local development, an SMTP connection-pool backend, and `PLAIN` / `XOAUTH2` / `LOGIN` authentication helpers.

---

## Configuration

The `Config` struct configures an SMTP server connection and also implements `net/smtp.Auth` (using `"LOGIN"` authentication by default):

```go
type Config struct {
	Host        string        // Mail server host.
	Port        int           // Mail server port.
	Username    string        // Mail server username.
	Password    string        // Mail server password.
	UseTLS      bool          // USE_TLS
	UseSSL      bool          // USE_SSL
	MailFrom    string        // Mail server from address.
	Timeout     time.Duration // Timeout duration for sending email.
	TLSConfig   *tls.Config   // TLS Config
	DefaultAuth smtp.Auth     // Default SMTP Auth
}

func (m *Config) Start(server *smtp.ServerInfo) (string, []byte, error)
func (m *Config) Next(fromServer []byte, more bool) ([]byte, error)
```

When passed to `NewPooledEmailBackend`:

- `Host`, `Username`, `Password`, and `MailFrom` are asserted to be non-empty via `assert.Truthy(...)` (an empty value panics).
- `Port` defaults to `25` when `0`.
- `Timeout` defaults to `10 * time.Second` when `0`.
- `UseTLS` attaches `TLSConfig` (or `&tls.Config{ServerName: cnf.Host}` if `TLSConfig == nil`). Note that `UseSSL` is currently unused in `NewPooledEmailBackend`.
- `DefaultAuth` overrides the default `LOGIN` auth implementation on `Config` when non-nil.

### Authentication Helpers

`src/core/mail` provides two additional `smtp.Auth` helpers:

```go
func PlainAuth(identity, username, password, host string) smtp.Auth
func XOAuth2Auth(username, token string) smtp.Auth
```

- **`PlainAuth`**: Implements `PLAIN` SMTP authentication and verifies `server.Name == host`.
- **`XOAuth2Auth`**: Implements `XOAUTH2` Bearer token authentication for providers such as Gmail or Microsoft 365.

---

## Backends and Interfaces

```go
const DefaultBackend = "default"

var (
	ErrConfigNil       = errors.New("config is nil")
	ErrBackendNotFound = errors.New("backend not found")
)

type EmailBackend interface {
	Send(e *email.Email) error
}

type OpenableEmailBackend interface {
	Open() error
	IsOpen() bool
	Close() error
}

func NewConsoleBackend(f io.Writer) EmailBackend
func NewPooledEmailBackend(poolCount int, cnf *Config) (EmailBackend, error)
```

- **`EmailBackend`**: Core interface implemented by every backend.
- **`OpenableEmailBackend`**: Optional lifecycle interface for backends that manage connections or resources. `Get` and `Send` automatically call `Open()` when `!IsOpen()`.
- **`NewConsoleBackend`**: Returns a backend that writes serialized email messages followed by a newline to `f`. During package `init()`, `DefaultBackend` (`"default"`) is registered to `NewConsoleBackend(os.Stdout)`.
- **`NewPooledEmailBackend`**: Creates an SMTP connection pool of size `poolCount` using `email.NewPool(...)` and returns it as an `EmailBackend`.

### Caveat: `NewPooledEmailBackend` and `OpenableEmailBackend`

Although the unexported `pooledManager` returned by `NewPooledEmailBackend` defines `Open() error` and `Close() error`, it currently does **not** implement `IsOpen() bool`. Because of this:

- `NewPooledEmailBackend` returns `(EmailBackend, error)`, so `backend.Close()` does not exist on the returned interface.
- `pooledManager` does **not** satisfy `OpenableEmailBackend`, so `mail.Close(backend)` is currently a no-op on pooled backends (and `Get` / `Send` do not invoke `Open()`, though `NewPooledEmailBackend` already opens the pool on creation).

---

## Registry and Package Functions

```go
func Register(name string, backend EmailBackend)
func Unregister(name string)
func Get(name string) EmailBackend
func Default() EmailBackend
func SetDefault(backend EmailBackend)
func Send(e *email.Email, backendOrName ...interface{}) error
func Close[T string | interface{}](backendOrName T) error
```

- **`Register` / `Unregister`**: Registers or removes a backend by name. Registering a named backend when `Default()` is `nil` also sets it as the default.
- **`Get` / `Default` / `SetDefault`**: Retrieves a backend by name (opening it if it implements `OpenableEmailBackend` and is closed), returns the `"default"` backend, or replaces `"default"`.
- **`Send`**: Sends `e` using the first `string` backend name or `EmailBackend` value passed in `backendOrName`, falling back to `Default()`. Returns `ErrBackendNotFound` if the resolved backend is `nil`.
- **`Close`**: Resolves a backend by name or `EmailBackend` instance and calls `Close()` if it implements `OpenableEmailBackend`.

---

## Examples

### Sending an Email via the Default Console Backend

```go
package main

import (
	"github.com/Nigel2392/go-django/src/core/mail"
	"github.com/jordan-wright/email"
)

func main() {
	msg := &email.Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Hello",
		Text:    []byte("This is a test email."),
	}

	if err := mail.Send(msg); err != nil {
		panic(err)
	}
}
```

### Registering and Sending via a Pooled SMTP Backend

```go
package main

import (
	"time"

	"github.com/Nigel2392/go-django/src/core/mail"
	"github.com/jordan-wright/email"
)

func main() {
	cnf := &mail.Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "your-username",
		Password: "your-password",
		UseTLS:   true,
		MailFrom: "sender@example.com",
		Timeout:  15 * time.Second,
	}

	backend, err := mail.NewPooledEmailBackend(5, cnf)
	if err != nil {
		panic(err)
	}

	mail.SetDefault(backend)

	msg := &email.Email{
		From:    cnf.MailFrom,
		To:      []string{"recipient@example.com"},
		Subject: "Hello",
		Text:    []byte("This is a test email."),
	}

	if err := mail.Send(msg); err != nil {
		panic(err)
	}
}
```

### Using a Named Backend for Tests or Capturing Output

```go
var buf bytes.Buffer
mail.Register("capture", mail.NewConsoleBackend(&buf))

if err := mail.Send(msg, "capture"); err != nil {
	panic(err)
}
```
