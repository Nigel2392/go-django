package todos

import (
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type Todo struct {
	models.Model `table:"todos"`
	ID           int
	Title        string
	Description  string
	Done         bool
}

func (m *Todo) FieldDefs() attrs.Definitions {
	return m.Model.Define(m, func(attrs.Definer) []attrs.Field {
		return []attrs.Field{
			attrs.NewField(m, "ID", &attrs.FieldConfig{
				Primary:  true, // this field is the primary key
				ReadOnly: true, // this field is read-only
				Label:    "ID",
				HelpText: trans.S("The unique identifier of the model"),
			}),
			attrs.NewField(m, "Title", &attrs.FieldConfig{
				Label:    "Title",
				HelpText: trans.S("The title of the todo"),
			}),
			attrs.NewField(m, "Description", &attrs.FieldConfig{
				Label:    "Description",
				HelpText: trans.S("A description of the todo"),

				// register a custom widget for this field
				// this will render a textarea instead of a text input
				FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
					return widgets.NewTextarea(nil)
				},
			}),
			attrs.NewField(m, "Done", &attrs.FieldConfig{
				Label:    "Done",
				HelpText: trans.S("Indicates whether the todo is done or not"),
				Blank:    true,
			}),
		}
	})

}
