package core

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/filesystem/tpl"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func About(w http.ResponseWriter, r *http.Request) {
	var session = sessions.Retrieve(r)

	fmt.Println(session.Get("page_key"))
	session.Set("page_key", "Last visited the about page")
	var form = auth.UserLoginForm(r)

	if form.IsValid() {
		fmt.Println("Form is valid")
		var err = form.Login()
		except.Assert(err == nil, 500, err)
	} else {
		fmt.Println("Form is invalid")
	}

	var user = auth.UserFromRequest(r)
	fmt.Println(user, form.Instance)

	var context = ctx.RequestContext(r)

	context.Set("Form", form)

	var err = tpl.FRender(
		w, context,
		"core", "core/about.tmpl",
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
