package views

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

type BaseView struct {
	AllowedMethods  []string
	BaseTemplateKey string
	TemplateName    string
	GetContextFn    func(req *http.Request) (ctx.Context, error)
}

// Placeholder method, will never get called.
func (v *BaseView) ServeXXX(w http.ResponseWriter, req *http.Request) {
}

func (v *BaseView) Methods() []string {
	return v.AllowedMethods
}

func (v *BaseView) GetContext(req *http.Request) (ctx.Context, error) {
	if v.GetContextFn != nil {
		return v.GetContextFn(req)
	}
	return ctx.RequestContext(req), nil
}

func (v *BaseView) GetTemplate(req *http.Request) string {
	return v.TemplateName
}

func (v *BaseView) Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) error {
	return tpl.FRender(w, context, v.BaseTemplateKey, templateName)
}
