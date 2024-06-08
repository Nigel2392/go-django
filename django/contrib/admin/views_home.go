package admin

import (
	"net/http"

	"github.com/Nigel2392/django/views"
)

var HomeHandler = &views.BaseView{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "admin",
	TemplateName:    "admin/views/home.tmpl",
}
