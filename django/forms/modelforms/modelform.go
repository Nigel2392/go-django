package modelforms

import (
	"context"
	"reflect"
	"slices"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/models"
)

type ModelForm[T any] interface {
	forms.Form
	Load()
	Save() error
	WithContext(ctx context.Context)
	Context() context.Context
	SetFields(fields ...string)
	SetExclude(exclude ...string)
	SetInstance(model T)
}

type modelFormFlag int

const (
	_ modelFormFlag = iota
	instanceWasSet
	excludeWasSet
	fieldsWasSet
	formLoaded
)

type BaseModelForm[T attrs.Definer] struct {
	*forms.BaseForm
	Model          T
	Definition     attrs.Definitions
	InstanceFields []attrs.Field
	context        context.Context

	flags modelFormFlag

	Initial      func() T
	ModelFields  []string
	ModelExclude []string
}

func NewBaseModelForm[T attrs.Definer](model T) *BaseModelForm[T] {
	var f = &BaseModelForm[T]{
		BaseForm: forms.NewBaseForm(),
		Model:    model,
	}

	var (
		rModelType = reflect.TypeOf(model)
		rModel     = reflect.ValueOf(model)
	)

	if f.modelIsNil(model) {
		if rModelType.Kind() == reflect.Ptr {
			rModel = reflect.New(rModelType.Elem())
		} else {
			rModel = reflect.New(rModelType).Elem()
		}

		f.Model = rModel.Interface().(T)
	}

	f.SetInstance(f.Model)

	return f
}

func (f *BaseModelForm[T]) modelIsNil(model T) bool {
	var rModel = reflect.ValueOf(model)
	var forPtr = rModel.Kind() == reflect.Ptr && (!rModel.IsValid() || rModel.IsNil())
	var forCpy = rModel.Kind() != reflect.Ptr && rModel.IsZero()
	return forPtr || forCpy
}

func (f *BaseModelForm[T]) wasSet(flag modelFormFlag) bool {
	return f.flags&flag != 0
}

func (f *BaseModelForm[T]) setFlag(flag modelFormFlag, b bool) {
	if b {
		f.flags |= flag
	} else {
		f.flags &= ^flag
	}
}

func (f *BaseModelForm[T]) SetInstance(model T) {
	assert.False(
		f.wasSet(formLoaded),
		"Instance has already been set",
	)

	f.Model = model
	f.Definition = model.FieldDefs()
	f.InstanceFields = model.FieldDefs().Fields()

	if f.wasSet(fieldsWasSet) {
		return
	}

	for _, field := range f.InstanceFields {
		var n = field.Name()
		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, n) {
			continue
		}

		f.ModelFields = append(f.ModelFields, n)
	}

	f.setFlag(instanceWasSet, true)
}

func (f *BaseModelForm[T]) SetFields(fields ...string) {
	assert.False(
		f.wasSet(formLoaded),
		"Fields cannot be set after the form fields have been loaded",
	)

	f.ModelFields = make([]string, 0)

	var fieldMap = make(map[string]struct{})
	for _, field := range fields {
		var _, assertFailed = fieldMap[field]
		assert.False(assertFailed, "Field %q specified multiple times", field)

		var field, ok = f.Definition.Field(field)
		assert.True(ok, "Field %q not found in %T", field, f.Model)

		f.ModelFields = append(f.ModelFields, field.Name())
		fieldMap[field.Name()] = struct{}{}
	}

	f.setFlag(fieldsWasSet, true)
}

func (f *BaseModelForm[T]) SetExclude(exclude ...string) {
	assert.False(
		f.wasSet(formLoaded),
		"Exclude cannot be set after the form fields have been loaded",
	)

	f.ModelExclude = make([]string, 0)

	var fieldMap = make(map[string]struct{})
	for _, field := range exclude {
		var _, assertFailed = fieldMap[field]
		assert.False(assertFailed, "Field %q specified multiple times", field)

		var field, ok = f.Definition.Field(field)
		assert.True(ok, "Field %q not found in %T", field, f.Model)

		f.ModelExclude = append(f.ModelExclude, field.Name())
		fieldMap[field.Name()] = struct{}{}
	}

	f.setFlag(excludeWasSet, true)
}

func (f *BaseModelForm[T]) Load() {
	assert.False(
		f.wasSet(formLoaded),
		"Form has already been loaded",
	)

	assert.True(
		f.wasSet(fieldsWasSet) || len(f.ModelFields) > 0,
		"Fields must be set before loading the form",
	)

	assert.True(
		f.wasSet(instanceWasSet) || any(f.Model) != nil,
		"Instance must be set before loading the form",
	)

	var model = f.Model
	if f.Initial != nil {
		model = f.Initial()
	}

	for _, name := range f.ModelFields {

		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, name) {
			continue
		}

		var field, ok = f.Definition.Field(name)
		assert.True(ok, "Field %q not found in %T", name, model)

		f.AddField(name, field.FormField())
	}

	var initialData = make(map[string]interface{})
	if !f.modelIsNil(model) {
		for _, def := range f.InstanceFields {
			var n = def.Name()
			if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, n) {
				continue
			}
			initialData[n] = attrs.Get[any](model, n)
		}
	} else {
		for _, def := range f.Definition.Fields() {
			var n = def.Name()
			if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, n) {
				continue
			}
			initialData[n] = def.GetDefault()
		}
	}

	f.SetInitial(initialData)
	f.setFlag(formLoaded, true)
}

func (f *BaseModelForm[T]) WithContext(ctx context.Context) {
	f.context = ctx
}

func (f *BaseModelForm[T]) Context() context.Context {
	if f.context == nil {
		return context.Background()
	}
	return f.context
}

func (f *BaseModelForm[T]) Save() error {
	var cleaned = f.CleanedData()
	var ctx = f.Context()

	for _, fieldname := range f.ModelFields {
		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, fieldname) {
			continue
		}

		var _, ok = f.Definition.Field(fieldname)
		assert.True(ok, "Field %q not found in %T", fieldname, f.Model)

		value, ok := cleaned[fieldname]
		if !ok {
			continue
		}

		if err := attrs.Set(f.Model, fieldname, value); err != nil {
			return err
		}
	}

	if instance, ok := any(f.Model).(models.Saver); ok {
		return instance.Save(ctx)
	}

	return nil
}
