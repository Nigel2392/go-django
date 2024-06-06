package admin

import (
	"net/http"
	"reflect"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/except"
)

type ModelDefinition struct {
	Name     string
	Fields   []string
	Exclude  []string
	Model    reflect.Type
	GetForID func(identifier any) (attrs.Definer, error)
	GetList  func(amount, offset uint) ([]attrs.Definer, error)
}

func (o *ModelDefinition) GetName() string {
	if o.Name == "" {
		var rTyp = reflect.TypeOf(o.Model)
		if rTyp.Kind() == reflect.Ptr {
			return rTyp.Elem().Name()
		}
		return rTyp.Name()
	}
	return o.Name
}

func (m *ModelDefinition) ModelFields(instance attrs.Definer) []attrs.Field {
	var defs = instance.FieldDefs()
	if len(m.Fields) == 0 {
		return defs.Fields()
	}

	var (
		fields = make([]attrs.Field, len(m.Fields))
		ok     bool
	)

	for i, name := range m.Fields {
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

	return m.GetList(amount, offset)
}

func (m *ModelDefinition) OnRegister(a *AdminApplication, app *AppDefinition) {

}
