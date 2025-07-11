package pages

import (
	"net/http"
	"strconv"

	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
)

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	var variables = mux.Vars(r)
	var pageID = variables.Get(PageIDVariableName)

	if !permissions.HasPermission(r, "pages:redirect") || !pageApp.useRedirectHandler {
		except.Fail(http.StatusForbidden, nil)
		return
	}

	var id, err = strconv.Atoi(pageID)
	if err != nil {
		except.Fail(http.StatusBadRequest, err)
		return
	}

	var qs = NewPageQuerySet().WithContext(r.Context())
	page, err := qs.GetNodeByID(
		int64(id),
	)
	if err != nil {
		except.Fail(http.StatusNotFound, err)
		return
	}

	var pageUrl = URLPath(page)
	http.Redirect(w, r, pageUrl, http.StatusFound)
}
