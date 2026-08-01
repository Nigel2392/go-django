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
		TemplateName:    "oauth2/admin/login.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = ctx.RequestContext(req)
			var configs = make([]*boundProviderInfo, 0, len(oa.Config.AuthConfigurations))
			for _, cnf := range oa.Config.AuthConfigurations {
				configs = append(configs, &boundProviderInfo{
					r:   r,
					cnf: cnf.ProviderInfo,
				})
			}
			context.Set("oauth2", oa)
			context.Set("configs", configs)
			return context, nil
		},
	}
	views.Invoke(v, w, r)
}
