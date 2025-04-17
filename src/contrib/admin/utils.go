package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
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

func FindDefinition(model attrs.Definer) *ModelDefinition {
	var modelType = reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	for head := AdminSite.Apps.Front(); head != nil; head = head.Next() {
		var app = head.Value
		for front := app.Models.Front(); front != nil; front = front.Next() {
			var modelDef = front.Value
			var typ = modelDef.rModel()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}

			if typ == modelType {
				return modelDef
			}
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
