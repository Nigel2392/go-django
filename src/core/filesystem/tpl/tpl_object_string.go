package tpl

import (
	"html/template"
	"io"
	"maps"

	"github.com/pkg/errors"
)

type stringTemplateObject struct {
	name     string
	content  string
	renderer *TemplateRenderer
	funcs    template.FuncMap
}

func (t *stringTemplateObject) Name() string {
	return t.name
}

func (t *stringTemplateObject) Execute(w io.Writer, data any) error {
	var context, request, err = t.renderer.setupTemplateContext(data)
	if err != nil {
		return errors.Wrapf(err, "failed to setup template context for %s", t.name)
	}

	tmpl, err := t.renderer.getTemplateFromString(
		t.name, t.content, func(tpl *template.Template, fm template.FuncMap) error {
			if request != nil {
				for _, fn := range t.renderer.reqFuncs {
					maps.Copy(fm, fn(request))
				}
			}
			maps.Copy(fm, t.funcs)
			tpl.Funcs(fm)
			return nil
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get template from string for %s", t.name)
	}

	return tmpl.ExecuteTemplate(w, t.name, context)
}
