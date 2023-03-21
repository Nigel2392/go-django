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
	fmt.Println("Creating app: ", appName)
	return nil
}