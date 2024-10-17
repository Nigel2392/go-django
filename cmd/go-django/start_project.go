package main

import (
	"io/fs"
	"os"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/urfave/cli/v2"
)

var startProjectCommand = &cli.Command{
	Name:  "startproject",
	Usage: "Start a new GO-Django project",
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
		var projectPath string
		if c.String("dir") != "" {
			projectPath = c.String("dir")
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

		var projectData = Project{
			ModulePath: module,
		}

		logger.Infof(
			"Creating project in %q with module path %q",
			projectPath, module,
		)

		return copyProjectFiles(
			projectFS, projectPath, projectData,
		)
	},
}

type Project struct {
	ModulePath string
}
