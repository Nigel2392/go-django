package chooser

import (
	"context"
	"net/http"
	"reflect"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var _ views.View = (*ChooserListPage[attrs.Definer])(nil)

var (
	_ views.View         = (*ChooserListPage[attrs.Definer])(nil)
	_ views.MethodsView  = (*ChooserListPage[attrs.Definer])(nil)
	_ views.BindableView = (*ChooserListPage[attrs.Definer])(nil)
	_ views.Renderer     = (*BoundChooserListPage[attrs.Definer])(nil)
)

type ChooserListPage[T attrs.Definer] struct {
	Template       string
	AllowedMethods []string

	// Fields to include for the model in the view
	Fields []string

	// Labels for the fields in the view
	//
	// This is a map of field name to a function that returns the label for the field.
	//
	// Allowing for custom labels for fields in the view.
	Labels map[string]func(ctx context.Context) string

	// PerPage is the number of items to show per page.
	//
	// This is used for pagination in the list view.
	PerPage uint64

	// Columns are used to define the columns in the list view.
	//
	// This allows for custom rendering logic of the columns in the list view.
	Columns map[string]list.ListColumn[attrs.Definer]

	// Format is a map of field name to a function that formats the field value.
	//
	// I.E. map[string]func(v any) any{"Name": func(v any) any { return strings.ToUpper(v.(string)) }}
	// would uppercase the value of the "Name" field in the list view.
	Format map[string]func(v any) any

	// GetQuerySet is a function that returns a queries.QuerySet to use for the list view.
	GetQuerySet func(r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) *queries.QuerySet[T]

	_Definition *ChooserDefinition[T]
}

func (v *ChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *ChooserListPage[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *ChooserListPage[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var base = &BoundChooserListPage[T]{
		View:           v,
		ResponseWriter: w,
		Request:        req,
		Model: attrs.NewObject[T](
			reflect.TypeOf(v._Definition.Model),
		),
	}
	return base, nil
}

func (v *ChooserListPage[T]) GetContext(req *http.Request, bound *BoundChooserListPage[T]) *ModalContext {
	var c = v._Definition.GetContext(req, v, bound)

	return c
}

type BoundChooserListPage[T attrs.Definer] struct {
	View           *ChooserListPage[T]
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	QuerySet       *queries.QuerySet[T]
	Model          T
}

func (v *BoundChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserListPage[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var c = v.View.GetContext(req, v)

	// Add the queryset to the context
	c.Set("queryset", v.QuerySet)

	return c, nil
}

func (v *BoundChooserListPage[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	return nil
}
