package tpl

import (
	"fmt"
	"html/template"
	"io"

	"github.com/pkg/errors"
)

var _ Template = (*templateObject)(nil)

type templateObject struct {
	name                    string
	path                    []string
	baseName                string
	renderer                *TemplateRenderer
	config                  *templates
	allPaths                []string
	template                *template.Template
	templateRequiresRequest bool
}

func (t *templateObject) Name() string {
	return t.name
}

func (t *templateObject) Execute(w io.Writer, data any) error {
	var context, request, err = t.renderer.setupTemplateContext(data)
	if err != nil {
		return errors.Wrapf(
			err, "failed to setup template context for %s / %s / %s",
			t.name, t.path[0], t.allPaths[0],
		)
	}

	// If the template is already set, we can use it directly, unless:
	// 1. The template is nil (not yet loaded)
	// 2. A request is provided, thus the template requires a request
	// 3. The template was acquired in a previouus render call and was passed a request on said call, even
	//    if the request is nil now, we still need to re-load the template to avoid security issues.
	//
	// This will cache the template for the next render call if no request is provided during this, or the next call.
	if t.template == nil || request != nil || t.templateRequiresRequest {
		if request != nil {
			t.templateRequiresRequest = true
			t.template, err = t.renderer.getTemplateForRequest(
				t.name, t.allPaths, t.config, request, t.renderer.reqFuncs,
			)
		} else {
			t.templateRequiresRequest = false
			t.template, err = t.renderer.getTemplate(
				t.name, t.path[0], t.allPaths, t.config,
			)
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to get template")
	}

	var tmpl = t.template.Lookup(t.baseName)
	if tmpl == nil {
		return errors.Errorf("template %s not found in %v",
			t.baseName, t.allPaths,
		)
	}

	if tmpl.Tree == nil {
		return fmt.Errorf(
			"template %q has no parse tree, it may not have been parsed correctly", t.name,
		)
	}

	return tmpl.ExecuteTemplate(w, t.baseName, context)
}
