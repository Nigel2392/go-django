package tpl

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-signals"
	"github.com/pkg/errors"
)

type TemplateFunc = any

type Template interface {
	Name() string
	Execute(w io.Writer, data any) error
}

type ContextBinder interface {
	BindContext(r *TemplateRenderer) any
}

type Renderer interface {
	Add(cfg Config)
	Processors(funcs ...func(any))
	Override(funcs ...func(any) (any, error))
	RequestProcessors(funcs ...func(ctx.ContextWithRequest))
	FirstRender() signals.Signal[*TemplateRenderer]
	GetTemplate(baseKey string, path ...string) Template
	GetTemplateFromString(name string, content string, funcs template.FuncMap) Template
	Render(buffer io.Writer, data any, appKey string, path ...string) error
	Funcs(funcs template.FuncMap)
	RequestFuncs(funcs func(*http.Request) template.FuncMap)
}

type Config struct {
	AppName      string
	FS           fs.FS
	Bases        []string
	Matches      func(path string) bool
	Funcs        template.FuncMap
	RequestFuncs func(*http.Request) template.FuncMap
}

func MergeConfig(dst, src *Config) *Config {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	dst.FS = filesystem.NewMultiFS(
		dst.FS,
		src.FS,
	)

	dst.Bases = append(dst.Bases, src.Bases...)

	switch {
	case dst.Matches == nil:
		dst.Matches = src.Matches
	case dst.Matches != nil && src.Matches != nil:
		dst.Matches = filesystem.MatchOr(
			dst.Matches,
			src.Matches,
		)
	}

	var fm = make(template.FuncMap, len(dst.Funcs)+len(src.Funcs))
	maps.Copy(fm, dst.Funcs)
	maps.Copy(fm, src.Funcs)
	dst.Funcs = fm

	var reqFuncs []func(*http.Request) template.FuncMap
	reqFuncs = append(reqFuncs, dst.RequestFuncs)
	reqFuncs = append(reqFuncs, src.RequestFuncs)
	dst.RequestFuncs = func(r *http.Request) template.FuncMap {
		var fm = make(template.FuncMap)
		for _, fn := range reqFuncs {
			if fn != nil {
				maps.Copy(fm, fn(r))
			}
		}
		return fm
	}

	return dst
}

type templates struct {
	*Config
}

func (t *templates) Lt(other *templates) bool {
	return t.AppName < other.AppName
}

type TemplateRenderer struct {
	configs           []*templates
	configMap         map[string]*templates
	cache             map[string]*template.Template
	ctxFuncs          []func(any)
	ctxOverrides      []func(any) (any, error)
	requestCtxFuncs   []func(ctx.ContextWithRequest)
	reqFuncs          []func(*http.Request) template.FuncMap
	funcs             template.FuncMap
	fs                *filesystem.CacheFS[*filesystem.MultiFS]
	firstRender       atomic.Bool
	firstRenderSignal signals.Signal[*TemplateRenderer]
}

