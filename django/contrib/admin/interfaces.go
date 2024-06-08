package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms"
)

type FormDefiner interface {
	attrs.Definer
	AdminForm(r *http.Request, app *AppDefinition, model *ModelDefinition) forms.Form
}
