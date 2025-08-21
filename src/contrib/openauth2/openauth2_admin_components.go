package openauth2

import (
	"html/template"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/media"
)

var (
	_ admin.AdminPageComponent = (*providerComponent)(nil)
)

type providerComponent struct {
	Request   *http.Request
	AdminSite *admin.AdminApplication
	App       *admin.AppDefinition
	Config    *OpenAuth2AppConfig
	Configs   []*boundProviderInfo
}

func newProviderComponent(config *OpenAuth2AppConfig) func(r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition) admin.AdminPageComponent {
	return func(r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition) admin.AdminPageComponent {
		var authConfigs = config.Providers()
		var configs = make([]*boundProviderInfo, 0, len(authConfigs))
		for _, cnf := range authConfigs {
			configs = append(configs, &boundProviderInfo{
				r:   r,
				cnf: cnf.ProviderInfo,
			})
		}

		return &providerComponent{
			Request:   r,
			AdminSite: adminSite,
			App:       app,
			Config:    config,
			Configs:   configs,
		}
	}
}

func (p *providerComponent) Media() media.Media {
	var m = media.NewMedia()
	m.AddCSS(media.CSS(django.Static("oauth2/admin/css/providers.css")))
	return m
}

func (p *providerComponent) HTML() template.HTML {
	var context = ctx.RequestContext(p.Request)

	context.Set("component", p)

	var html, err = tpl.Render(context, "oauth2/admin/components/providers.tmpl")
	if err != nil {
		logger.Error("openauth2: error rendering provider component: %v", err)
		return template.HTML("")
	}
	return template.HTML(html)
}

func (p *providerComponent) Ordering() int {
	return 0
}
