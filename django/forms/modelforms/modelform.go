package modelforms

import (
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/models"
)

type ModelForm interface {
	forms.Form
	models.Saver
}

type BaseModelForm struct {
	*forms.BaseForm
	Model  attrs.Definer
	Fields []attrs.Field
}

func NewBaseModelForm(model attrs.Definer) *BaseModelForm {
	var f = &BaseModelForm{
		BaseForm: forms.NewBaseForm(),
		Model:    model,
	}

	f.Fields = model.FieldDefs().Fields()
	for _, def := range f.Fields {
		f.BaseForm.FormFields.Set(
			def.Name(), def.FormField(),
		)
	}

	return f
}

func (f *BaseModelForm) Load() {
	var initialData = make(map[string]interface{})
	for _, def := range f.Fields {
		initialData[def.Name()] = def.GetDefault()
	}

	f.Initial = initialData
}

func (f *BaseModelForm) Save() error {
	var cleaned = f.CleanedData()

	for _, value := range f.Fields {
		var n = value.Name()
		attrs.Set(f.Model, n, cleaned[n])
	}

	if instance, ok := f.Model.(models.Saver); ok {
		return instance.Save()
	}

	return nil
}
