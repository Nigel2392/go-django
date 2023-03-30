package main

import (
	"fmt"

	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/logger"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

var App = app.New(appConfig)

func main() {
	// There are some default commandline flags registered.
	// You can add your own flags by using go-django's flag package.
	// See the help menu for more information. (go run ./src -h)

	auth.USER_MODEL_LOGIN_FIELD = "Email"
	App.Router.Get("/", index, "index")

	var _, err = auth.CreateAdminUser(
		"developer@local.local", // Email
		"Developer",             // Username
		"root",                  // First name
		"toor",                  // Last name
		"Developer123!",         // Password
	)
	if err != nil {
		fmt.Println(logger.Colorize(logger.Red, fmt.Sprintf("Error creating superuser: %s", err.Error())))
	}

	err = App.Run()
	if err != nil {
		panic(err)
	}
}

func index(req *request.Request) {
	var err = response.Render(req, "app/index.tmpl")
	req.Data.Set("title", "Home")
	if err != nil {
		req.Error(500, err.Error())
		return
	}
}