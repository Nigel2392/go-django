package todos

import (
	"errors"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms/widgets"
)

type myInt int

type MyModel struct {
	ID   int
	Name string
	Bio  string
	Age  myInt
}

func (m *MyModel) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Label:    "ID",
			HelpText: "The unique identifier of the model",
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{
			Label:    "Name",
			HelpText: "The name of the model",
		}),
		attrs.NewField(m, "Bio", &attrs.FieldConfig{
			Label:    "Biography",
			HelpText: "The biography of the model",
			FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
				return widgets.NewTextarea(nil)
			},
		}),
		attrs.NewField(m, "Age", &attrs.FieldConfig{
			Label:    "Age",
			HelpText: "The age of the model",
			Validators: []func(interface{}) error{
				func(v interface{}) error {
					if v.(myInt) <= myInt(0) {
						return errors.New("Age must be greater than 0")
					}
					return nil
				},
			},
		}),
	)
}
