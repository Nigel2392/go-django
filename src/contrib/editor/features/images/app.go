package images

import (
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"

	_ "embed"
)

const defaultMaxBytes uint = 1024 * 1024 * 32 // 32MB

type (
	Options struct {
		MediaBackend     mediafiles.Backend
		MediaDir         string
		MaxByteSize      uint
		AllowedFileExts  []string
		AllowedMimeTypes []string
	}

	AppConfig struct {
		*apps.DBRequiredAppConfig
		Options *Options
	}
)

var app *AppConfig

func NewAppConfig(opts *Options) *AppConfig {
	if app == nil {
		app = &AppConfig{
			DBRequiredAppConfig: apps.NewDBAppConfig(
				"images",
			),
		}
	}

	//	if opts.MediaDir == "" {
	//		opts.MediaDir = "images"
	//	}

	if opts.MaxByteSize == 0 {
		opts.MaxByteSize = defaultMaxBytes
	}

	app.Options = opts

	return app
}

func (c *AppConfig) MediaBackend() mediafiles.Backend {
	if c.Options.MediaBackend == nil {
		c.Options.MediaBackend = mediafiles.GetDefault()
	}
	return c.Options.MediaBackend
}

func (c *AppConfig) MediaDir() string {
	return c.Options.MediaDir
}

func (c *AppConfig) MaxByteSize() uint {
	if c.Options.MaxByteSize == 0 {
		return defaultMaxBytes
	}
	return c.Options.MaxByteSize
}

func (c *AppConfig) AllowedFileExts() []string {
	return c.Options.AllowedFileExts
}

func (c *AppConfig) AllowedMimeTypes() []string {
	return c.Options.AllowedMimeTypes
}
