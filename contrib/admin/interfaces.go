package admin

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

// FormDefiner is an interface that defines a form for the admin.
//
// AdminForm is a method that returns a modelform.ModelForm[attrs.Definer] for the admin.
//
// This can be used to create a custom form for your models, for the admin site.
type FormDefiner[T attrs.Definer] interface {
	attrs.Definer
	AdminForm(r *http.Request, app *AppDefinition, model *ModelDefinition) modelforms.ModelForm[T]
}
