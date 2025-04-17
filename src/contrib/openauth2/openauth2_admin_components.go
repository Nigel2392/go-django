package openauth2

import (
	"html/template"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
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
}

func newProviderComponent(config *OpenAuth2AppConfig) func(r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition) admin.AdminPageComponent {
	return func(r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition) admin.AdminPageComponent {
		return &providerComponent{
			Request:   r,
			AdminSite: adminSite,
			App:       app,
			Config:    config,
		}
	}
}

func (p *providerComponent) Media() media.Media {
	var m = media.NewMedia()
	m.AddCSS(media.CSS(django.Static("oauth2/admin/css/providers.css")))
	return m
}

func (p *providerComponent) HTML() template.HTML {
	var html, err = tpl.Render(p, "oauth2/admin/components/providers.tmpl")
	if err != nil {
		logger.Error("openauth2: error rendering provider component: %v", err)
		return template.HTML("")
	}
	return template.HTML(html)
}

func (p *providerComponent) Ordering() int {
	return 0
}
