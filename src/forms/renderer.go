package forms

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

//go:embed assets/**
var formAssets embed.FS

func InitTemplateLibrary() {
	var templates, err = fs.Sub(formAssets, "assets/templates")
	assert.True(err == nil, "failed to get form templates")

	static, err := fs.Sub(formAssets, "assets/static")
	assert.True(err == nil, "failed to get form static files")

	tpl.Add(tpl.Config{
		AppName: "forms",
		FS:      templates,
		Bases:   []string{},
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("forms/widgets/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".html"),
			),
		),
	})

	staticfiles.AddFS(static, filesystem.MatchPrefix("forms/"))
}

func init() {
	InitTemplateLibrary()
}
