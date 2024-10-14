package forms

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

//go:embed assets/**
var formTemplates embed.FS

func InitTemplateLibrary() {
	var templates, err = fs.Sub(formTemplates, "assets/templates")
	assert.True(err == nil, "failed to get form templates")

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
}

func init() {
	InitTemplateLibrary()
}
