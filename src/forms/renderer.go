package forms

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
)

//go:embed assets/**
var formAssets embed.FS

func initTemplateLibrary() {
	var templates, err = fs.Sub(formAssets, "assets/templates")
	assert.True(err == nil, "failed to get form templates")

	static, err := fs.Sub(formAssets, "assets/static")
	assert.True(err == nil, "failed to get form static files")

	var templateConfig = tpl.Config{
		AppName: "forms",
		FS:      templates,
		Bases:   []string{},
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("forms/widgets/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".html"),
			),
		),
	}

	for _, hook := range goldcrest.Get[FormTemplateFSHook](FormTemplateFSHookName) {
		templateConfig.FS = hook(templateConfig.FS, &templateConfig)
	}

	for _, hook := range goldcrest.Get[FormTemplateStaticHook](FormTemplateStaticHookName) {
		static = hook(static)
	}

	tpl.Add(templateConfig)
	staticfiles.AddFS(static, filesystem.MatchPrefix("forms/"))
}

type (
	FormTemplateFSHook     func(fSys fs.FS, cnf *tpl.Config) fs.FS
	FormTemplateStaticHook func(fSys fs.FS) fs.FS
)

const (
	FormTemplateFSHookName     = "forms.TemplateFSHook"
	FormTemplateStaticHookName = "forms.TemplateStaticHook"
)

func init() {
	tpl.FirstRender().Listen(func(s signals.Signal[*tpl.TemplateRenderer], tr *tpl.TemplateRenderer) error {
		initTemplateLibrary()
		return nil
	})
}
