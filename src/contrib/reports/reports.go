package reports

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/goldcrest"
)

func RegisterMenuItem(fn func(r *http.Request) []menu.MenuItem) {
	goldcrest.Register(
		ReportsMenuHook, 0,
		ReportsMenuHookFunc(fn),
	)

}
