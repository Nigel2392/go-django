package cmd

import (
	"flag"
	"fmt"
	"net/http"
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

var GITHUB_TAG_URL = "https://api.github.com/repos/Nigel2392/go-django/tags"
var GITHUB_REPO_URL = "https://github.com/Nigel2392/go-django"

// Initialize go.mod to get the latest version of the project.
// This only works for github repositories with tags in the following format:
//
//	vDIGITS.DIGITS.DIGITS
func initGoMod(projectName string) error {
	var name = httputils.NameFromPath(projectName)
	var cmd = exec.Command("go", "mod", "init", "go-django/"+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	var req, _ = http.NewRequest("GET", GITHUB_TAG_URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	// Decode the tags from the github api
	var tagList = DecodeTags(resp.Body)
	// Get the latest version of jsext
	var latestTag = tagList.Latest()

	var extra = ""
	if len(latestTag.Name) > 1 {
		var prefix = latestTag.Name[0]
		if prefix == 'v' || prefix == 'V' {
			extra = "/" + latestTag.Name[0:1]
		}
	}

	if err := runCmd("go", "get", GITHUB_REPO_URL+extra+"@"+latestTag.Name); err != nil {
		return err
	}
	if err := runCmd("go", "mod", "tidy"); err != nil {
		return err
	}
	return nil
}

// Run a system command.
func runCmd(name string, args ...string) error {
	var cmd = exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	var err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func startApp(appName string) {

}
