package forms

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/django/core/tpl"
)

//go:embed assets/**
var formTemplates embed.FS

func init() {
	var templates, err = fs.Sub(formTemplates, "assets/templates")
	if err != nil {
		panic(err)
	}

	tpl.AddFS(templates, tpl.MatchAnd(
		tpl.MatchPrefix("widgets/"),
		tpl.MatchOr(
			tpl.MatchExt(".html"),
		),
	))
}
