package core

import (
	"fmt"
	"net/http"
	"net/mail"

	"github.com/Nigel2392/django/blocks"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

type FormData struct {
	Email    *mail.Address
	Name     string
	Password string
	Age      int
	Data     map[string]any
	Block    map[string]any ``
}

func (f *FormData) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f)
}

func Index(w http.ResponseWriter, r *http.Request) {
	var form forms.Form = forms.Initialize(
		forms.NewBaseForm(),
		forms.WithRequestData("POST", r),
		forms.WithFields(
			fields.EmailField(
				fields.Label("Email"),
				fields.Name("email"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(250),
				fields.Widget(
					widgets.NewEmailInput(nil),
				),
			),
			fields.CharField(
				fields.Label("Name"),
				fields.Name("name"),
				fields.Required(true),
				fields.Regex(`^[a-zA-Z]+$`),
				fields.MinLength(2),
				fields.MaxLength(50),
			),
			fields.CharField(
				fields.Label("Password"),
				fields.Name("password"),
				fields.Required(true),
				fields.MinLength(8),
				fields.MaxLength(50),
				fields.Widget(
					widgets.NewPasswordInput(nil),
				),
			),
			fields.NumberField[int](
				fields.Label("Age"),
				fields.Name("age"),
				fields.Required(true),
			),
			fields.JSONField[map[string]any](
				fields.Label("Data"),
				fields.Name("data"),
				fields.Required(true),
			),
			blocks.BlockField(
				blocks.CharBlock(),
				fields.Label("Block"),
				fields.Name("block_data"),
			),
			func() fields.Field {
				var b = blocks.NewMultiBlock()

				b.AddField("name", blocks.CharBlock())
				b.AddField("age", blocks.NumberBlock())
				b.AddField("email", blocks.CharBlock())
				b.AddField("data", blocks.NewListBlock(blocks.TextBlock(), 3, 5))

				var f = blocks.BlockField(
					b,
					fields.Label("Block"),
					fields.Name("block"),
				)

				return f
			}(),
		),
		forms.OnValid(func(f forms.Form) {
			fmt.Println("Form is valid")
			var data = f.CleanedData()
			for k, v := range data {
				fmt.Printf("%T, %s: %v\n", v, k, v)
			}
		}),
		forms.OnInvalid(func(f forms.Form) {
			fmt.Println("Form is invalid:", f.BoundErrors())
		}),
		forms.OnFinalize(func(f forms.Form) {
			fmt.Println("Form is finalized:", f.BoundErrors())
		}),
	)

	if r.Method == "POST" && form.IsValid() {
		var data = form.CleanedData()
		var s = &FormData{}

		attrs.Set(s, "Email", data["email"])
		attrs.Set(s, "Name", data["name"])
		attrs.Set(s, "Password", data["password"])
		attrs.Set(s, "Age", data["age"])
		attrs.Set(s, "Data", data["data"])
		attrs.Set(s, "Block", data["block"])

		fmt.Printf("%+v\n", s)
	}

	var context = http_.Context(r)
	context.Set("Form", form)

	if err := tpl.FRender(w, context, "core", "core/index.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
