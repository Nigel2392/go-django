package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/google/uuid"
)

func init() {
	var writeEnv *string = flag.String("env", "", "Create a new environment file.")
	var _startProject *string = flag.String("start", "", "Create a new project in the current directory.")
	var _startApp *string = flag.String("app", "", "Create a new app in the current directory.")
	flag.Parse()
	if *writeEnv != "" {
		var env_str_generated_secret = fmt.Sprintf(env_template, uuid.New().String())
		createFile(*writeEnv, []byte(env_str_generated_secret))
		os.Exit(0)
	}
	if *_startProject != "" {
		startProject(*_startProject)
		os.Exit(0)
	}
	if *_startApp != "" {
		startApp(*_startApp)
		os.Exit(0)
	}
}

func startProject(projectName string) {
	createDir(projectName)
	os.Chdir(projectName)
	createDir("assets/static")
	createDir("assets/media")
	createDir("assets/templates/base")
	createDir("src/apps")
	createFile("src/main.go", []byte(mainTemplate))
	createFile("src/config.go", []byte(appConfigTemplate))
	var env_str_generated_secret = fmt.Sprintf(env_template, uuid.New().String())
	createFile(".env", []byte(env_str_generated_secret))

	initGoMod(projectName)
}

func initGoMod(projectName string) {
	var name = httputils.NameFromPath(projectName)
	var cmd = exec.Command("go", "mod", "init", "go-django/"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func startApp(appName string) {

}
