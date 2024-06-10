package admin

import (
	"net/http"
	"reflect"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/except"
)

type ViewOptions struct {
	Fields  []string
	Exclude []string
	Labels  map[string]func() string
}

type ListViewOptions struct {
	ViewOptions
	PerPage uint64
	Format  map[string]func(v any) any
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
	Name                string
	AddView             ViewOptions
	EditView            ViewOptions
	ListView            ListViewOptions
	RegisterToAdminMenu bool
	Labels              map[string]func() string
	GetForID            func(identifier any) (attrs.Definer, error)
	GetList             func(amount, offset uint, include []string) ([]attrs.Definer, error)
	Model               attrs.Definer
}

func (o *ModelOptions) GetName() string {
	if o.Name == "" {
		var rTyp = reflect.TypeOf(o.Model)
		if rTyp.Kind() == reflect.Ptr {
			return rTyp.Elem().Name()
		}
		return rTyp.Name()
	}
	return o.Name
}

type ModelDefinition struct {
	ModelOptions
	Name    string
	LabelFn func() string
	_rModel reflect.Type
}

func (o *ModelDefinition) rModel() reflect.Type {
	if o._rModel == nil {
		o._rModel = reflect.TypeOf(o.Model)
	}
	return o._rModel
}

func (o *ModelDefinition) NewInstance() attrs.Definer {
	var rTyp = o.rModel()
	if rTyp.Kind() == reflect.Ptr {
		return reflect.New(rTyp.Elem()).Interface().(attrs.Definer)
	}
	return reflect.New(rTyp).Interface().(attrs.Definer)
}

func (o *ModelDefinition) GetName() string {
	if o.Name == "" {
		var rTyp = o.rModel()
		if rTyp.Kind() == reflect.Ptr {
			return rTyp.Elem().Name()
		}
		return rTyp.Name()
	}
	return o.Name
}

func (o *ModelDefinition) Label() string {
	if o.LabelFn != nil {
		return o.LabelFn()
	}
	return o.GetName()
}

func (o *ModelDefinition) GetLabel(opts ViewOptions, field string, default_ string) func() string {
	if o.Labels != nil {
		var label, ok = o.Labels[field]
		if ok {
			return label
		}
	}
	if opts.Labels != nil {
		var label, ok = opts.Labels[field]
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
		assert.True(ok, "Field %s not found in model %s", name, m.Name)
	}

	return fields
}

func (m *ModelDefinition) GetInstance(identifier any) (attrs.Definer, error) {
	except.Assert(
		m.GetForID, http.StatusInternalServerError,
		"GetForID not implemented for model %s", m.GetName(),
	)

	return m.GetForID(identifier)
}

func (m *ModelDefinition) GetListInstances(amount, offset uint) ([]attrs.Definer, error) {
	except.Assert(
		m.GetList, http.StatusInternalServerError,
		"GetList not implemented for model %s", m.GetName(),
	)

	return m.GetList(amount, offset, m.ListView.Fields)
}

func (m *ModelDefinition) OnRegister(a *AdminApplication, app *AppDefinition) {
	viewDefaults(&m.AddView, m.Model)
	viewDefaults(&m.EditView, m.Model)
	viewDefaults(&m.ListView.ViewOptions, m.Model)
}
