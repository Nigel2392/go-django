package tpl

import (
	"html/template"
	"io"
	"strings"

	"github.com/Nigel2392/go-django/src/core/ctx"
)

var Global Renderer

func init() {
	Global = NewRenderer()
}

func Add(c Config) {
	Global.Add(c)
}

func Funcs(funcs template.FuncMap) {
	Global.Funcs(funcs)
}

func Overrides(funcs ...func(any) (any, error)) {
	Global.Override(funcs...)
}

func Processors(funcs ...func(any)) {
	Global.Processors(funcs...)
}

func RequestProcessors(funcs ...func(ctx.ContextWithRequest)) {
	Global.RequestProcessors(funcs...)
}

func FRender(b io.Writer, context any, baseKey string, path ...string) error {
	return Global.Render(b, context, baseKey, path...)
}

func Render(context any, baseKey string, path ...string) (template.HTML, error) {
	var b strings.Builder
	if err := Global.Render(&b, context, baseKey, path...); err != nil {
		return template.HTML(b.String()), err
	}
	return template.HTML(b.String()), nil
}
