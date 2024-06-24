package action_menu

import (
	"net/http"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/a-h/templ"
)

type Item interface {
	Name() string
	Order() int
	Component(r *http.Request, obj attrs.Definer) templ.Component
}

type ActionMenuItem struct {
	ID   string
	Name string
}
