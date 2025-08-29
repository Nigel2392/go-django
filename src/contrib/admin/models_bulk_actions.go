package admin

import (
	"fmt"
	"net/http"
	"net/url"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/a-h/templ"
)

type BulkAction interface {
	Order() int
	Name() string
	Button() templ.Component
	HasPermission(r *http.Request, model *ModelDefinition) bool
	Execute(w http.ResponseWriter, r *http.Request, model *ModelDefinition, qs *queries.QuerySet[attrs.Definer]) (int, error)
}

type BaseBulkAction struct {
	ID            string
	Btn           templ.Component
	Ordering      int
	Permissions   []string
	PermissionsFn func(r *http.Request, model *ModelDefinition) bool
	Func          func(w http.ResponseWriter, r *http.Request, model *ModelDefinition, qs *queries.QuerySet[attrs.Definer]) (int, error)
}

func (b *BaseBulkAction) Order() int {
	return b.Ordering
}

func (b *BaseBulkAction) Name() string {
	return b.ID
}

func (b *BaseBulkAction) Button() templ.Component {
	return b.Btn
}

func (b *BaseBulkAction) HasPermission(r *http.Request, model *ModelDefinition) bool {
	if b.PermissionsFn != nil && !b.PermissionsFn(r, model) {
		return false
	}

	if len(b.Permissions) == 0 {
		return true
	}

	return permissions.HasPermission(r, b.Permissions...)
}

func (b *BaseBulkAction) Execute(w http.ResponseWriter, r *http.Request, model *ModelDefinition, qs *queries.QuerySet[attrs.Definer]) (int, error) {
	return b.Func(w, r, model, qs)
}

func BulkActionButton(text any, typ components.ClassType, actionName string, requiresSelected bool) templ.Component {

	var attrs = map[string]any{
		"type":  "submit",
		"name":  "list_action",
		"value": actionName,
	}

	if requiresSelected {
		attrs["data-bulk-actions-target"] = "execute"
	}

	return components.Button(components.ButtonConfig{
		Text:  trans.GetTextFunc(text),
		Type:  typ,
		Attrs: attrs,
	})
}

var (
	BulkActionDelete = &BaseBulkAction{
		ID:       "delete",
		Ordering: 100,
		Btn:      BulkActionButton(trans.S("Delete"), components.ClassTypeDanger, "delete", true),
		PermissionsFn: func(r *http.Request, model *ModelDefinition) bool {
			return !model.DisallowDelete && permissions.HasPermission(r, "admin:delete")
		},
		Func: func(w http.ResponseWriter, r *http.Request, model *ModelDefinition, qs *queries.QuerySet[attrs.Definer]) (int, error) {
			var meta = attrs.GetModelMeta(model.NewInstance())
			var defs = meta.Definitions()
			var primary = defs.Primary()

			qs = qs.Select(primary.Name())

			var rowCnt, rowIter, err = qs.IterAll()
			if err != nil {
				return 0, err
			}

			var values = make([]string, 0, rowCnt)
			for row, err := range rowIter {
				if err != nil {
					return 0, err
				}

				var defs = row.Object.FieldDefs()
				var primary = defs.Primary()
				var val, err = primary.Value()
				if err != nil {
					return 0, err
				}

				values = append(
					values,
					attrs.ToString(val),
				)
			}

			var urlValues = url.Values{
				"pk_list": values,
				"next":    []string{r.URL.RequestURI()},
			}

			var deleteURL = fmt.Sprintf(
				"%s?%s",
				django.Reverse(
					"admin:apps:model:bulk_delete",
					model.App().Name, model.GetName(),
				),
				urlValues.Encode(),
			)

			http.Redirect(
				w, r, deleteURL, http.StatusSeeOther,
			)

			return 0, nil
		},
	}
)
