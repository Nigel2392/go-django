//go:build test
// +build test

package django

import (
	goErrs "errors"
	"net/http"
	"net/http/httptest"

	"github.com/Nigel2392/goldcrest"
)

type HTTPTestClient struct {
	*http.Client
	Server *HTTPTestServer
}

type HTTPTestServer struct {
	*httptest.Server
	App *Application
}

func (s *HTTPTestServer) Client() *HTTPTestClient {
	return &HTTPTestClient{s.Server.Client(), s}
}

func (a *Application) TestServe(autoStart bool) (*HTTPTestServer, error) {
	if !a.initialized.Load() {
		if err := a.Initialize(); err != nil {
			return nil, err
		}
	}

	var httpHandler http.Handler = a.Mux
	for _, mw := range a.middlewareBuiltins() {
		httpHandler = mw(httpHandler)
	}

	var server *httptest.Server

	if autoStart {
		server = httptest.NewServer(
			httpHandler,
		)
	} else {
		server = httptest.NewUnstartedServer(
			httpHandler,
		)
	}

	for _, h := range goldcrest.Get[DjangoHook](HOOK_SERVER_STARTUP) {
		if err := h(a); err != nil {
			return nil, err
		}
	}

	a.quitter = func() (err error) {

		for _, hook := range goldcrest.Get[DjangoHook](HOOK_SERVER_SHUTDOWN) {
			e := hook(a) // func(*Application) error
			if e != nil {
				if err != nil {
					err = goErrs.Join(err, e)
				} else {
					err = e
				}
			}
		}

		server.Close()
		a.quitter = nil
		return nil
	}

	var s = &HTTPTestServer{
		Server: server,
		App:    a,
	}

	return s, nil
}
