package views

import (
	"fmt"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	django_views "github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux"
)

type BaseView = django_views.BaseView

type ObjectView[T attrs.Definer] struct {
	BaseView
	ObjectVar   string
	ContextVar  string // optional, used to store the object in the context
	Object      T
	Permissions []string
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
