package todos

import (
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type Todo struct {
	models.Model
	ID          int
	Title       string
	Description string
	Done        bool
}

func (m *Todo) FieldDefs() attrs.Definitions {
	return m.Model.Define(m, func(attrs.Definer) []attrs.Field {
		return []attrs.Field{
			attrs.NewField(m, "ID", &attrs.FieldConfig{
				Primary:  true,
				ReadOnly: true,
				Label:    "ID",
				HelpText: "The unique identifier of the model",
			}),
			attrs.NewField(m, "Title", &attrs.FieldConfig{
				Label:    "Title",
				HelpText: "The title of the todo",
			}),
			attrs.NewField(m, "Description", &attrs.FieldConfig{
				Label:    "Description",
				HelpText: "A description of the todo",
				FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
					return widgets.NewTextarea(nil)
				},
			}),
			attrs.NewField(m, "Done", &attrs.FieldConfig{
				Label:    "Done",
				HelpText: "Indicates whether the todo is done or not",
			}),
		}
	})

}
