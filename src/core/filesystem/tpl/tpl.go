package tpl

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/pkg/errors"
)

type Renderer interface {
	Add(cfg Config)
	Processors(funcs ...func(any))
	RequestProcessors(funcs ...func(ctx.ContextWithRequest))
	Render(buffer io.Writer, data any, appKey string, path ...string) error
	Funcs(funcs template.FuncMap)
}

type Config struct {
	AppName string
	FS      fs.FS
	Bases   []string
	Matches func(path string) bool
	Funcs   template.FuncMap
}

type templates struct {
	*Config
}

func (t *templates) Lt(other *templates) bool {
	return t.AppName < other.AppName
}

type TemplateRenderer struct {
	configs         []*templates
	cache           map[string]*templateObject
	ctxFuncs        []func(any)
	requestCtxFuncs []func(ctx.ContextWithRequest)
	funcs           template.FuncMap
	fs              *filesystem.MultiFS
}

func NewRenderer() *TemplateRenderer {
	var r = &TemplateRenderer{
		funcs:           make(template.FuncMap),
		cache:           make(map[string]*templateObject),
		ctxFuncs:        make([]func(any), 0),
		requestCtxFuncs: make([]func(ctx.ContextWithRequest), 0),
		fs:              filesystem.NewMultiFS(),
	}
	r.Funcs(template.FuncMap{
		"include": func(context any, baseKey string, path ...string) template.HTML {
			var html, err = Render(context, baseKey, path...)
			if err != nil {
				return template.HTML(err.Error())
			}
			return html
		},
		"add": func(x, y int) any {
			return x + y
		},
		"sub": func(y, x int) any {
			return x - y
		},
		"safe": func(v any) template.HTML {
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

func (r *TemplateRenderer) Processors(funcs ...func(any)) {
	r.ctxFuncs = append(r.ctxFuncs, funcs...)
}

func (r *TemplateRenderer) RequestProcessors(funcs ...func(ctx.ContextWithRequest)) {
	r.requestCtxFuncs = append(r.requestCtxFuncs, funcs...)
}

func (r *TemplateRenderer) Add(cfg Config) {
	//_, ok := r.configs[cfg.AppName]
	//assert.False(ok, "config '%s' already exists", cfg.AppName)

	var config = &templates{
		Config: &cfg,
	}

	r.fs.Add(cfg.FS, cfg.Matches)
	r.configs = append(r.configs, config)
}

type templateObject struct {
	name  string
	paths []string
	cfg   *templates
	t     *template.Template
}

func (t *templateObject) Execute(w io.Writer, data any) error {
	var clone, err = t.t.Clone()
	if err != nil {
		return errors.Wrap(err, "failed to clone template")
	}

	var name = getTemplateName(t.paths[0])
	clone = clone.Lookup(name)
	return clone.ExecuteTemplate(w, name, data)
}

func getTemplateName(path string) string {
	name := filepath.Base(
		filepath.ToSlash(path),
	)

	name = strings.TrimSuffix(
		name, filepath.Ext(name),
	)
	return name
}

func (r *TemplateRenderer) getTemplate(baseKey string, path ...string) (*templateObject, error) {

	assert.False(
		len(path) == 0 && baseKey == "",
		"path is required",
	)

	if len(path) == 0 {
		path = []string{baseKey}
		baseKey = ""
	}

	var name = getTemplateName(path[0])
	if tmpl, ok := r.cache[path[0]]; ok && tmpl != nil {
		var clone, err = tmpl.t.Clone()
		if err != nil {
			return nil, errors.Wrap(err, "failed to clone template")
		}

		return &templateObject{
			name:  tmpl.name,
			cfg:   tmpl.cfg,
			paths: tmpl.paths,
			t:     clone,
		}, nil
	}

	// Check if the templates for this path have already been cached
	var cfg *templates
	for _, c := range r.configs {
		if baseKey == c.AppName || c.Matches != nil && c.Matches(path[0]) {
			cfg = c
			break
		}
	}

	if cfg == nil {
		return nil, errors.Errorf(
			"no config found for template %s", path[0],
		)
	}

	var funcMap = make(template.FuncMap)
	maps.Copy(funcMap, r.funcs)
	maps.Copy(funcMap, cfg.Funcs)

	tmpl := template.New(name)
	tmpl = tmpl.Funcs(funcMap)

	var tpls []string
	if baseKey != "" {
		tpls = make([]string, 0, len(cfg.Bases)+len(path))
		tpls = append(tpls, cfg.Bases...)
		tpls = append(tpls, path...)
	} else {
		tpls = path
	}

	tmpl, err := tmpl.ParseFS(r.fs, tpls...)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse template %s", path,
		)
	}

	var t = &templateObject{
		name:  name,
		cfg:   cfg,
		paths: tpls,
		t:     tmpl,
	}

	r.cache[path[0]] = t

	return t, nil
}

func (r *TemplateRenderer) Render(b io.Writer, context any, baseKey string, path ...string) error {
	var tmpl, err = r.getTemplate(baseKey, path...)
	if err != nil {
		return errors.Wrap(err, "failed to get template")
	}

	if context != nil {
		for _, f := range r.ctxFuncs {
			assert.False(f == nil, "nil context function")
			f(context)
		}

		if requestContext, ok := context.(ctx.ContextWithRequest); ok {
			if requestContext.Request() == nil {
				goto render
			}
			for _, f := range r.requestCtxFuncs {
				assert.False(f == nil, "nil context function")
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
