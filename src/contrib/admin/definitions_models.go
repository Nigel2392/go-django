package admin

import (
	"context"
	"net/http"
	"reflect"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

type Prefetch struct {
	SelectRelated   []string
	PrefetchRelated []any
}

func (p *Prefetch) Merge(other Prefetch) {
	var (
		thisSelectMap   = make(map[string]struct{})
		thisPrefetchMap = make(map[any]struct{})
	)

	for _, s := range p.SelectRelated {
		thisSelectMap[s] = struct{}{}
	}
	for _, s := range p.PrefetchRelated {
		thisPrefetchMap[s] = struct{}{}
	}

	var selectRelated = make([]string, len(p.SelectRelated), len(thisSelectMap)+len(other.SelectRelated))
	var prefetchRelated = make([]any, len(p.PrefetchRelated), len(thisPrefetchMap)+len(other.PrefetchRelated))
	copy(selectRelated, p.SelectRelated)
	copy(prefetchRelated, p.PrefetchRelated)

	for _, s := range other.SelectRelated {
		if _, ok := thisSelectMap[s]; !ok {
			selectRelated = append(selectRelated, s)
		}
	}

	for _, s := range other.PrefetchRelated {
		if _, ok := thisPrefetchMap[s]; !ok {
			prefetchRelated = append(prefetchRelated, s)
		}
	}

	p.SelectRelated = selectRelated
	p.PrefetchRelated = prefetchRelated
}

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
	Labels map[string]func(ctx context.Context) string
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

	// GetQuerySet is a function that returns a queries.QuerySet to use for the list view.
	GetQuerySet func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) *queries.QuerySet[attrs.Definer]

	// Prefetch is used to define the prefetching options for the list view.
	Prefetch Prefetch
}

type DeleteViewOptions struct {
	// FormatMessage func(instance attrs.Definer) string

	// GetHandler is a function that returns a views.View for the model.
	//
	// This allows you to have full control over the view used for deleting the model.
	//
	// This does mean that any logic provided by the admin when deleting the model should be implemented by the developer.
	GetHandler func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) views.View

	// DeleteInstance is a function that deletes the instance.
	//
	// This allows for custom logic to be executed when deleting the instance.
	DeleteInstance func(context.Context, attrs.Definer) error
}

func viewDefaults(o *ViewOptions, mdl any, check func(attrs.FieldDefinition, attrs.Definer) bool) {
	if len(o.Fields) > 0 && len(o.Exclude) > 0 {
		assert.Fail("Fields and Exclude cannot be used together")
	}
	if len(o.Fields) == 0 {
		var exclMap = make(map[string]struct{}, len(o.Exclude))
		for _, field := range o.Exclude {
			exclMap[field] = struct{}{}
		}

		var meta = attrs.GetModelMeta(mdl)
		var defs = meta.Definitions()
		var fields = defs.Fields()
		o.Fields = make([]string, 0, len(fields))
		for _, field := range fields {
			if _, ok := exclMap[field.Name()]; ok {
				continue
			}

			if check != nil && !check(field, mdl.(attrs.Definer)) {
				continue
			}

			o.Fields = append(
				o.Fields,
				field.Name(),
			)
		}
	}
}

func panelDefaults(o *FormViewOptions, mdl attrs.Definer, nameOfMethod string) {
	var method func() []Panel
	var ok bool

	if len(o.Fields) > 0 && len(o.Exclude) > 0 {
		assert.Fail("Fields and Exclude cannot be used together")
	}

	if len(o.Panels) > 0 {
		goto checkFields
	}

	method, ok = attrs.Method[func() []Panel](mdl, nameOfMethod)
	if !ok {
		return
	}

	o.Panels = method()

checkFields:
	if len(o.Fields) > 0 {
		return
	}

	//	o.Fields = make([]string, 0, len(o.Panels))
	//	var exclMap = make(map[string]struct{}, len(o.Exclude))
	//	for _, field := range o.Exclude {
	//		exclMap[field] = struct{}{}
	//	}

	//	for _, panel := range o.Panels {
	//		for _, field := range panel.Fields() {
	//			if _, ok := exclMap[field]; ok {
	//				continue
	//			}
	//
	//			o.Fields = append(o.Fields, field)
	//		}
	//	}
}

