package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/google/uuid"
)

var VERSION = ""

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
	return initGoMod(v.String(), VERSION)
}

type Tag struct {
	Name    string `json:"name"`
	Integer int    `json:"-"`
}

type Tags []Tag

func (t Tags) Len() int {
	return len(t)
}

func DecodeTags(data io.Reader) Tags {
	var tags = Tags{}
	decoder := json.NewDecoder(data)
	err := decoder.Decode(&tags)
	if err != nil {
		panic(err)
	}
	return tags
}
func (t Tags) initInts() {
	for i := 0; i < len(t); i++ {
		var tag = strings.TrimPrefix(t[i].Name, "v")
		var tag_parts = strings.Split(tag, ".")
		var newTag = strings.Join(tag_parts, "")
		for len(newTag) < 4 {
			newTag += "0"
		}
		var newTagInt, err = strconv.Atoi(newTag)
		if err != nil {
			panic(errors.New("Could not create tags: " + err.Error()))
		}
		t[i].Integer = newTagInt
	}
}

func (t Tags) Descending() {
	t.initInts()
	for i := 0; i < len(t); i++ {
		for j := i + 1; j < len(t); j++ {
			if t[i].Integer < t[j].Integer {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
}

func (t Tags) Ascending() {
	t.initInts()
	for i := 0; i < len(t); i++ {
		for j := i + 1; j < len(t); j++ {
			if t[i].Integer > t[j].Integer {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
}

func (t Tags) Latest() Tag {
	t.initInts()
	t.Descending()
	return t[0]
}

var GITHUB_TAG_URL = "https://api.github.com/repos/Nigel2392/go-django/tags"
var GITHUB_REPO_URL = "github.com/Nigel2392/go-django"

// Initialize go.mod to get the latest version of the project.
// This only works for github repositories with tags in the following format:
func initGoMod(projectName string, extra string) error {
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
