//go:build test
// +build test

package messages_test

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/mux"
)

var _ messages.MessageBackend = (*DummyBackend)(nil)

type DummyBackend struct {
	under *messages.SessionBackend
}

func (d *DummyBackend) Get() (messages []messages.Message, AllRetrieved bool) {
	return d.under.Get()
}
func (d *DummyBackend) Store(message messages.Message) error {
	return d.under.Store(message)
}
func (d *DummyBackend) Clear() error {
	return d.under.Clear()
}
func (d *DummyBackend) Level() messages.MessageTag {
	return d.under.Level()
}
func (d *DummyBackend) SetLevel(level messages.MessageTag) error {
	return d.under.SetLevel(level)
}

var (
	server  *httptest.Server
	app     *django.Application
	client  *http.Client
	backend messages.MessageBackend
)

func newTestAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("testapp")
	app.Routing = func(m django.Mux) {
		m.Handle(mux.ANY, "/", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			backend = messages.BackendFromContext(req)
			w.Write([]byte("Hello World"))
		}))

		m.Handle(mux.ANY, "/add", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			var err = messages.AddMessage(
				req, messages.INFO, "Test message",
			)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error adding message"))
				return
			}

			w.Write([]byte("Message added"))
		}))

		m.Handle(mux.ANY, "/test", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			var messages, _ = messages.Messages(req)
			var lenStr = strconv.Itoa(len(messages))
			w.Write([]byte(lenStr))
		}))
	}
	return app
}

func init() {
	var settings = make(map[string]any)
	settings[django.APPVAR_ALLOWED_HOSTS] = []string{"*"}
	settings[django.APPVAR_DEBUG] = true

	app = django.App(
		django.Configure(settings),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			newTestAppConfig,
		),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipCmds,
		),
	)

	var err error
	server, err = app.TestServe(false)
	if err != nil {
		panic(err)
	}

	server.Start()

	client = server.Client()

	var cookieJar *cookiejar.Jar
	cookieJar, err = cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client.Jar = cookieJar
}

func TestMessagesMiddleware(t *testing.T) {

	var response, err = client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	if response.Body != nil {
		defer response.Body.Close()
	}

	if response.StatusCode != 200 {
		t.Fatalf("Expected status code 200, got %d", response.StatusCode)
	}

	if backend == nil {
		t.Fatal("Expected backend to be set, got nil")
	}
}

func TestAddMessages(t *testing.T) {

	var response, err = client.Get(server.URL + "/add")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	var body = new(bytes.Buffer)
	if response.Body != nil {
		defer response.Body.Close()
		body.ReadFrom(response.Body)
	}

	if response.StatusCode != 200 {
		t.Fatalf("Expected status code 200, got %d", response.StatusCode)
	}

	if body.String() != "Message added" {
		t.Fatalf("Expected body to be 'Message added', got %s", body.String())
	}

	response, err = client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	body = new(bytes.Buffer)
	if response.Body != nil {
		defer response.Body.Close()
		body.ReadFrom(response.Body)
	}

	count, err := strconv.Atoi(body.String())
	if err != nil {
		t.Fatalf("Expected count to be an integer, got %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected count to be 1, got %d", count)
	}
}
