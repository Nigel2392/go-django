package tpl

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/pkg/errors"
)

type RequestContext interface {
	ctx.Context
	Request() *http.Request
}

type Renderer interface {
	AddFS(filesys fs.FS, matches func(path string) bool)
	Funcs(funcs template.FuncMap)
	Processors(funcs ...func(RequestContext))
	Bases(key string, path ...string) error
	Render(buffer io.Writer, data any, baseKey string, path ...string) error
}

type TemplateRenderer struct {
	cache    map[string]*template.Template
	ctxFuncs []func(RequestContext)
	funcs    template.FuncMap
	fs       *MultiFS
}

func NewRenderer() *TemplateRenderer {
	var r = &TemplateRenderer{
		cache:    make(map[string]*template.Template),
		funcs:    make(template.FuncMap),
		ctxFuncs: make([]func(RequestContext), 0),
		fs:       NewMultiFS(),
	}
	r.Funcs(template.FuncMap{
		"include": func(context any, baseKey string, path ...string) template.HTML {
			var html, err = Render(context, baseKey, path...)
			if err != nil {
				return template.HTML(err.Error())
			}
			return html
		},
		"safeHTML": func(v any) template.HTML {
			var s = fmt.Sprint(v)
			return template.HTML(s)
		},
		"safeAttrs": func(v any) template.HTMLAttr {
			var attrs = fmt.Sprint(v)
			return template.HTMLAttr(attrs)
		},
		"safeURL": func(v any) template.URL {
			var url = fmt.Sprint(v)
			return template.URL(url)
		},
		"safeJS": func(v any) template.JS {
			var js = fmt.Sprint(v)
			return template.JS(js)
		},
		"safeCSS": func(v any) template.CSS {
			var css = fmt.Sprint(v)
			return template.CSS(css)
		},
	})
	return r
}

func (r *TemplateRenderer) FS() fs.FS {
	return r.fs
}

func (r *TemplateRenderer) Funcs(funcs template.FuncMap) {
	maps.Copy(r.funcs, funcs)
}

func (r *TemplateRenderer) Bases(key string, path ...string) error {
	if len(path) == 0 {
		panic("path is required")
	}
	var tmpl, _, err = r.getTemplate("", path...)
	if err != nil {
		return errors.Wrap(err, "failed to get template")
	}

	r.cache[key] = tmpl
	return nil
}

func (r *TemplateRenderer) Processors(funcs ...func(RequestContext)) {
	r.ctxFuncs = append(r.ctxFuncs, funcs...)
}

func (r *TemplateRenderer) AddFS(fs fs.FS, matches func(string) bool) {
	r.fs.Add(fs, matches)
}

func (r *TemplateRenderer) getTemplate(baseKey string, path ...string) (*template.Template, string, error) {
	if len(path) == 0 && baseKey == "" {
		panic("path is required")
	} else if len(path) == 0 {
		path = []string{baseKey}
		baseKey = ""
	}

	if tmpl, ok := r.cache[path[0]]; ok && tmpl != nil {
		tmpl, err := tmpl.Clone()
		return tmpl, "", err
	}

	var name = filepath.Base(
		filepath.ToSlash(path[0]),
	)

	name = strings.TrimSuffix(name, filepath.Ext(name))

	var (
		tmpl = r.newTemplate(name)
		err  error
	)

	if baseKey != "" {
		var baseTmpl, ok = r.cache[baseKey]
		if !ok {
			return nil, "", errors.Errorf(
				"base template %s not found", baseKey,
			)
		}
		tmpl, err = tmpl.AddParseTree(baseKey, baseTmpl.Tree.Copy())
		if err != nil {
			return nil, "", errors.Wrapf(
				err, "failed to add base template %s", baseKey,
			)
		}
	}

	tmpl, err = tmpl.ParseFS(r.fs, path...)
	if err != nil {
		return nil, "", errors.Wrapf(
			err, "failed to parse template %s", path,
		)
	}

	clone, err := tmpl.Clone()
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to clone template")
	}
	r.cache[path[0]] = clone

	return tmpl, name, nil
}

func (r *TemplateRenderer) newTemplate(name string) *template.Template {
	return template.New(name).Funcs(r.funcs)
}

func (r *TemplateRenderer) Render(b io.Writer, context any, baseKey string, path ...string) error {
	var tmpl, _, err = r.getTemplate(baseKey, path...)
	if err != nil {
		return errors.Wrap(err, "failed to get template")
	}

	if context != nil {
		if requestContext, ok := context.(RequestContext); ok {
			if requestContext.Request() == nil {
				goto render
			}
			for _, f := range r.ctxFuncs {
				if f == nil {
					panic("nil context function")
				}
				f(requestContext)
			}
		}
	}

render:
	return errors.Wrap(
		tmpl.Execute(b, context),
		"failed to render template",
	)
}
