package chooser

import (
	"context"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var _ views.View = (*ChooserListPage[attrs.Definer])(nil)

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
	GetQuerySet func(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) *queries.QuerySet[T]
}

func (v *ChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}
