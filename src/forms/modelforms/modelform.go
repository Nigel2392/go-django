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

var _ forms.Form = (*BaseModelForm[attrs.Definer])(nil)
var _ ModelForm[attrs.Definer] = (*BaseModelForm[attrs.Definer])(nil)

type ModelForm[T any] interface {
	forms.Form
	Load()
	Save() (map[string]interface{}, error)
	WithContext(ctx context.Context)
	Context() context.Context
	SetFields(fields ...string)
	SetExclude(exclude ...string)
	SetOnLoad(fn func(model T, initialData map[string]interface{}))
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
	OnLoad         func(model T, initialData map[string]interface{})

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
		context:  ctx,
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

func (w *BaseModelForm[T]) SetOnLoad(fn func(model T, initialData map[string]interface{})) {
	w.OnLoad = fn
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

	var excluded = make(map[string]struct{})
	for _, n := range f.ModelExclude {
		excluded[n] = struct{}{}
	}

	for _, field := range f.InstanceFields {
		var n = field.Name()
		if _, ok := excluded[n]; ok && f.wasSet(excludeWasSet) {
			continue
		}

		var formField = field.FormField()
		if formField == nil {
			continue
		}

		f.ModelFields = append(f.ModelFields, n)
	}

	var initial = make(map[string]interface{})
	for _, def := range f.InstanceFields {
		var formField = def.FormField()
		if formField == nil {
			continue
		}

		var v = def.GetValue()
		var n = def.Name()
		if attrs.IsZero(v) {
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
	for _, fieldName := range fields {
		var _, assertFailed = fieldMap[fieldName]
		assert.False(assertFailed, "Field %q specified multiple times", fieldName)

		var field, ok = f.Definition.Field(fieldName)
		if !ok {
			fieldMap[fieldName] = struct{}{}
			continue
		}

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
		if !ok {
			continue
		}

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
	if f.wasSet(formLoaded) {
		return
	}

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

	var excluded = make(map[string]struct{})
	for _, n := range f.ModelExclude {
		excluded[n] = struct{}{}
	}

	for _, name := range f.ModelFields {

		if _, ok := excluded[name]; ok && f.wasSet(excludeWasSet) {
			continue
		}

		var formField fields.Field
		var field, ok = f.Definition.Field(name)
		if !ok {
			formField, ok = f.BaseForm.Field(name)
			assert.True(ok, "Field %q not found in %T", name, model)
		} else {
			formField = field.FormField()
		}

		if formField == nil {
			continue
		}

		_, ok = f.Field(name)
		if !ok {
			f.AddField(name, formField)
		}
	}

	var initialData = make(map[string]interface{})
	var fieldDefs = model.FieldDefs()

	if !f.modelIsNil(model) {
		for _, def := range f.InstanceFields {
			var n = def.Name()
			if _, ok := excluded[n]; ok && f.wasSet(excludeWasSet) {
				continue
			}

			var field, ok = fieldDefs.Field(n)
			assert.True(ok, "Field %q not found in %T", n, model)

			initialData[n] = field.GetValue()
		}
	} else {
		for _, def := range f.Definition.Fields() {
			var n = def.Name()
			if _, ok := excluded[n]; ok && f.wasSet(excludeWasSet) {
				continue
			}

			var formField = def.FormField()
			if formField == nil {
				continue
			}

			var field, ok = fieldDefs.Field(n)
			assert.True(ok, "Field %q not found in %T", n, model)

			initialData[n] = field.GetDefault()
		}
	}

	if f.OnLoad != nil {
		f.OnLoad(model, initialData)
	}

	f.SetInitial(initialData)
	f.setFlag(formLoaded, true)
}

func (f *BaseModelForm[T]) WithContext(ctx context.Context) {
	f.context = ctx
	f.BaseForm.FormContext = ctx
}

func (f *BaseModelForm[T]) Context() context.Context {
	if f.context == nil {
		return context.Background()
	}
	return f.context
}

func (f *BaseModelForm[T]) IsValid() bool {
	var cleaned, err = f.BaseForm.Save()
	if err != nil {
		f.AddFormError(err)
		return false
	}

	for _, fieldname := range f.ModelFields {
		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, fieldname) {
			continue
		}

		var field, ok = f.Definition.Field(fieldname)
		assert.True(ok, "Field %q not found in %T", fieldname, f.Model)

		if !field.AllowEdit() {
			continue
		}

		value, ok := cleaned[fieldname]
		if !ok {
			continue
		}

		formField, ok := f.Field(fieldname)
		if !ok {
			continue
		}

		if _, ok := formField.(ModelFieldSaver); ok {
			continue
		}

		if err := field.SetValue(value, true); err != nil {
			f.AddError(
				fieldname,
				err,
			)
		}
	}

	return len(f.ErrorList_) == 0 && (f.Errors == nil || f.Errors.Len() == 0)
}

func (f *BaseModelForm[T]) Save() (map[string]interface{}, error) {
	var cleaned = f.CleanedData()
	var ctx = f.Context()
	var err error
	for _, fieldname := range f.ModelFields {
		if f.wasSet(excludeWasSet) && slices.Contains(f.ModelExclude, fieldname) {
			continue
		}

		var field, ok = f.Definition.Field(fieldname)
		assert.True(ok, "Field %q not found in %T", fieldname, f.Model)

		if !field.AllowEdit() {
			continue
		}

		value, ok := cleaned[fieldname]
		if !ok {
			continue
		}

		formField, ok := f.Field(fieldname)
		if !ok {
			continue
		}

		if saver, ok := formField.(ModelFieldSaver); ok {
			if err := saver.SaveField(ctx, field, value); err != nil {
				f.AddError(
					fieldname,
					err,
				)
				return cleaned, err
			}
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
