package core

import (
	"net/http"

	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/django/core/tpl"
)

func About(w http.ResponseWriter, r *http.Request) {
	var err = tpl.FRender(
		w, http_.Context(r),
		"core", "core/about.tmpl",
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
