package tpl

import (
	"html/template"
	"io"
	"strings"
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

func Processors(funcs ...func(RequestContext)) {
	Global.Processors(funcs...)
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
