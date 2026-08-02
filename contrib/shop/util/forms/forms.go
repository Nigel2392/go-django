package forms

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

func NewAdminForm[MODEL attrs.Definer](r *http.Request, instance MODEL, getPanels func(MODEL) []admin.Panel) *admin.AdminForm[*modelforms.BaseModelForm[MODEL], MODEL] {
	var panels = getPanels(instance)
	var form = modelforms.NewBaseModelForm[MODEL](r.Context(), instance)
	var adminForm = admin.NewAdminForm(r, form, panels...)
	return adminForm
}
