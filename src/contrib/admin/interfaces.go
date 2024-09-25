package admin

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

type FormDefiner interface {
	attrs.Definer
	AdminForm(r *http.Request, app *AppDefinition, model *ModelDefinition) modelforms.ModelForm[attrs.Definer]
}
