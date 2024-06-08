package forms

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/tpl"
)

//go:embed assets/**
var formTemplates embed.FS

func init() {
	var templates, err = fs.Sub(formTemplates, "assets/templates")
	assert.True(err == nil, "failed to get form templates")

	tpl.Add(tpl.Config{
		AppName: "forms",
		FS:      templates,
		Bases:   []string{},
		Matches: tpl.MatchAnd(
			tpl.MatchPrefix("forms/widgets/"),
			tpl.MatchOr(
				tpl.MatchExt(".html"),
			),
		),
	})
}
