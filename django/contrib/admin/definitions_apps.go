package admin

import (
	"reflect"

	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
)

type ModelOptions struct {
	Name     string
	Fields   []string
	Exclude  []string
	Labels   map[string]func() string
	GetForID func(identifier any) (attrs.Definer, error)
	GetList  func(amount, offset uint, include []string) ([]attrs.Definer, error)
	Model    attrs.Definer
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

type AppDefinition struct {
	Name   string
	Models *orderedmap.OrderedMap[
		string, *ModelDefinition,
	]
	URLs []core.URL
}

func (a *AppDefinition) Register(opts ModelOptions) *ModelDefinition {

	var rTyp = reflect.TypeOf(opts.Model)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}

	assert.False(
		rTyp.Kind() == reflect.Invalid,
		"Model must be a valid type")

	assert.False(
		opts.GetForID == nil,
		"GetForID must be implemented",
	)

	assert.False(
		opts.GetList == nil,
		"GetList must be implemented",
	)

	assert.True(
		rTyp.Kind() == reflect.Struct,
		"Model must be a struct")

	assert.True(
		rTyp.NumField() > 0,
		"Model must have fields")

	var model = &ModelDefinition{
		Name:         opts.GetName(),
		ModelOptions: opts,
		_rModel:      rTyp,
	}

	assert.True(
		model.Name != "",
		"Model must have a name")

	assert.True(
		nameRegex.MatchString(model.Name),
		"Model name must match regex %v",
		nameRegex,
	)

	a.Models.Set(model.Name, model)

	return model
}

func (a *AppDefinition) OnReady(adminSite *AdminApplication) {
	var models = a.Models.Keys()
	for _, model := range models {
		var modelDef, ok = a.Models.Get(model)
		assert.True(ok, "Model not found")
		modelDef.OnRegister(adminSite, a)
	}
}
