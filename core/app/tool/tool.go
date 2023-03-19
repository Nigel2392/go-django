package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/google/uuid"
)

func main() {
	var flags = flag.NewFlags("Go-Django-Tools")
	flags.Info = `Go-Django-Tools is a tool to help you create a new Go-Django project.`
	flags.Register("env", "", "", func(value flag.Value) error {
		if !value.IsZero() {
			var env_str_generated_secret = fmt.Sprintf(env_template, uuid.New().String())
			return createFileStr(value.String(), env_str_generated_secret)
		}
		return nil
	})

	flags.Register("start", "", "", startProject)
	flags.Register("app", "", "", startApp)

	if !flags.Run() {
		fmt.Println("No flags passed")
	}
}

func startProject(v flag.Value) error {
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
	if err := createDir("assets/templates/base"); err != nil {
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
	var env_str_generated_secret = fmt.Sprintf(env_template, uuid.New().String())
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

func startApp(v flag.Value) error {
	var appName = v.String()
	fmt.Println("Creating app: ", appName)
	return nil
}
