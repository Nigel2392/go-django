package admin

import (
	"context"
	"html/template"
	"net/http"
	"runtime/debug"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux/middleware/authentication"
)

// AdminPageComponent is an interface for custom page components
//
// It allows for custom page components to be added to the page
// dynamically. The ordering of the components is determined by the
// Ordering method.
type AdminPageComponent interface {
	HTML() template.HTML
	Ordering() int
	Media() media.Media
}

type componentWrapper struct {
	AdminPageComponent
}

func (c *componentWrapper) HTML() template.HTML {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Recovered from panic in AdminPageComponent: %v\n%s", r, debug.Stack())
		}
	}()
	return c.AdminPageComponent.HTML()
}

func getFuncTyped[T any](text any, request *http.Request, fallBack func(ctx context.Context) T) func(ctx context.Context) T {
	switch t := text.(type) {
	case T:
		return func(ctx context.Context) T {
			return t
		}

	case func(ctx context.Context) T:
		return t

	case func(*http.Request) T:
		return func(ctx context.Context) T {
			return t(request)
		}
	}
	return fallBack
}

func init() {
	goldcrest.Register(RegisterHomePageComponentHook, 0, RegisterHomePageComponentHookFunc(func(r *http.Request, a *AdminApplication) AdminPageComponent {
		var panelFuncs = goldcrest.Get[RegisterHomePageDisplayPanelHookFunc](RegisterHomePageDisplayPanelHook)
		var boundPanels = make([]*boundDisplayPanel, 0, len(panelFuncs))
		for _, fn := range panelFuncs {
			var panels = fn(r, a)
			for i, panel := range panels {
				var count, err = panel.QuerySet(r).Count()
				if err != nil {
					logger.Errorf("Error counting panel items for panel %d: %v", i, err)
					continue
				}

				if panel.IsShown != nil && !panel.IsShown(r) {
					continue
				}

				boundPanels = append(boundPanels, &boundDisplayPanel{
					DisplayPanel: &panel,
					r:            r,
					count:        count,
				})
			}
		}

		if len(boundPanels) == 0 {
			return nil
		}

		return &homePageDisplayPanels{panels: boundPanels, r: r}
	}))
}

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
					r.Context(),
					"Welcome to the Go-Django admin dashboard, %s!",
					attrs.ToString(user),
				)
			},
		)

		// Set up media files for the home page template
		var homeMedia media.Media = media.NewMedia()
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
			breadCrumbs = hook(req, AdminSite, breadCrumbs)
		}

		var actions = make([]Action, 0)
		var actionHooks = goldcrest.Get[RegisterHomePageActionHookFunc](RegisterHomePageActionHook)
		for _, hook := range actionHooks {
			actions = hook(req, AdminSite, actions)
		}

		// Add custom home page components
		var componentHooks = goldcrest.Get[RegisterHomePageComponentHookFunc](RegisterHomePageComponentHook)
		var components = make([]AdminPageComponent, 0)
		for _, hook := range componentHooks {
			var component = hook(req, AdminSite)
			if component != nil {
				components = append(components, &componentWrapper{component})
			}
		}
		components = sortComponents(components)

		context.SetPage(PageOptions{
			TitleFn:     getFuncTyped[string](pageTitle, req, nil),
			SubtitleFn:  getFuncTyped[string](pageSubtitle, req, nil),
			BreadCrumbs: breadCrumbs,
			Actions:     actions,
			MediaFn: func() media.Media {
				for _, component := range components {
					var media = component.Media()
					if media != nil {
						homeMedia = homeMedia.Merge(media)
					}
				}
				return homeMedia
			},
		})

		// Order the components by their ordering
		context.Set("Components", components)

		if logoPath != "" {
			context.Set("LogoPath", logoPath)
		}

		return context, nil
	},
}
