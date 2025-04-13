package openauth2

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/views"
)

func (oa *OpenAuth2AppConfig) AdminLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Handle the authentication logic here
	var v = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "admin",
		TemplateName:    "oauth2/admin_login.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = ctx.RequestContext(req)
			context.Set("oauth2", oa)
			return context, nil
		},
	}
	views.Invoke(v, w, r)
}
