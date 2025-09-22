package main

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/urfave/cli/v2"
)

var startAppCommand = &cli.Command{
	Name:        "startapp",
	Usage:       "Start a new GO-Django application",
	Description: "Create a new application in the project with the given name",
	UsageText:   "go-django startapp [appname]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dir",
			Aliases: []string{"d"},
			Usage:   "The directory to create the app in",
		},
	},
	Action: func(c *cli.Context) error {
		var projectFS, err = fs.Sub(assetFiles, "assets/templates/app")
		if err != nil {
			return err
		}

		var projectPath string
		var directory string = c.String("dir")
		if directory != "" {
			projectPath = directory
			os.MkdirAll(projectPath, 0755)
		} else {

			// Create in current directory
			// If a 'src' directory exists, use that instead.
			projectPath, err = os.Getwd()
			if err != nil {
				return err
			}

			if _, err := os.Stat(filepath.Join(projectPath, "src")); err != nil {
				if os.IsNotExist(err) {
					goto setupModule
				}
			}

			projectPath = filepath.Join(
				projectPath, "src",
			)
		}

		// Go-To label for for copying project files
	setupModule:
		if c.NArg() < 1 {
			return cli.ShowCommandHelp(
				c, "startapp",
			)
		}

		var appName = c.Args().Get(0)
		var app = Application{
			AppName: appName,
		}

		logger.Infof(
			"Creating application %q in directory %q",
			appName, projectPath,
		)

		return copyProjectFiles(
			projectFS, projectPath, app,
		)
	},
}

type Application struct {
	AppName string
}
