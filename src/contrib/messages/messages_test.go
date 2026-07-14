//go:build test
// +build test

package messages_test

import (
	"bytes"
	"fmt"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"strconv"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

var (
	server  *django.HTTPTestServer
	app     *django.Application
	client  *django.HTTPTestClient
	backend messages.MessageBackend
)

func newTestAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("testapp")
	app.Routing = func(m mux.Multiplexer) {
		m.Handle(mux.ANY, "/", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			logger.Infof("Request cookies: %v", req.Cookies())
			backend = messages.BackendFromContext(req)
			w.Write([]byte("Hello World"))
		}))

		m.Handle(mux.ANY, "/setlevel", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			logger.Infof("Request cookies: %v", req.Cookies())
			backend = messages.BackendFromContext(req)
			if err := backend.SetLevel(messages.INFO); err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error adding message"))
				return
			}
			w.Write([]byte("level_set"))
		}))

		m.Handle(mux.ANY, "/getlevel", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			logger.Infof("Request cookies: %v", req.Cookies())
			backend = messages.BackendFromContext(req)
			w.Write([]byte(backend.Level()))
		}))

		m.Handle(mux.ANY, "/add", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			logger.Infof("Request cookies: %v", req.Cookies())
			var err = messages.AddMessage(
				req, messages.INFO, "Test message",
			)
		returnError:
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error adding message"))
				return
			}

			if err = messages.Debug(req, "Test message"); err != nil {
				goto returnError
			}
			if err = messages.Info(req, "Test message"); err != nil {
				goto returnError
			}
			if err = messages.Success(req, "Test message"); err != nil {
				goto returnError
			}
			if err = messages.Warning(req, "Test message"); err != nil {
				goto returnError
			}
			if err = messages.Error(req, "Test message"); err != nil {
				goto returnError
			}

			w.Write([]byte("Message added"))
		}))

		m.Handle(mux.ANY, "/test", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			logger.Infof("Request cookies: %v", req.Cookies())
			var messages, _ = messages.Messages(req)
			var lenStr = strconv.Itoa(len(messages))
			w.Write([]byte(lenStr))
		}))

		m.Handle(mux.ANY, "/clear", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			logger.Infof("Request cookies: %v", req.Cookies())
			backend = messages.BackendFromContext(req)
			if err := backend.Clear(); err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error clearing messages"))
				return
			}

			w.Write([]byte("success"))
		}))

	}
	return app
}

func init() {
	var settings = make(map[string]any)
	settings[django.APPVAR_ALLOWED_HOSTS] = []string{"*"}
	settings[django.APPVAR_DEBUG] = true
	settings[django.APPVAR_RECOVERER] = false

	messages.ConfigureBackend(
		messages.NewCookieBackend,
	)

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
			django.FlagSkipChecks,
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

var backendFuncs = map[string]func(r *http.Request) (messages.MessageBackend, error){
	"Cookie":  messages.NewCookieBackend,
	"Session": messages.NewSessionBackend,
}

func TestMessagesMiddleware(t *testing.T) {
	for name, backendFunc := range backendFuncs {
		t.Run(fmt.Sprintf("Test_%s_Backend", name), func(t *testing.T) {
			backend = nil

			messages.ConfigureBackend(backendFunc)

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
		})
	}
}

func TestAddMessages(t *testing.T) {

	for name, backendFunc := range backendFuncs {
		t.Run(fmt.Sprintf("Test_%s_Backend", name), func(t *testing.T) {
			backend = nil

			messages.ConfigureBackend(backendFunc)

			var response, err = client.Get(server.URL + "/getlevel")
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			var body = new(bytes.Buffer)
			if response.Body != nil {
				defer response.Body.Close()
				body.ReadFrom(response.Body)
			}

			if body.String() != "debug" {
				t.Fatalf("Expected 'debug', got %q", body.String())
			}

			response, err = client.Get(server.URL + "/setlevel")
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			body = new(bytes.Buffer)
			if response.Body != nil {
				defer response.Body.Close()
				body.ReadFrom(response.Body)
			}

			if body.String() != "level_set" {
				t.Fatalf("Expected 'level_set', got %q", body.String())
			}

			response, err = client.Get(server.URL + "/getlevel")
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			body = new(bytes.Buffer)
			if response.Body != nil {
				defer response.Body.Close()
				body.ReadFrom(response.Body)
			}

			if body.String() != "info" {
				t.Fatalf("Expected 'info', got %q", body.String())
			}

			response, err = client.Get(server.URL + "/add")
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			body = new(bytes.Buffer)
			if response.Body != nil {
				body.ReadFrom(response.Body)
				response.Body.Close()
			}

			if response.StatusCode != 200 {
				t.Fatalf("Expected status code 200, got %d", response.StatusCode)
			}

			if body.String() != "Message added" {
				t.Fatalf("Expected body to be 'Message added', got %s", body.String())
			}

			if name == "Cookie" {
				var cookie = response.Cookies()
				var cm = make(map[string]*http.Cookie)
				for _, c := range cookie {
					cm[c.Name] = c
				}

				if c, ok := cm["messages.cookieBackendKey"]; !ok {
					t.Fatal("Expected cookie 'messages.cookieBackendKey' to be set, got keys:", slices.Collect(maps.Keys(cm)))
				} else {
					if c.Value == "" {
						t.Fatal("Expected cookie 'messages.cookieBackendKey' to have a value")
					}
				}
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

			if count != 5 {
				t.Fatalf("Expected count to be 5, got %d", count)
			}

			response, err = client.Get(server.URL + "/clear")
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			body = new(bytes.Buffer)
			if response.Body != nil {
				defer response.Body.Close()
				body.ReadFrom(response.Body)
			}

			if body.String() != "success" {
				t.Fatalf("Expected 'success'")
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

			count, err = strconv.Atoi(body.String())
			if err != nil {
				t.Fatalf("Expected count to be an integer, got %v", err)
			}

			if count != 0 {
				t.Fatalf("Expected count to be 0, got %d", count)
			}

		})
	}

}
