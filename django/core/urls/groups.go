package urls

import (
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/mux"
)

type URLPatternGroup struct {
	Patterns []http_.URL
	Pattern  string
	Name     string
}

// Group creates a new URLPatternGroup with the given pattern and name.
// The pattern is the base path for all the patterns in the group.
// The name is the name of the group.
func Group(info ...string) *URLPatternGroup {
	var (
		pattern string
		name    string
	)

	if len(info) > 0 {
		pattern = info[0]
	}

	if len(info) > 1 {
		name = info[1]
	}

	assert.Lt(info, 3, "urls.Group: too many arguments")

	return &URLPatternGroup{
		Patterns: make([]http_.URL, 0),
		Pattern:  pattern,
		Name:     name,
	}
}

func (g *URLPatternGroup) Register(m http_.Mux) {
	var group = m.Handle(mux.ANY, g.Pattern, nil, g.Name)
	for _, pattern := range g.Patterns {
		pattern.Register(group)
	}
}

func (g *URLPatternGroup) Add(patterns ...http_.URL) *URLPatternGroup {
	g.Patterns = append(g.Patterns, patterns...)
	return g
}
