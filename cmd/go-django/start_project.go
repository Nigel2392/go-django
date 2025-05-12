package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/urfave/cli/v2"
)

type githubTagInfo struct {
	Name string `json:"name"`
}

func fetchLatestVersion() (string, error) {
	// https://api.github.com/repos/%s/tags
	var url = fmt.Sprintf(
		"https://api.github.com/repos/%s/tags",
		GO_DJANGO_PACKAGE,
	)

	var resp, err = http.Get(url)
	if err != nil {
		return "", err
	}

	var l = make([]githubTagInfo, 0)
	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return "", err
	}

	if len(l) == 0 {
		return "", fmt.Errorf("no tags found for %s", GO_DJANGO_PACKAGE)
	}

	var latest = l[0]
	return latest.Name, nil
}

var startProjectCommand = &cli.Command{
	Name:        "startproject",
	Usage:       "Start a new GO-Django project",
	Description: `Create a new project with the given name and module path.`,
	UsageText:   "go-django startproject [projectname]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "module",
			Aliases:  []string{"m"},
			Usage:    "The GO module path of the project",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "dir",
			Aliases:  []string{"d"},
			Usage:    "The directory to create the project in",
			Required: false,
		},
	},
	Action: func(c *cli.Context) error {
		var projectFS, err = fs.Sub(assetFiles, "assets/templates/project")
		if err != nil {
			return err
		}
		var projectPath string = c.String("dir")
		if projectPath != "" {
			os.MkdirAll(projectPath, 0755)
		} else {
			projectPath, err = os.Getwd()
		}
		if err != nil {
			return err
		}

		var module = "github.com/user/project"
		if c.IsSet("module") {
			module = c.String("module")
		}

		if c.NArg() < 1 {
			return cli.ShowCommandHelp(
				c, "startproject",
			)
		}

		var projectName = c.Args().Get(0)
		if !strings.HasPrefix(projectPath, "./") && projectPath != "." {
			projectPath = filepath.Join(
				projectPath, projectName,
			)
			os.MkdirAll(projectPath, 0755)
		}

		var projectData = Project{
			ModulePath:  module,
			ProjectName: projectName,
		}

		logger.Infof(
			"Creating project in %q with module path %q",
			projectPath, module,
		)

		err = copyProjectFiles(
			projectFS, projectPath, projectData,
		)
		if err != nil {
			return err
		}

		logger.Infof(
			"Executing `go mod init %q` in %q",
			module, projectPath,
		)
		var cmd = exec.Command("go", "mod", "init", module)
		cmd.Dir = projectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		latest, err := fetchLatestVersion()
		if err != nil {
			return err
		}

		logger.Infof(
			"Executing `go get -u %s@%s` in %q",
			GO_DJANGO_PACKAGE, latest, projectPath,
		)
		cmd = exec.Command("go", "get", "-u", fmt.Sprintf(
			"%s%s@%s", GITHUB_PREFIX_URL, GO_DJANGO_PACKAGE, latest,
		))
		cmd.Dir = projectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		logger.Infof(
			"Executing 'go mod tidy' in %q",
			projectPath,
		)
		cmd = exec.Command("go", "mod", "tidy")
		cmd.Dir = projectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	},
}

type Project struct {
	ModulePath  string
	ProjectName string
}
