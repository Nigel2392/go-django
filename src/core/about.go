package core

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func About(w http.ResponseWriter, r *http.Request) {
	var session = sessions.Retrieve(r)

	fmt.Println(session.Get("page_key"))
	session.Set("page_key", "Last visited the about page")

	var err = tpl.FRender(
		w, http_.Context(r),
		"core", "core/about.tmpl",
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
