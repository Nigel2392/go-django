package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms/modelforms"
)

type FormDefiner interface {
	attrs.Definer
	AdminForm(r *http.Request, app *AppDefinition, model *ModelDefinition) modelforms.ModelForm[attrs.Definer]
}
