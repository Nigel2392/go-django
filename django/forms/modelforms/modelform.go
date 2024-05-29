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
	Model attrs.Definer
}

func NewBaseModelForm(model attrs.Definer) *BaseModelForm {
	return &BaseModelForm{
		BaseForm: forms.NewBaseForm(),
		Model:    model,
	}
}

func (f *BaseModelForm) Save() error {
	for name, value := range f.CleanedData() {
		attrs.Set(f.Model, name, value)
	}

	if instance, ok := f.Model.(models.Saver); ok {
		return instance.Save()
	}

	return nil
}
