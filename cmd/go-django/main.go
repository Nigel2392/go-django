package main

import (
	"embed"
	"os"
	"strings"
	"text/template"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/urfave/cli/v2"
)

//go:embed assets/**
var assetFiles embed.FS

var funcMap = template.FuncMap{
	"lowercase": func(s string) string {
		return strings.ToLower(s)
	},
}

func setLogLevel(level logger.LogLevel) func(c *cli.Context, b bool) error {
	return func(c *cli.Context, b bool) error {
		if b {
			logger.SetLevel(level)
		}
		return nil
	}
}

func main() {
	logger.Setup(&logger.Logger{
		Level: logger.WRN,
	})
	logger.SetOutput(
		logger.OutputAll,
		os.Stdout,
	)

	var app = &cli.App{
		Name:  "go-django",
		Usage: "A tool to help manage GO-Django projects",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose logging",
				Action:  setLogLevel(logger.INF),
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"vv"},
				Usage:   "Enable debug logging",
				Action:  setLogLevel(logger.DBG),
			},
		},
		Commands: []*cli.Command{
			dockerCommand,
			startProjectCommand,
			startAppCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Error(err)
	}

}
