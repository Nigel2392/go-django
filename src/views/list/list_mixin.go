package list

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"text/tabwriter"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
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

type FieldExportColumn[T attrs.Definer] struct {
	FieldName     string
	HasPermission func(r *http.Request) bool
	Header        func(c context.Context) string
}

func (c *FieldExportColumn[T]) IsExportable(r *http.Request) bool {
	if c.HasPermission != nil {
		return c.HasPermission(r)
	}
	return true
}

func (c *FieldExportColumn[T]) FieldNames() []string {
	return []string{c.FieldName}
}

func (c *FieldExportColumn[T]) ExportHeader(r *http.Request) string {
	if c.Header != nil {
		return c.Header(r.Context())
	}
	return c.FieldName
}

func (c *FieldExportColumn[T]) ExportValue(r *http.Request, obj T) (interface{}, error) {
	var defs = obj.FieldDefs()
	var f, ok = defs.Field(c.FieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T", c.FieldName, obj,
		)
	}
	return f.Value()
}

type ListExportMixin[T attrs.Definer] struct {
	Model          T
	Columns        []ExportColumn[T]
	GetFileName    func(r *http.Request) string
	ChangeQuerySet func(r *http.Request, qs *queries.QuerySet[T]) (*queries.QuerySet[T], error)
	Export         func(w http.ResponseWriter, r *http.Request, m *ListExportMixin[T], qs *queries.QuerySet[T], export *Export) error
}

func (m *ListExportMixin[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (m *ListExportMixin[T]) Hijack(w http.ResponseWriter, r *http.Request, view views.View, qs *queries.QuerySet[T], viewCtx ctx.Context) (http.ResponseWriter, *http.Request, error) {
	if m.Export == nil || r.Method != http.MethodGet || r.URL.Query().Get("_export") != "1" {
		return w, r, nil
	}

	var exportCols []ExportColumn[T]
	if colGetter, ok := view.(listView__ExportColumnGetter[T]); ok && len(m.Columns) == 0 {
		exportCols = colGetter.ExportColumns()
	} else {
		exportCols = m.Columns
	}

	if len(exportCols) == 0 {
		var meta = attrs.GetModelMeta(m.Model)
		var defs = meta.Definitions()
		for _, field := range queries.ForSelectAllFields[attrs.FieldDefinition](defs) {
			exportCols = append(exportCols, &FieldExportColumn[T]{
				FieldName: field.Name(),
				Header:    field.Label,
			})
		}
	}

	var filteredCols = make([]ExportColumn[T], 0, len(exportCols))
	for _, col := range exportCols {
		if !col.IsExportable(r) {
			continue
		}

		filteredCols = append(filteredCols, col)
	}

	if len(filteredCols) == 0 {
		logger.Warnf("No exportable columns found for model %T in list view", new(T))
		return w, r, nil
	}

	var fieldNames = make([]string, 0, len(filteredCols))
	for _, col := range filteredCols {
		fieldNames = append(
			fieldNames, col.FieldNames()...,
		)
	}

	qs = qs.Select(
		attrutils.InterfaceList(fieldNames)...,
	)

	var selectedFields = make([]attrs.FieldDefinition, 0, len(fieldNames))
	for _, fieldName := range fieldNames {
		var res, err = qs.WalkField(fieldName)
		if err != nil {
			return nil, nil, err
		}

		for _, inf := range res.Fields {
			selectedFields = append(selectedFields, inf.Fields...)
		}
	}

	var err error
	if m.ChangeQuerySet != nil {
		if qs, err = m.ChangeQuerySet(r, qs); err != nil {
			return nil, nil, err
		}
	}

	rowCnt, rowIter, err := qs.IterAll()
	if err != nil {
		return nil, nil, err
	}

	var headers = make([]string, 0, len(filteredCols))
	for _, col := range filteredCols {
		headers = append(headers, col.ExportHeader(r))
	}

	var values = make([][]interface{}, 0, rowCnt)
	for row, err := range rowIter {
		if err != nil {
			return nil, nil, err
		}

		var rowValues = make([]interface{}, 0, len(headers))
		for _, col := range filteredCols {
			value, err := col.ExportValue(r, row.Object)
			if err != nil {
				return nil, nil, err
			}

			rowValues = append(rowValues, value)
		}
		values = append(values, rowValues)
	}

	var fileName = "export"
	if m.GetFileName != nil {
		fileName = m.GetFileName(r)
	}

	return nil, nil, m.Export(w, r, m, qs, &Export{
		Filename: fileName,
		Header:   headers,
		Rows:     values,
	})
}

type Export struct {
	Filename string          `json:"-"`
	Header   []string        `json:"header"`
	Rows     [][]interface{} `json:"rows"`
}

func ExportJSON[T attrs.Definer](w http.ResponseWriter, r *http.Request, m *ListExportMixin[T], qs *queries.QuerySet[T], export *Export) error {
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, export.Filename))
	w.Header().Set("Content-Type", "application/json")

	var enc = json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(export)
}

func ExportCSV[T attrs.Definer](w http.ResponseWriter, r *http.Request, m *ListExportMixin[T], qs *queries.QuerySet[T], export *Export) error {
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, export.Filename))
	w.Header().Set("Content-Type", "text/csv")

	var writer = csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write(export.Header); err != nil {
		return err
	}

	// Write rows
	for _, row := range export.Rows {
		var record []string
		for _, value := range row {
			record = append(record, fmt.Sprintf("%v", value))
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func ExportText[T attrs.Definer](w http.ResponseWriter, r *http.Request, m *ListExportMixin[T], qs *queries.QuerySet[T], export *Export) error {
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.txt"`, export.Filename))
	w.Header().Set("Content-Type", "text/plain")

	var writer = tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	defer writer.Flush()

	// Write header
	for _, field := range export.Header {
		if _, err := writer.Write([]byte(field + "\t")); err != nil {
			return err
		}
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return err
	}

	// Write rows
	for _, row := range export.Rows {
		var record []string
		for _, value := range row {
			record = append(record, fmt.Sprintf("%v", value))
		}
		if _, err := writer.Write([]byte(strings.Join(record, "\t") + "\n")); err != nil {
			return err
		}
	}

	return nil
}
