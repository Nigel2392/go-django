package settings

import (
	"embed"
	"io/fs"
)

var (
	//go:embed assets/**
	settingsFS embed.FS

	tplFS    fs.FS
	staticFS fs.FS

	//go:embed migrations/*
	migrationFS embed.FS
)

func init() {
	var err error
	tplFS, err = fs.Sub(settingsFS, "assets/templates")
	if err != nil {
		panic(err)
	}

	staticFS, err = fs.Sub(settingsFS, "assets/static")
	if err != nil {
		panic(err)
	}
}
