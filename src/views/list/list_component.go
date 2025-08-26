package list

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type List[T attrs.Definer] struct {
	Model        T
	Columns      []ListColumn[T]
	HideHeadings bool
	groups       []ColumnGroup[T]
	request      *http.Request
	formObject   *ListForm[T]
}

func NewList[T attrs.Definer](r *http.Request, modelObj T, list []T, columns ...ListColumn[T]) *List[T] {
	return NewListWithGroups(r, modelObj, list, columns, func(r *http.Request, obj T, cols []ListColumn[T]) ColumnGroup[T] {
		return NewColumnGroup(r, obj, cols)
	})
}

func NewListWithGroups[T attrs.Definer](r *http.Request, modelObj T, list []T, columns []ListColumn[T], newGroup func(r *http.Request, obj T, cols []ListColumn[T]) ColumnGroup[T]) *List[T] {
	var l = &List[T]{
		Model:   modelObj,
		Columns: columns,
		groups:  make([]ColumnGroup[T], 0, len(list)),
		request: r,
	}

	for _, item := range list {
		l.groups = append(
			l.groups,
			newGroup(r, item, columns),
		)
	}

	return l
}

func (l *List[T]) Media() media.Media {
	var m media.Media = media.NewMedia()
	var defs attrs.StaticDefinitions
	if reflect.ValueOf(l.Model).IsValid() {
		var meta = attrs.GetModelMeta(l.Model)
		defs = meta.Definitions()
	}
	for _, col := range l.Columns {
		if mc, ok := col.(media.MediaDefiner); ok {
			m = m.Merge(mc.Media())
		}
		if defs != nil {
			if mc, ok := col.(ListMediaColumn); ok {
				m = m.Merge(mc.Media(defs))
			}
		}
	}
	return m
}

func (l *List[T]) Form(opts ...func(forms.Form)) *ListForm[T] {
	if l.formObject != nil {
		return l.formObject
	}
	var form = NewListForm(l, opts...)
	l.formObject = form
	return form
}

func (l *List[T]) Render() string {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("%v - %s", r, debug.Stack())
			panic(r)
		}
	}()

	var component = l.Component()
	var b strings.Builder
	component.Render(l.request.Context(), &b)
	return b.String()
}

type ListForm[T attrs.Definer] struct {
	List  *List[T]
	forms map[any]forms.Form
}

func NewListForm[T attrs.Definer](l *List[T], opts ...func(forms.Form)) *ListForm[T] {
	var form = &ListForm[T]{
		List:  l,
		forms: make(map[any]forms.Form),
	}

	for _, group := range l.groups {
		var pk = attrs.PrimaryKey(group.Row())
		var rowForm = group.Form(l.request, append(opts, func(f forms.Form) {
			f.SetPrefix(fmt.Sprintf("%srow--%s", f.Prefix(), attrs.ToString(pk)))
		})...)
		if rowForm == nil {
			continue
		}
		form.forms[pk] = rowForm
	}

	if len(form.forms) == 0 {
		return nil
	}

	return form
}

func (lf *ListForm[T]) Forms() map[any]forms.Form {
	return lf.forms
}

func (lf *ListForm[T]) ForInstance(instance T) forms.Form {
	return lf.forms[attrs.PrimaryKey(instance)]
}

func (lf *ListForm[T]) IsValid() bool {
	for _, form := range lf.Forms() {
		if !form.IsValid() {
			return false
		}
	}
	return true
}

func (lf *ListForm[T]) CleanedData() map[any]map[string]any {
	var data = make(map[any]map[string]any)
	for key, form := range lf.Forms() {
		data[key] = form.CleanedData()
	}
	return data
}

func (lf *ListForm[T]) Errors() []error {
	var errs []error
	for _, form := range lf.Forms() {
		errs = append(errs, form.ErrorList()...)
	}
	return errs
}

func (lf *ListForm[T]) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	var errs = orderedmap.NewOrderedMap[string, []error]()
	for _, form := range lf.Forms() {
		var formErrors = form.BoundErrors()
		for head := formErrors.Front(); head != nil; head = head.Next() {
			var errList = errs.GetOrDefault(head.Key, make([]error, 0, len(head.Value)))
			errList = append(errList, head.Value...)
			errs.Set(head.Key, errList)
		}
	}
	return errs
}
