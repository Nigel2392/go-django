package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

func sortComponents(components []AdminPageComponent) []AdminPageComponent {
	slices.SortStableFunc(components, func(i, j AdminPageComponent) int {
		if i.Ordering() < j.Ordering() {
			return -1
		} else if i.Ordering() > j.Ordering() {
			return 1
		}
		return 0
	})
	return components
}

func FindDefinition(model any, appName ...string) *ModelDefinition {
	var app = ""
	if len(appName) > 0 {
		app = appName[0]
	}

	var modelTypeName string
	switch m := model.(type) {
	case attrs.Definer:
		var cType = contenttypes.NewContentType(m)

		if app == "" {
			app = cType.AppLabel()
		}

		modelTypeName = cType.Model()
	case string:
		modelTypeName = m
	default:
		panic(fmt.Sprintf(
			"FindDefinition requires a model of type attrs.Definer or string, got %T",
			model,
		))
	}

	if app != "" {
		app, ok := AdminSite.Apps.Get(app)
		if !ok {
			goto slowFind
		}

		m, ok := app.Models.Get(modelTypeName)
		if ok {
			return m
		}
	}

slowFind:
	for head := AdminSite.Apps.Front(); head != nil; head = head.Next() {
		var app = head.Value

		var m, ok = app.Models.Get(modelTypeName)
		if ok {
			return m
		}
	}

	return nil
}

func ReLogin(w http.ResponseWriter, r *http.Request, nextURL ...string) {
	var redirectURL = django.Reverse("admin:relogin")
	if len(nextURL) > 0 {
		redirectURL = fmt.Sprintf(
			"%s?next=%s",
			redirectURL,
			url.QueryEscape(nextURL[0]),
		)
	}
	http.Redirect(
		w, r,
		redirectURL,
		http.StatusSeeOther,
	)

}

func Home(w http.ResponseWriter, r *http.Request, errorMessage ...string) {
	if len(errorMessage) > 0 {
		messages.Error(r, errorMessage[0])
	}
	var redirectURL = django.Reverse("admin")
	http.Redirect(
		w, r,
		redirectURL,
		http.StatusSeeOther,
	)
}
