package list

import (
	"context"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/a-h/templ"
)

type listContextKey struct {
	v string
}

var (
	contextKeyPaginator      = &listContextKey{"paginator"}
	contextKeyPage           = &listContextKey{"page"}
	contextKeyAllowEdit      = &listContextKey{"allow_edit"}
	contextKeyAllowRowSelect = &listContextKey{"allow_row_select"}
)

func PaginatorFromContext[T attrs.Definer](ctx context.Context) pagination.Pagination[T] {
	paginator, _ := ctx.Value(contextKeyPaginator).(pagination.Pagination[T])
	return paginator
}

func PageFromContext[T any](ctx context.Context) pagination.PageObject[T] {
	page, _ := ctx.Value(contextKeyPage).(pagination.PageObject[T])
	return page
}

func AllowListEdit(ctx context.Context) bool {
	allowEdit, _ := ctx.Value(contextKeyAllowEdit).(bool)
	return allowEdit
}

func SetAllowListEdit(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, contextKeyAllowEdit, allow)
}

func AllowListRowSelect(ctx context.Context) bool {
	allowRowSelect, _ := ctx.Value(contextKeyAllowRowSelect).(bool)
	return allowRowSelect
}

func SetAllowListRowSelect(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, contextKeyAllowRowSelect, allow)
}

type ListColumn[T attrs.Definer] interface {
	Header(r *http.Request) templ.Component
	FieldName() string
	Component(r *http.Request, defs attrs.Definitions, row T) templ.Component
}

type ListMediaColumn interface {
	Media(defs attrs.StaticDefinitions) media.Media
}

type ListEditableColumn[T attrs.Definer] interface {
	ListColumn[T]
	FormField(r *http.Request, row T) fields.Field
	EditableComponent(r *http.Request, defs attrs.Definitions, row T, form forms.Form, field *forms.BoundFormField) templ.Component
}

type ColumnGroup[T attrs.Definer] interface {
	Row() T
	AddColumn(column ListColumn[T])
	Form(r *http.Request, opts ...func(forms.Form)) forms.Form
	Component(r *http.Request, form *ListForm[T]) templ.Component
}

type listView__QuerySetGetter[T attrs.Definer] interface {
	GetQuerySet(r *http.Request) (*queries.QuerySet[T], error)
}

type listView__QuerySetFilterer[T attrs.Definer] interface {
	FilterQuerySet(r *http.Request, qs *queries.QuerySet[T]) (*queries.QuerySet[T], error)
}

type listView__Paginator[T attrs.Definer] interface {
	GetPaginator(r *http.Request, qs *queries.QuerySet[T]) (pagination.Pagination[T], pagination.PageObject[T], error)
}

type listView__ContextGetter[T attrs.Definer] interface {
	GetContext(r *http.Request, qs *queries.QuerySet[T], context ctx.Context) (ctx.Context, error)
}

type listView__MixinContextGetter[T attrs.Definer] interface {
	GetContext(r *http.Request, view views.View, qs *queries.QuerySet[T], context ctx.Context) (ctx.Context, error)
}

type listView__MixinHijacker[T attrs.Definer] interface {
	Hijack(w http.ResponseWriter, r *http.Request, view views.View, qs *queries.QuerySet[T], context ctx.Context) (http.ResponseWriter, *http.Request, error)
}

type StringRenderer interface {
	Render() string
}

type listView__ColumnGetter[T attrs.Definer] interface {
	GetListColumns(r *http.Request) ([]ListColumn[T], error)
}

type listView__ListGetter[T attrs.Definer] interface {
	GetList(r *http.Request, pageObject pagination.PageObject[T], columns []ListColumn[T], context ctx.Context) (StringRenderer, error)
}
