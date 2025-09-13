//go:build test
// +build test

package django

import (
	"net/http"
	"net/http/httptest"

	"github.com/Nigel2392/goldcrest"
	"github.com/justinas/nosurf"
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

	var disableNosurf = ConfigGet(
		a.Settings, APPVAR_DISABLE_NOSURF, false,
	)

	var httpHandler http.Handler = a.Mux
	if !disableNosurf {
		var handler = nosurf.New(a.Mux)
		var hooks = goldcrest.Get[NosurfSetupHook](HOOK_SETUP_NOSURF)
		for _, hook := range hooks {
			hook(a, handler)
		}
		httpHandler = handler
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

	a.quitter = func() error {
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
