package views

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
)

var httpMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

func Serve(view View) ViewHandler {
	var b = &boundView{
		view:           view,
		allowedMethods: make([]string, 0, 3),
	}

	for _, method := range httpMethods {
		if _, ok := attrs.Method[http.HandlerFunc](
			view, fmt.Sprintf("Serve%s", method),
		); ok {
			b.allowedMethods = append(b.allowedMethods, method)
		}
	}

	if methodNamer, ok := view.(MethodsView); ok {
		for _, method := range methodNamer.Methods() {
			if !slices.Contains(b.allowedMethods, method) {
				b.allowedMethods = append(b.allowedMethods, method)
			}
		}
	}

	assert.True(
		len(b.allowedMethods) > 0,
		"View must have at least one Serve method defined, I.E. ServeGET, ServePOST, etc...",
	)

	return b
}