func NewRenderer() *TemplateRenderer {
	var r = &TemplateRenderer{
		configs:           make([]*templates, 0),
		configMap:         make(map[string]*templates),
		funcs:             make(template.FuncMap),
		cache:             make(map[string]*template.Template),
		ctxFuncs:          make([]func(any), 0),
		requestCtxFuncs:   make([]func(ctx.ContextWithRequest), 0),
		fs:                filesystem.NewCacheFS(filesystem.NewMultiFS()),
		firstRenderSignal: signals.New[*TemplateRenderer]("tpl.FirstRender"),
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
			var sb = new(strings.Builder)
			switch v := v.(type) {
			case string:
				sb.WriteString(v)
			case fmt.Stringer:
				sb.WriteString(v.String())
			case map[string]string:
				for k, v := range v {
					fmt.Fprintf(sb, "%s=%s", k, strconv.Quote(v))
				}
			case map[string]any:
				for k, v := range v {
					switch v := v.(type) {
					case string:
						fmt.Fprintf(sb, "%s=%s", k, strconv.Quote(v))
					case []string:
						fmt.Fprintf(sb, "%s=%s", k, strconv.Quote(strings.Join(v, " ")))
					}
				}
			default:
				fmt.Fprint(sb, v)
			}

			return template.HTMLAttr(sb.String())
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

func (r *TemplateRenderer) RequestFuncs(funcs func(*http.Request) template.FuncMap) {
	r.reqFuncs = append(r.reqFuncs, funcs)
}

func (r *TemplateRenderer) Override(funcs ...func(any) (any, error)) {
	r.ctxOverrides = append(r.ctxOverrides, funcs...)
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

	r.fs.FS.Add(cfg.FS, cfg.Matches)
	r.configs = append(r.configs, config)

	if config.AppName != "" {
		r.configMap[config.AppName] = config
	}

	r.fs.Changed()
}

func (r *TemplateRenderer) FirstRender() signals.Signal[*TemplateRenderer] {
	return r.firstRenderSignal
}

func (r *TemplateRenderer) GetTemplate(baseKey string, path ...string) Template {
	path, baseKey = normalizeTemplatePath(baseKey, path...)

	config, err := r.getTemplateConfig(baseKey, path)
	if err != nil {
		panic(errors.Wrapf(
			err, "failed to get template config for %s", strings.Join(path, ", "),
		))
	}

	var tpls = r.getTemplatePaths(config, path)
	return &templateObject{
		path:     path,
		name:     getTemplateName(path[0]),
		baseName: getTemplateName(tpls[0]),
		allPaths: tpls,
		config:   config,
		renderer: r,
	}
}

func (r *TemplateRenderer) GetTemplateFromString(name string, content string, funcs template.FuncMap) Template {
	return &stringTemplateObject{
		name:     name,
		renderer: r,
		content:  content,
		funcs:    funcs,
	}
}

func (r *TemplateRenderer) Render(buffer io.Writer, context any, baseKey string, path ...string) error {
	if !r.firstRender.Load() {
		r.firstRender.Store(true)
		r.onFirstRender()
	}

	var tmpl = r.GetTemplate(baseKey, path...)
	return tmpl.Execute(buffer, context)
}

func (r *TemplateRenderer) onFirstRender() {
	r.firstRenderSignal.Send(r)
}

func (r *TemplateRenderer) getTemplateConfig(baseKey string, path []string) (*templates, error) {
	var cfg *templates

	if baseKey == "" {
		return nil, nil
	}

	if baseKey != "" {
		if c, ok := r.configMap[baseKey]; ok {
			return c, nil
		}
	}

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

	return cfg, nil
}

func (r *TemplateRenderer) getTemplatePaths(cfg *templates, path []string) []string {
	var tpls []string
	if cfg != nil {
		tpls = make([]string, 0, len(cfg.Bases)+len(path))
		tpls = append(tpls, cfg.Bases...)
		tpls = append(tpls, path...)
	} else {
		tpls = path
	}

	return tpls
}

func (r *TemplateRenderer) getTemplate(name string, basePath string, paths []string, config *templates, fromCache bool, fsys fs.FS) (*template.Template, error) {
	if fsys == nil {
		fsys = r.fs
	}

	if fromCache {
		if tmpl, ok := r.cache[basePath]; ok && tmpl != nil {
			return tmpl, nil
		}
	}

	var err error
	var funcMap = make(template.FuncMap)
	maps.Copy(funcMap, r.funcs)
	if config != nil {
		maps.Copy(funcMap, config.Funcs)
	}

	tmpl := template.New(name)
	tmpl = tmpl.Funcs(funcMap)

	tmpl, err = tmpl.ParseFS(fsys, paths...)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse template %s", paths,
		)
	}

	r.cache[basePath] = tmpl

	return tmpl, nil
}

func (r *TemplateRenderer) getTemplateForRequest(name string, paths []string, config *templates, req *http.Request, buildFuncs []func(*http.Request) template.FuncMap, fsys fs.FS) (*template.Template, error) {
	if fsys == nil {
		fsys = r.fs
	}

	funcMap := make(template.FuncMap)
	maps.Copy(funcMap, r.funcs)
	if config != nil {
		maps.Copy(funcMap, config.Funcs)
	}
	for _, fn := range buildFuncs {
		maps.Copy(funcMap, fn(req))
	}
	for _, fn := range r.reqFuncs {
		maps.Copy(funcMap, fn(req))
	}

	var err error
	tmpl := template.New(name)
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.ParseFS(fsys, paths...)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse templates %v", paths,
		)
	}

	return tmpl, nil
}

func (r *TemplateRenderer) getTemplateFromString(name string, content string, beforeParse ...func(*template.Template, template.FuncMap) error) (*template.Template, error) {
	funcMap := make(template.FuncMap)
	maps.Copy(funcMap, r.funcs)

	tmpl := template.New(name)
	tmpl = tmpl.Funcs(funcMap)

	for _, fn := range beforeParse {
		if err := fn(tmpl, funcMap); err != nil {
			return nil, errors.Wrapf(
				err, "failed to execute beforeParse function for template %s", name,
			)
		}
	}

	tmpl, err := tmpl.Parse(content)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to parse template %s", name,
		)
	}

	return tmpl, nil
}

func (r *TemplateRenderer) setupTemplateContext(context any) (any, *http.Request, error) {
	var request *http.Request
	if context != nil {
		for _, f := range r.ctxFuncs {
			assert.False(f == nil, "nil context function")
			f(context)
		}

		if requestContext, ok := context.(ctx.ContextWithRequest); ok {
			var req = requestContext.Request()
			if req == nil {
				goto overrideContext
			}

			request = req

			for _, f := range r.requestCtxFuncs {
				assert.False(f == nil, "nil context function")
				f(requestContext)
			}
		}
	}

overrideContext:
	var err error
	for _, f := range r.ctxOverrides {
		if f == nil {
			continue
		}

		context, err = f(context)
		if err != nil {
			return context, nil, errors.Wrap(err, "failed to override context")
		}
	}

	if binder, ok := context.(ContextBinder); ok {
		context = binder.BindContext(r)
	}

	return context, request, nil
}

func normalizeTemplatePath(baseKey string, path ...string) ([]string, string) {
	assert.False(
		len(path) == 0 && baseKey == "",
		"path is required",
	)

	if len(path) == 0 {
		path = []string{baseKey}
		baseKey = ""
	}

	return path, baseKey
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
