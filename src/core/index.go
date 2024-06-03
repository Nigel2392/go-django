package core

import (
	"fmt"
	"net/http"
	"net/mail"

	"github.com/Nigel2392/django/contrib/blocks"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/mux/middleware/sessions"
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

var validFormData map[string]interface{}

func Index(w http.ResponseWriter, r *http.Request) {
	var session = sessions.Retrieve(r)
	fmt.Println(session.Get("page_key"))
	session.Set("page_key", "Last visited the index page")

	var form = forms.Initialize(
		forms.NewBaseForm(),
		forms.WithRequestData("POST", r),
		forms.WithFields(
			fields.EmailField(
				fields.Label("Email"),
				fields.HelpText("Enter your email"),
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
				fields.HelpText("Enter your name"),
				fields.Name("name"),
				fields.Required(true),
				fields.Regex(`^[a-zA-Z]+$`),
				fields.MinLength(2),
				fields.MaxLength(50),
			),
			fields.CharField(
				fields.Label("Password"),
				fields.HelpText("Enter your password"),
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
				fields.HelpText("Enter your age"),
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
				var b = blocks.NewStructBlock()

				b.AddField("name", blocks.CharBlock())
				b.AddField("age", blocks.NumberBlock())
				b.AddField("email", blocks.EmailBlock())
				b.AddField("password", blocks.PasswordBlock())
				b.AddField("date", blocks.DateBlock())
				b.AddField("datetime", blocks.DateTimeBlock())

				var lb = blocks.NewListBlock(blocks.TextBlock(
					blocks.WithValidators[*blocks.FieldBlock](func(i interface{}) error {
						fmt.Println("Validating", i)
						if i == nil || i == "" {
							return errs.ErrFieldRequired
						}
						return nil
					}),
					blocks.WithLabel[*blocks.FieldBlock]("Data Sub-Block"),
				), 3, 5)

				lb.LabelFunc = func() string {
					return "Data List"
				}

				b.AddField("data", lb)

				// var c = blocks.NewMultiBlock()
				// c.AddField("name", blocks.CharBlock())
				// c.AddField("age", blocks.NumberBlock())
				// c.AddField("email", blocks.EmailBlock())
				//
				// b.AddField("data2", c)

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

	if validFormData != nil {
		form.Initial = validFormData
	}

	if r.Method == "POST" && form.IsValid() {
		validFormData = form.CleanedData()
		var s = &FormData{}

		attrs.Set(s, "Email", validFormData["email"])
		attrs.Set(s, "Name", validFormData["name"])
		attrs.Set(s, "Password", validFormData["password"])
		attrs.Set(s, "Age", validFormData["age"])
		attrs.Set(s, "Data", validFormData["data"])
		attrs.Set(s, "Block", validFormData["block"])

		fmt.Printf("%+v\n", s)
	}

	var context = core.Context(r)
	context.Set("Form", form)

	if err := tpl.FRender(w, context, "core", "core/index.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
