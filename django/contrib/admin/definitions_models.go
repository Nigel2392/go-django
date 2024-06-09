package admin

import (
	"net/http"
	"reflect"
	"sync"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/except"
)

type ModelDefinition struct {
	ModelOptions
	Name     string
	_rModel  reflect.Type
	_include map[string]struct{}
}

func (o *ModelDefinition) rModel() reflect.Type {
	if o._rModel == nil {
		o._rModel = reflect.TypeOf(o.Model)
	}
	return o._rModel
}

var globalMu *sync.Mutex = new(sync.Mutex)

func (o *ModelDefinition) include() map[string]struct{} {
	if o._include == nil {
		globalMu.Lock()
		defer globalMu.Unlock()

		var include = len(o.Fields) == 0
		var exclude = len(o.Exclude) > 0
		if include && !exclude {
			return nil
		}

		var excludeMap = make(map[string]struct{})
		for _, name := range o.Exclude {
			excludeMap[name] = struct{}{}
		}

		if exclude && !include {
			for _, name := range o.Exclude {
				assert.True(name != "", "Exclude field cannot be empty")
			}

			var (
				n      = o.NewInstance()
				defs   = n.FieldDefs()
				fields = defs.Fields()
			)

			o.Fields = make([]string, 0, len(fields))
			o._include = make(map[string]struct{})

			for _, field := range fields {
				var name = field.Name()
				if _, ok := excludeMap[name]; !ok {
					o.Fields = append(o.Fields, name)
					o._include[name] = struct{}{}
				}
			}

			return o._include
		}

		assert.False(
			include && exclude,
			"Cannot have both include and exclude fields",
		)

		o._include = make(map[string]struct{})
		for _, name := range o.Fields {
			o._include[name] = struct{}{}
		}
	}
	return o._include
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

func (o *ModelDefinition) GetLabel(field string, default_ string) func() string {
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
	if o.Format == nil {
		return field
	}

	var format, ok = o.Format[field]
	if !ok {
		return field
	}

	return func(defs attrs.Definitions, row attrs.Definer) interface{} {
		var value = defs.Get(field)
		return format(value)
	}
}

func (m *ModelDefinition) ModelFields(instance attrs.Definer) []attrs.Field {
	var defs = instance.FieldDefs()
	if len(m.Fields) == 0 {
		return defs.Fields()
	}

	var (
		fields  = make([]attrs.Field, len(m.Fields))
		include = m.include()
		ok      bool
	)

	for i, name := range m.Fields {
		if _, ok = include[name]; !ok {
			continue
		}

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

	return m.GetList(amount, offset, m.Fields)
}

func (m *ModelDefinition) OnRegister(a *AdminApplication, app *AppDefinition) {

}
