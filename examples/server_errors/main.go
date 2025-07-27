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

	/*
		This example demonstrates how to handle server errors
		using the go-django framework.

		It includes custom error handlers for various HTTP status codes,
		such as 400 Bad Request, 403 Forbidden, 404 Not Found, and 500 Internal Server Error.

		A helper function is defined to easily create the required setting string to handle the error code.
		Essentially, it is the same as doing `fmt.Sprintf("HandleErrorCode%d", <error_code>)`.

		To use the example application, run it and visit the following URLs:
		- http://127.0.0.1:8080/ - Displays a welcome message.
		- http://127.0.0.1:8080/raise/400 - Raises a 400 Bad Request error.
		- http://127.0.0.1:8080/raise/403 - Raises a 403 Forbidden error.
		- http://127.0.0.1:8080/raise/404 - Raises a 404 Not Found error.
		- http://127.0.0.1:8080/raise/500 - Raises a 500 Internal Server Error.

		You can also raise an error with a custom code by visiting:
		- http://127.0.0.1:8080/raise/<code> - Replace `<code>` with any HTTP status code you want to test.

		For example, to raise a 418 I'm a teapot error, visit:
		- http://127.0.0.1:8080/raise/418
	*/

	var app = django.App(
		// Configure the go-django application
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

		// Initialize the logger for the application
		django.AppLogger(&logger.Logger{
			Level:       logger.DBG,
			OutputTime:  true,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),

		// Add apps to the application's app registry
		// It is OK to either pass the NewAppConfig function
		// or the app config struct directly.
		django.Apps(&apps.AppConfig{
			AppName: "server_errors",
			Routing: func(m mux.Multiplexer) {
				m.Any("/", mux.NewHandler(Index))
				m.Any("/raise/<<code>>", mux.NewHandler(RaiseErrorCode))
			},
		}),
	)

	// Initialize the application
	var err = app.Initialize()
	if err != nil {
		panic(err)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
