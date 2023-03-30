package tool

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/google/uuid"
)

func StartProject(v flag.Value) error {
	var projectName = v.String()
	if err := createDir(projectName); err != nil {
		return err
	}
	os.Chdir(projectName)
	if err := createDir("assets/static"); err != nil {
		return err
	}
	if err := createDir("assets/media"); err != nil {
		return err
	}
	if err := createFile("assets/templates/app/index.tmpl", []byte(defaultHTMLTemplate)); err != nil {
		return err
	}
	if err := createFile("assets/templates/base/base.tmpl", []byte(defaultBaseHTMLTemplate)); err != nil {
		return err
	}
	if err := createFile("assets/templates/base/messages.tmpl", []byte(defaultMessagesTemplate)); err != nil {
		return err
	}
	if err := createDir("src/apps"); err != nil {
		return err
	}
	if err := createFile("src/main.go", []byte(mainTemplate)); err != nil {
		return err
	}
	if err := createFile("src/config.go", []byte(appConfigTemplate)); err != nil {
		return err
	}
	var env_str_generated_secret = fmt.Sprintf(Env_template, uuid.New().String())
	if err := createFile(".env", []byte(env_str_generated_secret)); err != nil {
		return err
	}
	return initGoMod(v)
}

func initGoMod(v flag.Value) error {
	var projectName = v.String()
	var name = httputils.NameFromPath(projectName)
	var cmd = exec.Command("go", "mod", "init", "go-django/"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func StartApp(v flag.Value) error {
	var appName = v.String()
	if err := createDir(appName); err != nil {
		panic(err)
	}
	if err := os.Chdir(appName); err != nil {
		panic(err)
	}
	createFile("urls.go", []byte(getURLSTemplate(httputils.NameFromPath(appName))))
	createFile("views.go", []byte(getViewsTemplate(httputils.NameFromPath(appName))))
	createFile("models.go", []byte(getModelsTemplate(httputils.NameFromPath(appName))))
	return nil
}

func getURLSTemplate(appName string) string {
	return fmt.Sprintf(urlsTemplate, appName, appName, appName, appName, appName, appName)
}

func getViewsTemplate(appName string) string {
	return fmt.Sprintf(viewsTemplate, appName, appName)
}

func getModelsTemplate(appName string) string {
	return fmt.Sprintf(modelsTemplate, appName, appName)
}

var urlsTemplate = `package %s

import (
	"github.com/Nigel2392/router/v3"
)

var %s_route = router.Group("/%s", "%s")

func URLs() router.Registrar {
	%s_route.Get("", index, "index")
	return %s_route
}`

var viewsTemplate = `package %s

import (
	"github.com/Nigel2392/router/v3/request"
)

func index(r *request.Request) {
	r.WriteString("Hello from %s")
}`

var modelsTemplate = `package %s

import (
	"github.com/Nigel2392/go-django/core/models"
)

// Declare your models here.
type %s_model struct {
	models.Model
}`
