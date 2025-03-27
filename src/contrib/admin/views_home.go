package admin

import (
	"html/template"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux/middleware/authentication"
)

func getFuncTyped[T any](text any, request *http.Request, fallBack func() T) func() T {
	switch t := text.(type) {
	case T:
		return func() T {
			return t
		}

	case func() T:
		return t

	case func(*http.Request) T:
		return func() T {
			return t(request)
		}
	}
	return fallBack
}

const (
	APPVAR_HOME_PAGE_TITLE     = "APPVAR_HOME_PAGE_TITLE"
	APPVAR_HOME_PAGE_SUBTITLE  = "APPVAR_HOME_PAGE_SUBTITLE"
	APPVAR_HOME_PAGE_LOGO_PATH = "APPVAR_HOME_PAGE_LOGO"

	RegisterHomePageBreadcrumbHook = "admin:register_home_page_breadcrumb"
	RegisterHomePageActionHook     = "admin:register_home_page_action"
	RegisterHomePageComponentHook  = "admin:register_home_page_component"
)

type (
	Component struct {
		template.HTML
		Ordering int
	}
	RegisterHomePageBreadcrumbHookFunc = func(*http.Request, *AdminApplication, []BreadCrumb)
	RegisterHomePageActionHookFunc     = func(*http.Request, *AdminApplication, []Action)
	RegisterHomePageComponentHookFunc  = func(*http.Request, *AdminApplication) *Component
)

var HomeHandler = &views.BaseView{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: BASE_KEY,
	TemplateName:    "admin/views/home.tmpl",
	GetContextFn: func(req *http.Request) (ctx.Context, error) {
		var context = NewContext(req, AdminSite, nil)
		var user = authentication.Retrieve(req)

		// Initialize / retrieve home page intro texts
		var pageTitle = django.ConfigGet[any](
			django.Global.Settings,
			APPVAR_HOME_PAGE_TITLE,
			trans.S("Home"),
		)
		var pageSubtitle = django.ConfigGet[any](
			django.Global.Settings,
			APPVAR_HOME_PAGE_SUBTITLE,
			func(r *http.Request) string {
				return trans.T(
					"Welcome to the Go-Django admin dashboard, %s!",
					user,
				)
			},
		)

		// Set up media files for the home page template
		var homeMedia = media.NewMedia()
		homeMedia.AddCSS(media.CSS(django.Static(
			"admin/css/home.css",
		)))

		// Initialize / retrieve the home page logo's static path
		var logo = django.ConfigGet[any](
			django.Global.Settings,
			APPVAR_HOME_PAGE_LOGO_PATH,
			django.Static("admin/images/home_logo.png"),
		)
		var logoPath string
		switch t := logo.(type) {
		case string:
			logoPath = t
		case func() string:
			logoPath = t()
		case func(*http.Request) string:
			logoPath = t(req)
		}

		// Add home page breadcrumbs and actions
		var breadCrumbs = make([]BreadCrumb, 0)
		var breadcrumbHooks = goldcrest.Get[RegisterHomePageBreadcrumbHookFunc](RegisterHomePageBreadcrumbHook)
		for _, hook := range breadcrumbHooks {
			hook(req, AdminSite, breadCrumbs)
		}

		var actions = make([]Action, 0)
		var actionHooks = goldcrest.Get[RegisterHomePageActionHookFunc](RegisterHomePageActionHook)
		for _, hook := range actionHooks {
			hook(req, AdminSite, actions)
		}

		// Add custom home page components
		var componentHooks = goldcrest.Get[RegisterHomePageComponentHookFunc](RegisterHomePageComponentHook)
		var components = make([]*Component, 0)
		for _, hook := range componentHooks {
			var component = hook(req, AdminSite)
			if component != nil {
				components = append(components, component)
			}
		}

		context.SetPage(PageOptions{
			TitleFn:     getFuncTyped[string](pageTitle, req, nil),
			SubtitleFn:  getFuncTyped[string](pageSubtitle, req, nil),
			BreadCrumbs: breadCrumbs,
			Actions:     actions,
			MediaFn: func() media.Media {
				return homeMedia
			},
		})

		context.Set("Components", components)

		if logoPath != "" {
			context.Set("LogoPath", logoPath)
		}

		return context, nil
	},
}
