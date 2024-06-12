package editor

import (
	"embed"
	"io/fs"
)

//go:embed static/**
var _editorJS_FS embed.FS
var editorJS_FS fs.FS

func init() {
	editorJS_FS, _ = fs.Sub(_editorJS_FS, "static")
}
