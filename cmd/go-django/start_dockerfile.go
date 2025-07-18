package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/urfave/cli/v2"
)

var dockerCommand = &cli.Command{
	Name:  "dockerfile",
	Usage: "Generate a Dockerfile for the project",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dir",
			Aliases: []string{"d"},
			Usage:   "The directory to write the Dockerfile to",
		},
		&cli.UintFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "The port the application listens on",
		},
		&cli.StringFlag{
			Name:    "executable",
			Aliases: []string{"e"},
			Usage:   "The name of the executable",
		},
		&cli.BoolFlag{
			Name:    "scratch",
			Aliases: []string{"s"},
			Usage:   "Use a scratch image",
		},
		&cli.BoolFlag{
			Name:    "vendored",
			Aliases: []string{"v"},
			Usage:   "Use a vendored build",
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force overwrite of existing Dockerfile",
		},
	},
	Action: func(c *cli.Context) error {
		var d = NewDockerfile()
		if c.IsSet("port") {
			d.DefaultPort = uint16(c.Uint("port"))
		}
		if c.IsSet("executable") {
			d.ExecutableName = c.String("executable")
		}
		if c.IsSet("scratch") {
			d.Scratch = c.Bool("scratch")
		}
		if c.IsSet("vendored") {
			d.Vendored = c.Bool("vendored")
		}
		var (
			err   error
			dir   = c.String("dir")
			force = c.Bool("force")
		)
		if dir == "" {
			dir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		var path = filepath.Join(dir, "Dockerfile")
		if _, err := os.Stat(path); err == nil && !force {
			return cli.Exit(
				"File already exists, use --force to overwrite", 1,
			)
		}

		logger.Infof("Writing Dockerfile to %q", path)

		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = d.WriteTo(file)
		return err
	},
}

type Dockerfile struct {
	DefaultPort    uint16
	ExecutableName string
	GoVersion      string
	Scratch        bool
	Vendored       bool
}

func NewDockerfile() *Dockerfile {
	var d = &Dockerfile{}
	d.defaults()
	return d
}

func (d *Dockerfile) WriteTo(w io.Writer) (int64, error) {
	d.defaults()

	var buf = new(bytes.Buffer)
	var err = RenderTemplateFile(
		assetFiles, buf, "assets/templates/Dockerfile.tmpl", d,
	)
	if err != nil {
		return 0, err
	}

	return buf.WriteTo(w)
}

func (d *Dockerfile) defaults() {
	if d.DefaultPort == 0 {
		d.DefaultPort = 8080
	}

	var info, ok = debug.ReadBuildInfo()
	if ok {
		d.GoVersion = strings.TrimPrefix(
			info.GoVersion, "go",
		)
	}

	if d.GoVersion == "" {
		d.GoVersion = "1.24.5"
	}

	if d.ExecutableName == "" {
		d.ExecutableName = "main"
	}

	if d.DefaultPort == 0 {
		d.DefaultPort = 8080
	}
}
