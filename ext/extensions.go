package ext

import (
	gohtml "github.com/Nigel2392/go-html"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/templates/extensions"
)

type GoHTMLExtension struct {
	extensions.Base
	HTML func(r *request.Request) *gohtml.Element
}

func NewGoHTMLExtension(name string, callback func(r *request.Request) map[string]any, html func(r *request.Request) *gohtml.Element) *GoHTMLExtension {
	return &GoHTMLExtension{
		Base: extensions.Base{
			ExtensionName: name,
			Callback:      callback,
		},
		HTML: html,
	}
}

func (s *GoHTMLExtension) String(r *request.Request) string {
	return s.HTML(r).String()
}

// Returns the name of the extension.
func (s *GoHTMLExtension) Name() string {
	return s.ExtensionName
}

// Returns the template data for the extension.
func (s *GoHTMLExtension) View(r *request.Request) map[string]any {
	return s.Callback(r)
}