type ModelOptions struct {
	// Name is the name of the model.
	//
	// This is used for the model's name in the admin.
	Name string

	// MenuIcon is a function that returns the icon for the model in the admin menu.
	//
	// This should return an HTML element, I.E. "<svg>...</svg>".
	MenuIcon func(ctx context.Context) string

	// MenuOrder is the order of the model in the admin menu.
	MenuOrder int

	// MenuLabel is the label for the model in the admin menu.
	//
	// This is used for the model's label in the admin.
	MenuLabel func(ctx context.Context) string

	// DisallowCreate is a flag that determines if the model should be disallowed from being created.
	//
	// This is used to prevent the model from being created in the admin.
	DisallowCreate bool

	// DisallowEdit is a flag that determines if the model should be disallowed from being edited.
	//
	// This is used to prevent the model from being edited in the admin.
	DisallowEdit bool

	// DisallowDelete is a flag that determines if the model should be disallowed from being deleted.
	//
	// This is used to prevent the model from being deleted in the admin.
	DisallowDelete bool

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
	RegisterToAdminMenu any

	// Labels for the fields in the model.
	//
	// This provides a simple top- level override for the labels of the fields in the model.
	//
	// Any custom labels for fields implemented in the AddView, EditView or ListView will take precedence over these labels.
	Labels map[string]func(ctx context.Context) string

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

func (o *ModelDefinition) EditContext(key string, context ctx.Context) {
	var rq = context.(ctx.ContextWithRequest)
	context.Set(key, &WrappedModelDefinition{
		Wrapped: o,
		Context: rq.Request().Context(),
	})
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
	return attrs.NewObject[attrs.Definer](o.rModel())
}

func (o *ModelDefinition) GetName() string {
	if o.ModelOptions.Name == "" {
		return o._cType.Name()
	}
	return o.ModelOptions.Name
}

func (o *ModelDefinition) Label(ctx context.Context) string {
	return o._cType.Label(ctx)
}

func (o *ModelDefinition) PluralLabel(ctx context.Context) string {
	return o._cType.PluralLabel(ctx)
}

func (o *ModelDefinition) getMenuLabel(ctx context.Context) string {
	if o.ModelOptions.MenuLabel != nil {
		return o.ModelOptions.MenuLabel(ctx)
	}
	return o.Label(ctx)
}

func (o *ModelDefinition) GetColumn(ctx context.Context, opts ListViewOptions, field string) list.ListColumn[attrs.Definer] {
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

func (o *ModelDefinition) GetLabel(opts ViewOptions, field string, default_ string) func(ctx context.Context) string {
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
	return func(ctx context.Context) string {
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

	return func(_ *http.Request, defs attrs.Definitions, _ attrs.Definer) interface{} {
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

func (m *ModelDefinition) GetInstance(ctx context.Context, identifier any) (attrs.Definer, error) {
	var instance, err = m._cType.Instance(ctx, identifier)
	if err != nil {
		return nil, err
	}
	return instance.(attrs.Definer), nil
}

func (m *ModelDefinition) GetListInstances(ctx context.Context, amount, offset uint) ([]attrs.Definer, error) {
	var instances, err = m._cType.Instances(ctx, amount, offset)
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
	panelDefaults(&m.AddView, m.Model, "GetAddPanels")
	panelDefaults(&m.EditView, m.Model, "GetEditPanels")
	viewDefaults(&m.ListView.ViewOptions, m.Model, func(fd attrs.FieldDefinition, d attrs.Definer) bool {
		var rel = fd.Rel()
		if rel == nil {
			return true
		}
		var relType = rel.Type()
		var relThrough = rel.Through()
		switch {
		case relType == attrs.RelManyToOne,
			relType == attrs.RelOneToOne && relThrough == nil:
			return true
		default:
			return false
		}
	})

}

func (m *ModelDefinition) Check(ctx context.Context, app *AppDefinition) []checks.Message {
	var messages = make([]checks.Message, 0, 2)

	if !m.ModelOptions.DisallowCreate && m.ModelOptions.AddView.GetHandler == nil && len(m.ModelOptions.AddView.Fields) == 0 && len(m.ModelOptions.AddView.Exclude) == 0 && len(m.ModelOptions.AddView.Panels) == 0 {
		messages = append(messages, checks.Warning(
			"admin.models.add_view.panels",
			"Model has no fields or panels defined for AddView",
			m.GetName(),
		))
	}

	if !m.ModelOptions.DisallowEdit && m.ModelOptions.EditView.GetHandler == nil && len(m.ModelOptions.EditView.Fields) == 0 && len(m.ModelOptions.EditView.Exclude) == 0 && len(m.ModelOptions.EditView.Panels) == 0 {
		messages = append(messages, checks.Warning(
			"admin.models.edit_view.panels",
			"Model has no fields or panels defined for EditView",
			m.GetName(),
		))
	}

	// If the model has a custom handler for the list view, we don't check the columns.
	if m.ModelOptions.ListView.GetHandler == nil && len(m.ModelOptions.ListView.Columns) == 0 && len(m.ModelOptions.ListView.Fields) == 0 && len(m.ModelOptions.ListView.Exclude) == 0 {
		messages = append(messages, checks.Warning(
			"admin.models.list_view.panels",
			"Model has no fields or panels defined for ListView",
			m.GetName(),
		))
	}

	return messages
}

type WrappedModelDefinition struct {
	Wrapped *ModelDefinition
	Context context.Context
}

func (o *WrappedModelDefinition) GetName() string {
	return o.Wrapped.GetName()
}

func (o *WrappedModelDefinition) NewInstance() attrs.Definer {
	return o.Wrapped.NewInstance()
}

func (o *WrappedModelDefinition) Label() string {
	return o.Wrapped.Label(o.Context)
}

func (o *WrappedModelDefinition) PluralLabel() string {
	return o.Wrapped.PluralLabel(o.Context)
}

func (o *WrappedModelDefinition) DisallowCreate() bool {
	return o.Wrapped.ModelOptions.DisallowCreate
}

func (o *WrappedModelDefinition) DisallowEdit() bool {
	return o.Wrapped.ModelOptions.DisallowEdit
}

func (o *WrappedModelDefinition) DisallowDelete() bool {
	return o.Wrapped.ModelOptions.DisallowDelete
}
