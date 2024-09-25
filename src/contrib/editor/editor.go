package editor

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

//go:embed static/**
var _editorJS_FS embed.FS
var editorJS_FS fs.FS

func init() {
	editorJS_FS, _ = fs.Sub(_editorJS_FS, "static")
	attrs.RegisterFormFieldType(
		&EditorJSBlockData{},
		func(opts ...func(fields.Field)) fields.Field {
			return EditorJSField(nil, opts...)
		},
	)

	staticfiles.AddFS(
		editorJS_FS,
		filesystem.MatchAnd(
			filesystem.MatchPrefix("editorjs"),
			filesystem.MatchOr(
				filesystem.MatchExt(".js"),
				filesystem.MatchExt(".css"),
			),
		),
	)
}
