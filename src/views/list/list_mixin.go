package list

import (
	"fmt"
	"net/http"
	"slices"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/views"
)

var (
	_ views.View                                  = (*ListObjectMixin[attrs.Definer])(nil)
	_ listView__MixinContextGetter[attrs.Definer] = (*ListObjectMixin[attrs.Definer])(nil)
)

type ListObjectMixin[T attrs.Definer] struct {
	ListView    *View[T]
	List        StringRenderer
	FormOptions []func(forms.Form)
}

func (m *ListObjectMixin[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (m *ListObjectMixin[T]) GetContext(r *http.Request, view views.View, qs *queries.QuerySet[T], viewCtx ctx.Context) (ctx.Context, error) {
	var listGetter, ok = view.(listView__ListGetter[T])
	if !ok {
		return viewCtx, nil
	}

	var (
		err     error
		columns []ListColumn[T]
	)
	if colGetter, ok := view.(listView__ColumnGetter[T]); ok {
		columns, err = colGetter.GetListColumns(r)
	} else {
		columns, err = m.ListView.GetListColumns(r)
	}
	if err != nil {
		return viewCtx, err
	}

	var pageObject = PageFromContext[T](r.Context())
	list, err := listGetter.GetList(r, pageObject, columns, viewCtx)
	if err != nil {
		return viewCtx, err
	}

	m.List = list
	viewCtx.Set("view_list", m.List)
	return viewCtx, nil
}

func (m *ListObjectMixin[T]) Hijack(w http.ResponseWriter, r *http.Request, view views.View, qs *queries.QuerySet[T], viewCtx ctx.Context) (http.ResponseWriter, *http.Request, error) {

	var listObj, ok = m.List.(*List[T])
	if !ok {
		return w, r, nil
	}

	var clone = slices.Clone(m.FormOptions)
	if r.Method == http.MethodPost {
		clone = append(
			clone,
			forms.WithRequestData(http.MethodPost, r),
		)
	}

	var form = listObj.Form(clone...)
	if form == nil {
		return w, r, nil
	}

	var forms = form.Forms()
	if len(forms) == 0 {
		return w, r, nil
	}

	if r.Method != http.MethodPost {
		viewCtx.Set("view_list_form", form)
		return w, r, nil
	}

	var isValid = form.IsValid()
	if !isValid {
		var errs = form.Errors()
		if len(errs) > 0 {
			messages.Error(r, fmt.Sprint(errs))
		}
		var boundErrs = form.BoundErrors()
		if boundErrs != nil {
			for head := boundErrs.Front(); head != nil; head = head.Next() {
				messages.Error(r, fmt.Sprintf("Field %q: %s", head.Key, head.Value))
			}
		}
		viewCtx.Set("view_list_form", form)
		return w, r, nil
	}

	var cleanedData = form.CleanedData()
	var includedFields = make([]string, 0, len(cleanedData))
	for _, col := range listObj.Columns {
		var editableCol, ok = col.(ListEditableColumn[T])
		if !ok {
			continue
		}

		includedFields = append(
			includedFields,
			editableCol.FieldName(),
		)
	}

	var models = make([]T, 0, len(forms))
	for _, group := range listObj.groups {
		var instance = group.Row()

		models = append(models, instance)

		var defs = instance.FieldDefs()
		var pk = attrs.PrimaryKey(instance)
		rowForm, ok := forms[pk]
		if !ok {
			continue
		}

		rowData, ok := cleanedData[pk]
		if !ok {
			return nil, nil, errors.ValueError.Wrapf(
				"no cleaned data found for instance with primary key %v", pk,
			)
		}

		var fieldsMap = rowForm.FieldMap()
		if fieldsMap.Len() == 0 {
			continue
		}

		for head := fieldsMap.Front(); head != nil; head = head.Next() {
			var fieldName = head.Value.Name()
			var attrField, ok = defs.Field(fieldName)
			if !ok {
				except.Fail(
					http.StatusInternalServerError,
					errors.FieldNotFound.Wrapf(
						"field %q not found in model %T", fieldName, instance,
					),
				)
			}

			fieldValue, ok := rowData[fieldName]
			if !ok {
				return nil, nil, errors.ValueError.Wrapf(
					"no field value found for field %q in instance with primary key %v", fieldName, pk,
				)
			}

			if err := attrField.SetValue(fieldValue, false); err != nil {
				return nil, nil, err
			}
		}
	}

	if len(models) > 0 {
		updated, err := qs.
			Select(nil).
			Select(attrutils.InterfaceList(includedFields)...).
			BulkUpdate(models)
		if err != nil {
			if errors.Is(err, errors.ValueError) || errors.Is(err, errs.ErrInvalidValue) {
				messages.Error(r, err.Error())
				viewCtx.Set("view_list_form", form)
				return w, r, nil
			}
			return nil, nil, err
		}

		messages.Success(r, trans.T(
			r.Context(), "%d items updated successfully", updated,
		))
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
	return nil, nil, nil
}

type ListExportMixin[T attrs.Definer] struct {
	ListView       *View[T]
	ExportFields   []string
	ChangeQuerySet func(r *http.Request, qs *queries.QuerySet[T]) (*queries.QuerySet[T], error)
	Export         func(w http.ResponseWriter, r *http.Request, m *ListExportMixin[T], qs *queries.QuerySet[T], fields []attrs.FieldDefinition, values [][]interface{}) error
}

func (m *ListExportMixin[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (m *ListExportMixin[T]) Hijack(w http.ResponseWriter, r *http.Request, view views.View, qs *queries.QuerySet[T], viewCtx ctx.Context) (http.ResponseWriter, *http.Request, error) {
	if m.Export == nil || r.Method == http.MethodGet || r.URL.Query().Get("_export") != "1" {
		return w, r, nil
	}

	var (
		meta      = attrs.GetModelMeta(m.ListView.Model)
		defs      = meta.Definitions()
		fieldsLen = len(m.ExportFields)
	)
	if fieldsLen == 0 {
		fieldsLen = defs.Len()
	}

	var fieldNames = make([]any, fieldsLen)
	if len(m.ExportFields) > 0 {
		for i, fieldName := range m.ExportFields {
			fieldNames[i] = fieldName
		}
	} else {
		for i, field := range defs.Fields() {
			fieldNames[i] = field.Name()
		}
	}

	qs = qs.Select(fieldNames...)

	var (
		err        error
		qsInfo     expr.QueryInformation
		valuesList [][]interface{}
	)

	if m.ChangeQuerySet != nil {
		if qs, err = m.ChangeQuerySet(r, qs); err != nil {
			return nil, nil, err
		}
	}

	qsInfo = qs.Peek()

	valuesList, err = qs.ValuesList()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			return w, r, nil
		}
		return nil, nil, err
	}

	return nil, nil, m.Export(w, r, m, qs, qsInfo.Select, valuesList)
}
