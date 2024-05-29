package urls

import (
	"net/http"

	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/mux"
)

type PathInfo struct {
	Methods []string
	Pattern string
}

// Shortcut to create a new PathInfo object for a URL pattern
func P(path string, methods ...string) *PathInfo {
	if len(methods) == 0 {
		methods = []string{mux.ANY}
	}
	return &PathInfo{Methods: methods, Pattern: path}
}

// Shortcut to create a new PathInfo object for a URL pattern with an empty path
func M(methods ...string) *PathInfo {
	return P("", methods...)
}

type URLPattern struct {
	Methods []string
	Pattern string
	Handler http.Handler
	Name    string
}

func Pattern(p *PathInfo, handler http.Handler, name ...string) *URLPattern {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	return &URLPattern{Methods: p.Methods, Pattern: p.Pattern, Handler: handler, Name: n}
}

func (p *URLPattern) Register(m http_.Mux) {
	for _, method := range p.Methods {
		m.Handle(method, p.Pattern, p.Handler, p.Name)
	}
}
