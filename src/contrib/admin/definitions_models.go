package admin

import (
	"context"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

// Basic options for a model-based view which includes a form.
type ViewOptions struct {
	// Fields to include for the model in the view
	Fields []string

	// Fields to exclude from the model in the view
	Exclude []string

	// Labels for the fields in the view
	//
	// This is a map of field name to a function that returns the label for the field.
	//
	// Allowing for custom labels for fields in the view.
	Labels map[string]func() string
}

// Options for a model-specific form view.
type FormViewOptions struct {
	ViewOptions

	// Widgets are used to define the widgets for the fields in the form.
	//
	// This allows for custom rendering logic of the fields in the form.
	Widgets map[string]widgets.Widget

	// GetForm is a function that returns a modelform.ModelForm for the model.
	GetForm func(req *http.Request, instance attrs.Definer, fields []string) modelforms.ModelForm[attrs.Definer]

	// FormInit is a function that can be used to initialize the form.
	//
	// This may be useful for providing custom form initialization logic.
	FormInit func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer])

	// GetHandler is a function that returns a views.View for the model.
	//
	// This allows you to have full control over the view used for saving the model.
	//
	// This does mean that any logic provided by the admin when saving the model should be implemented by the developer.
	GetHandler func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) views.View

	// A custom function for saving the instance.
	//
	// This function will be called when the form is saved and allows for custom logic to be executed when saving the instance.
	SaveInstance func(context.Context, attrs.Definer) error

	// Panels are used to group fields in the form.
	//
	// This allows for a more simple, maintainable and organized form layout.
	Panels []Panel
}

type ListViewOptions struct {
	ViewOptions

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

	// GetHandler is a function that returns a views.View for the model.
	//
	// This allows you to have full control over the view used for listing the models.
	//
	// This does mean that any logic provided by the admin when listing the models should be implemented by the developer.
	GetHandler func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) views.View
}

type DeleteViewOptions struct {
	// FormatMessage func(instance attrs.Definer) string

	// GetHandler is a function that returns a views.View for the model.
	//
	// This allows you to have full control over the view used for deleting the model.
	//
	// This does mean that any logic provided by the admin when deleting the model should be implemented by the developer.
	GetHandler func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) views.View
}

func viewDefaults(o *ViewOptions, mdl any) {
	if len(o.Fields) > 0 && len(o.Exclude) > 0 {
		assert.Fail("Fields and Exclude cannot be used together")
	}
	if len(o.Fields) == 0 {
		o.Fields = attrs.FieldNames(mdl, o.Exclude)
	}
}

type ModelOptions struct {
	// Name is the name of the model.
	//
	// This is used for the model's name in the admin.
	Name string

	// AddView is the options for the add view of the model.
	//
	// This allows for custom creation logic and formatting form fields / layout.
	AddView FormViewOptions

	// EditView is the options for the edit view of the model.
	//
	// This allows for custom editing logic and formatting form fields / layout.
	EditView FormViewOptions

	// ListView is the options for the list view of the model.
	//
	// This allows for custom listing logic and formatting list columns.
	ListView ListViewOptions

	// DeleteView is the options for the delete view of the model.
	//
	// Any custom logic for deleting the model should be implemented here.
	DeleteView DeleteViewOptions

	// RegisterToAdminMenu is a flag that determines if the model should be automatically registered to the admin menu.
	RegisterToAdminMenu bool

	// Labels for the fields in the model.
	//
	// This provides a simple top- level override for the labels of the fields in the model.
	//
	// Any custom labels for fields implemented in the AddView, EditView or ListView will take precedence over these labels.
	Labels map[string]func() string

	// Model is the object that the above- defined options are for.
	Model attrs.Definer
}

// A struct which is mainly used internally (but can be used by third parties)
// to easily work with models in admin views.
type ModelDefinition struct {
	ModelOptions
	_app    *AppDefinition
	_rModel reflect.Type
	_cType  *contenttypes.ContentTypeDefinition
}

func (o *ModelDefinition) rModel() reflect.Type {
	if o._rModel == nil {
		o._rModel = reflect.TypeOf(o.Model)
	}
	return o._rModel
}

func (o *ModelDefinition) App() *AppDefinition {
	return o._app
}

// Returns a new instance of the model.
//
// This works the same as calling `reflect.New` on the model type.
func (o *ModelDefinition) NewInstance() attrs.Definer {
	var rTyp = o.rModel()
	if rTyp.Kind() == reflect.Ptr {
		return reflect.New(rTyp.Elem()).Interface().(attrs.Definer)
	}
	return reflect.New(rTyp).Interface().(attrs.Definer)
}

func (o *ModelDefinition) GetName() string {
	if o.ModelOptions.Name == "" {
		return o._cType.Name()
	}
	return o.ModelOptions.Name
}

func (o *ModelDefinition) Label() string {
	return o._cType.Label()
}

func (o *ModelDefinition) PluralLabel() string {
	return o._cType.PluralLabel()
}

func (o *ModelDefinition) GetColumn(opts ListViewOptions, field string) list.ListColumn[attrs.Definer] {
	if opts.Columns != nil {
		var col, ok = opts.Columns[field]
		if ok {
			return col
		}
	}
	return list.Column[attrs.Definer](
		o.GetLabel(opts.ViewOptions, field, field),
		o.FormatColumn(field),
	)
}

func (o *ModelDefinition) GetLabel(opts ViewOptions, field string, default_ string) func() string {
	if opts.Labels != nil {
		var label, ok = opts.Labels[field]
		if ok {
			return label
		}
	}
	if o.Labels != nil {
		var label, ok = o.Labels[field]
		if ok {
			return label
		}
	}
	return func() string {
		return default_
	}
}

func (o *ModelDefinition) FormatColumn(field string) any {
	if o.ListView.Format == nil {
		return field
	}

	var format, ok = o.ListView.Format[field]
	if !ok {
		return field
	}

	return func(defs attrs.Definitions, row attrs.Definer) interface{} {
		var value = defs.Get(field)
		return format(value)
	}
}

func (m *ModelDefinition) ModelFields(opts ViewOptions, instace attrs.Definer) []attrs.Field {
	var defs = instace.FieldDefs()
	if len(opts.Fields) == 0 {
		return defs.Fields()
	}

	var (
		fields = make([]attrs.Field, len(opts.Fields))
		ok     bool
	)

	for i, name := range opts.Fields {
		fields[i], ok = defs.Field(name)
		assert.True(ok, "Field %s not found in model %s", name, m.GetName())
	}

	return fields
}

func (m *ModelDefinition) GetInstance(identifier any) (attrs.Definer, error) {
	var instance, err = m._cType.GetInstance(identifier)
	if err != nil {
		return nil, err
	}
	return instance.(attrs.Definer), nil
}

func (m *ModelDefinition) GetListInstances(amount, offset uint) ([]attrs.Definer, error) {
	var instances, err = m._cType.GetInstances(amount, offset)
	if err != nil {
		return nil, err
	}
	var defs = make([]attrs.Definer, len(instances))
	for i, inst := range instances {
		defs[i] = inst.(attrs.Definer)
	}
	return defs, nil
}

func (m *ModelDefinition) OnRegister(a *AdminApplication, app *AppDefinition) {
	viewDefaults(&m.AddView.ViewOptions, m.Model)
	viewDefaults(&m.EditView.ViewOptions, m.Model)
	viewDefaults(&m.ListView.ViewOptions, m.Model)
}
