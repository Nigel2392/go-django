package views

import (
	"fmt"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	django_views "github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux"
)

type BaseView = django_views.BaseView

var (
	_ views.ContextGetter = (*ObjectView[attrs.Definer])(nil)
	_ views.Checker       = (*ObjectView[attrs.Definer])(nil)
)

type ObjectView[T attrs.Definer] struct {
	BaseView
	ObjectVar   string
	ContextVar  string // optional, used to store the object in the context
	Object      T
	Permissions []string
}

// Check checks the request before serving it.
// Useful for checking if the request is valid before serving it.
// Like checking permissions, etc...
func (v *ObjectView[T]) Check(w http.ResponseWriter, req *http.Request) error {
	if !permissions.HasPermission(req, v.Permissions...) {
		return errors.PermissionDenied.Wrapf(
			"user does not have permission to access %T", v.Object,
		)
	}
	return nil
}

// Fail is a helper function to fail the request.
// This can be used to redirect, etc.
func (v *ObjectView[T]) Fail(w http.ResponseWriter, req *http.Request, err error) {
	if errors.Is(err, errors.PermissionDenied) {
		except.Fail(
			http.StatusForbidden, err,
		)
	} else {
		except.Fail(
			http.StatusInternalServerError, err,
		)
	}
}

func (v *ObjectView[T]) GetQuerySet(r *http.Request) *queries.QuerySet[T] {
	return queries.GetQuerySet(v.Object).WithContext(r.Context())
}

func (v *ObjectView[T]) GetObject(r *http.Request, pk any) (*queries.Row[T], error) {
	meta := attrs.GetModelMeta(v.Object)
	defs := meta.Definitions()
	primaryField := defs.Primary()
	qs := v.GetQuerySet(r).Filter(primaryField.Name(), pk)
	row, err := qs.Get()
	if err != nil {
		return nil, err
	}

	return row, nil
}

func (v *ObjectView[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var (
		context    = ctx.RequestContext(req)
		vars       = mux.Vars(req)
		objectVar  = v.ObjectVar
		contextVar = v.ContextVar
	)

	if objectVar == "" {
		objectVar = "pk"
	}

	if contextVar == "" {
		contextVar = "object"
	}

	var pk = vars.Get(objectVar)
	if pk == "" {
		return nil, errors.FieldNull.Wrapf(
			"missing required path variable \"%s\" for %T",
			objectVar, v.Object,
		)
	}

	var definer = attrs.NewObject[T](v.Object)
	var defs = definer.FieldDefs()
	var primary = defs.Primary()
	if primary == nil {
		return nil, errors.FieldNull.Wrapf(
			"model %T does not have a primary key field defined",
			v.Object,
		)
	}

	if err := primary.Scan(pk); err != nil {
		return nil, errors.FieldNull.Wrapf(
			"failed to scan primary key field %q of %T: %v",
			primary.Name(), v.Object, err,
		)
	}

	row, err := v.GetObject(req, primary.GetValue())
	if err != nil {
		return nil, err
	}

	context.Set(fmt.Sprintf("%sRow", contextVar), row)
	context.Set(contextVar, row.Object)

	return context, nil
}
