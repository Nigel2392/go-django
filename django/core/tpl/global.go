package tpl

import (
	"html/template"
	"io"
	"io/fs"
	"strings"
)

var Global Renderer

func init() {
	Global = NewRenderer()
}

func AddFS(fs fs.FS, matches func(string) bool) {
	Global.AddFS(fs, matches)
}

func Funcs(funcs template.FuncMap) {
	Global.Funcs(funcs)
}

func Processors(funcs ...func(RequestContext)) {
	Global.Processors(funcs...)
}

func Bases(key string, path ...string) error {
	return Global.Bases(key, path...)
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
