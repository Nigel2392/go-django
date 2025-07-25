package modelforms

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/models"
)

type ModelForm[T any] interface {
	forms.Form
	Load()
	Save() (map[string]interface{}, error)
	WithContext(ctx context.Context)
	Context() context.Context
	SetFields(fields ...string)
	SetExclude(exclude ...string)
	Instance() T
	SetInstance(model T)
}

type ModelFieldSaver interface {
	SaveField(ctx context.Context, field attrs.Field, value interface{}) error
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
	initialData    map[string]interface{}

	flags modelFormFlag

	Initial      func() T
	SaveInstance func(context.Context, T) error
	ModelFields  []string
	ModelExclude []string
}

func NewBaseModelForm[T attrs.Definer](ctx context.Context, model T, opts ...func(forms.Form)) *BaseModelForm[T] {
	var f = &BaseModelForm[T]{
		BaseForm: forms.NewBaseForm(ctx, opts...),
		Model:    model,
	}

	var (
		rModelType = reflect.TypeOf(model)
		rModel     = reflect.ValueOf(model)
	)

	if f.modelIsNil(model) {
		rModel = reflect.ValueOf(attrs.NewObject[attrs.Definer](rModelType))
		f.Model = rModel.Interface().(T)
	}

	f.SetInstance(f.Model)

	return f
}

// Add InitialData sets the initial data for the form.
//
// This is done on the wrapped [ModelForm] and not on the [BaseForm] itself.
// The [BaseForm] will reset all initial data once the form is loaded with [forms.WithRequestData].
// This means that any initial data would otherwise be lost.
func (f *BaseModelForm[T]) SetInitialData(initial map[string]interface{}) {
	assert.False(
		f.wasSet(formLoaded),
		"Initial data cannot be set after the form fields have been loaded",
	)

	f.initialData = initial
}

func (f *BaseModelForm[T]) InitialData() map[string]interface{} {
	if f.initialData == nil {
		f.initialData = make(map[string]interface{})
	}

	return f.initialData
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

	if f.wasSet(fieldsWasSet) {
		return
	}

	f.Model = model
	f.Definition = model.FieldDefs()
	f.InstanceFields = f.Definition.Fields()

	for _, field := range f.InstanceFields {
		var n = field.Name()
		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, n) {
			continue
		}

		f.ModelFields = append(f.ModelFields, n)
	}

	var initial = make(map[string]interface{})
	for _, def := range f.InstanceFields {
		if !def.AllowEdit() {
			continue
		}
		var v = def.GetValue()
		var n = def.Name()
		if fields.IsZero(v) {
			initial[n] = def.GetDefault()
		} else {
			initial[n] = v
		}
	}

	f.SetInitial(initial)

	f.setFlag(instanceWasSet, true)
}

func (f *BaseModelForm[T]) Instance() T {
	return f.Model
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

func (f *BaseModelForm[T]) Reset() {
	f.initialData = nil
	f.BaseForm.Reset()
	f.setFlag(formLoaded, false)
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

		var formField = field.FormField()
		if formField == nil {
			continue
		}

		f.AddField(name, formField)
	}

	var initialData = make(map[string]interface{})
	var fieldDefs = model.FieldDefs()
	if !f.modelIsNil(model) {
		for _, def := range f.InstanceFields {
			var n = def.Name()
			if !def.AllowEdit() {
				continue
			}

			if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, n) {
				continue
			}

			var field, ok = fieldDefs.Field(n)
			assert.True(ok, "Field %q not found in %T", n, model)

			initialData[n] = field.GetValue()
		}
	} else {
		for _, def := range f.Definition.Fields() {
			var n = def.Name()
			if !def.AllowEdit() {
				continue
			}

			if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, n) {
				continue
			}

			var field, ok = fieldDefs.Field(n)
			assert.True(ok, "Field %q not found in %T", n, model)

			initialData[n] = field.GetDefault()
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

func (f *BaseModelForm[T]) Save() (map[string]interface{}, error) {
	var cleaned, err = f.BaseForm.Save()
	if err != nil {
		return cleaned, err
	}

	var ctx = f.Context()
	for _, fieldname := range f.ModelFields {
		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, fieldname) {
			continue
		}

		var field, ok = f.Definition.Field(fieldname)
		assert.True(ok, "Field %q not found in %T", fieldname, f.Model)

		value, ok := cleaned[fieldname]
		if !ok {
			continue
		}

		formField, ok := f.Field(fieldname)
		if !ok {
			continue
		}

		if saver, ok := formField.(ModelFieldSaver); ok {
			err = saver.SaveField(ctx, field, value)
		} else {
			err = field.SetValue(value, true)
		}

		if err != nil {
			f.AddError(
				fieldname,
				err,
			)
			return cleaned, err
		}
	}

	if f.SaveInstance != nil {
		err = f.SaveInstance(ctx, f.Model)
	} else {
		var saved bool
		saved, err = models.SaveModel(ctx, f.Model)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", f.Model)
		}
		//if instance, ok := any(f.Model).(models.Saver); ok {
		//	err = instance.Save(ctx)
		//}
	}
	if err != nil {
		return cleaned, err
	}

	f.Reset()
	f.Load()

	return cleaned, nil
}
