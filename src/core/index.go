package core

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/contrib/blocks"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/mux/middleware/sessions"
)

type MainStruct struct {
	Email    *mail.Address       `attrs:"label=Email;helptext=Enter your email;null;required;min_length=5;max_length=250"`
	Name     string              `attrs:"label=Name;helptext=Enter your name;primary;required;regex=^[a-zA-Z]+$;min_length=2;max_length=50"`
	Password string              `attrs:"label=Password;helptext=Enter your password;required;min_length=8;max_length=50"`
	Age      int                 `attrs:"label=Age;helptext=Enter your age;required"`
	Data     map[string]any      `attrs:"label=Object Data;required;help_text=Enter your data"`
	Block    *blocks.StructBlock `attrs:"label=Block;required;help_text=Enter your block data"`
}

func (m *MainStruct) AdminForm(r *http.Request, app *admin.AppDefinition, model *admin.ModelDefinition) modelforms.ModelForm[attrs.Definer] {
	//widgets.NewCheckboxInput(),
	//widgets.NewRadioInput(),
	//widgets.NewSelectInput(),

	//var form = forms.Initialize(
	//	forms.NewBaseForm(),
	//	forms.WithRequestData("POST", r),
	//	forms.WithFields(
	//		fields.EmailField(
	//			fields.Label("Email"),
	//			fields.HelpText("Enter your email"),
	//			fields.Name("email"),
	//			fields.Required(true),
	//			fields.MinLength(5),
	//			fields.MaxLength(250),
	//		),
	//		fields.CharField(
	//			fields.Label("Name"),
	//			fields.HelpText("Enter your name"),
	//			fields.Name("name"),
	//			fields.Required(true),
	//			fields.Regex(`^[a-zA-Z\s+]+$`),
	//			fields.MinLength(2),
	//			fields.MaxLength(50),
	//		),
	//		auth.NewPasswordField(
	//			fields.Label("Password"),
	//			fields.HelpText("Enter your password"),
	//			fields.Name("password"),
	//			fields.Required(true),
	//			fields.MinLength(8),
	//			fields.MaxLength(50),
	//		),
	//
	//		//fields.CharField(
	//		//	fields.Name("checkbox"),
	//		//	fields.Label("Checkbox"),
	//		//	fields.Required(true),
	//		//	fields.Widget(
	//		//		widgets.NewCheckboxInput(nil, func() []widgets.Option {
	//		//			return []widgets.Option{
	//		//				widgets.NewOption("checkbox1", "Checkbox 1", "1"),
	//		//				widgets.NewOption("checkbox2", "Checkbox 2", "2"),
	//		//				widgets.NewOption("checkbox3", "Checkbox 3", "3"),
	//		//			}
	//		//		}),
	//		//	),
	//		//),
	//		//fields.CharField(
	//		//	fields.Name("checkbox"),
	//		//	fields.Label("Checkbox"),
	//		//	fields.Required(true),
	//		//	fields.Widget(
	//		//		widgets.NewRadioInput(nil, func() []widgets.Option {
	//		//			return []widgets.Option{
	//		//				widgets.NewOption("checkbox1", "Checkbox 1", "1"),
	//		//				widgets.NewOption("checkbox2", "Checkbox 2", "2"),
	//		//				widgets.NewOption("checkbox3", "Checkbox 3", "3"),
	//		//			}
	//		//		}),
	//		//	),
	//		//),
	//		//fields.CharField(
	//		//	fields.Name("checkbox"),
	//		//	fields.Label("Checkbox"),
	//		//	fields.Required(true),
	//		//	fields.Widget(
	//		//		widgets.NewSelectInput(nil, func() []widgets.Option {
	//		//			return []widgets.Option{
	//		//				widgets.NewOption("checkbox1", "Checkbox 1", "1"),
	//		//				widgets.NewOption("checkbox2", "Checkbox 2", "2"),
	//		//				widgets.NewOption("checkbox3", "Checkbox 3", "3"),
	//		//			}
	//		//		}),
	//		//	),
	//		//),
	//	),
	//)
	//
	//return form

	var f modelforms.ModelForm[attrs.Definer] = modelforms.NewBaseModelForm[attrs.Definer](m)
	return forms.Initialize(
		f, forms.WithFields(
			fields.EmailField(
				fields.Label("Email"),
				fields.HelpText("Enter your email"),
				fields.Name("Email"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(250),
			),
			fields.CharField(
				fields.Label("Name"),
				fields.HelpText("Enter your name"),
				fields.Name("Name"),
				fields.Required(true),
				fields.Regex(`^[a-zA-Z\s+]+$`),
				fields.MinLength(2),
				fields.MaxLength(50),
			),
			auth.NewPasswordField(
				fields.Label("Password"),
				fields.HelpText("Enter your password"),
				fields.Name("Password"),
				fields.Required(true),
				fields.MinLength(8),
				fields.MaxLength(50),
			),
		),
	)
}

func (m *MainStruct) GetBlockDef() blocks.Block {
	var b = blocks.NewStructBlock()
	b.LabelFunc = func() string {
		return "Data Block"
	}

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

	lb = blocks.NewListBlock(blocks.TextBlock(
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

	b.AddField("data2", lb)
	// var c = blocks.NewMultiBlock()
	// c.AddField("name", blocks.CharBlock())
	// c.AddField("age", blocks.NumberBlock())
	// c.AddField("email", blocks.EmailBlock())
	//
	// b.AddField("data2", c)

	return b
}

var _ = admin.RegisterApp(
	"core",
	admin.AppOptions{
		RegisterToAdminMenu: true,
		MenuLabel:           fields.S("Core"),
	},
	admin.ModelOptions{
		RegisterToAdminMenu: true,
		Fields:              []string{"Email", "Name", "Password", "Age", "Data", "Block"},
		Labels: map[string]func() string{
			"Email":    fields.S("Object Email"),
			"Name":     fields.S("Object Name"),
			"Password": fields.S("Object Password"),
			"Age":      fields.S("Object Age"),
			"Data":     fields.S("Object Data"),
			"Block":    fields.S("Object Block"),
		},
		Format: map[string]func(interface{}) interface{}{
			"Age": func(v any) interface{} {
				return fmt.Sprintf("%d years old", v)
			},
			"Password": func(v any) interface{} {
				return "********"
			},
			"Data": func(v any) interface{} {
				var data = v.(map[string]any)
				var b strings.Builder
				for k, v := range data {
					b.WriteString(fmt.Sprintf("%s: %v\n", k, v))
				}
				return b.String()
			},
			"Block": func(v any) interface{} {
				return "Block"
			},
		},
		Model: &MainStruct{},
		GetForID: func(identifier any) (attrs.Definer, error) {
			return &MainStruct{
				Email:    &mail.Address{Address: "test@localhost"},
				Name:     "Test User",
				Password: "password",
				Age:      30,
				Data:     map[string]any{"key": "value"},
				Block:    &blocks.StructBlock{},
			}, nil
		},
		GetList: func(amount, offset uint, fields []string) ([]attrs.Definer, error) {
			var listItemCount = 10
			var items = make([]attrs.Definer, listItemCount)
			for i := 0; i < listItemCount; i++ {
				items[i] = &MainStruct{
					Email:    &mail.Address{Address: fmt.Sprintf("user-%d@test.localhost", i)},
					Name:     fmt.Sprintf("User %d", i),
					Password: "password",
					Age:      i + 20,
					Data:     map[string]any{"key": fmt.Sprintf("value-%d", i)},
					Block:    &blocks.StructBlock{},
				}
			}
			return items, nil
		},
	},
)

func (f *MainStruct) FieldDefs() attrs.Definitions {
	if f == nil {
		panic("MainStruct is nil")
	}
	return attrs.AutoDefinitions(f)
}

var validFormData map[string]interface{}

func Index(w http.ResponseWriter, r *http.Request) {
	var session = sessions.Retrieve(r)
	fmt.Println(session.Get("page_key"))
	session.Set("page_key", "Last visited the index page")
	var instance = &MainStruct{}
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
				instance.GetBlockDef(),
				fields.Label("Block"),
				fields.Name("block"),
			),
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
		var s = &MainStruct{}

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
