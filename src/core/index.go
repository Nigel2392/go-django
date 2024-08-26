package core

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"slices"
	"strings"

	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/blocks"
	"github.com/Nigel2392/django/contrib/editor"
	_ "github.com/Nigel2392/django/contrib/editor/features"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/filesystem/tpl"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/django/models"
	"github.com/Nigel2392/django/views/list"
	"github.com/Nigel2392/mux/middleware/sessions"
)

var _ (models.Saver) = (*MainStruct)(nil)

var mainStructMap = make(map[string]*MainStruct)

type MainStruct struct {
	Email      *mail.Address       `attrs:"label=Email;primary;helptext=Enter your email;null;required;min_length=5;max_length=250"`
	Name       string              `attrs:"label=Name;helptext=Enter your name;required;regex=^[a-zA-Z]+$;min_length=2;max_length=50"`
	Password   string              `attrs:"label=Password;helptext=Enter your password;required;min_length=8;max_length=50"`
	Age        int                 `attrs:"label=Age;helptext=Enter your age;required"`
	Data       json.RawMessage     `attrs:"label=Object Data;required;help_text=Enter your data"`
	Block      *blocks.StructBlock `attrs:"label=Block;required;help_text=Enter your block data"`
	_blockData map[string]interface{}
	Editor     *editor.EditorJSBlockData `attrs:"label=Editor;required;help_text=Enter your editor data"`
}

func (m *MainStruct) SetBlock(v interface{}) {
	var val, ok = v.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("Invalid block data: %T", v))
	}
	m._blockData = val

}

func (m *MainStruct) GetBlock() interface{} {
	return m._blockData
}

func (m *MainStruct) GetDefaultBlock() interface{} {
	return make(map[string]interface{})
}

func (m *MainStruct) Save(context.Context) error {
	fmt.Println("Saving", m)
	mainStructMap[m.Email.Address] = m
	for _, v := range m.Editor.Blocks {
		fmt.Println("Block:", v.Type(), v.Data())
	}
	return nil
}

//	func (m *MainStruct) AdminForm(r *http.Request, app *admin.AppDefinition, model *admin.ModelDefinition) modelforms.ModelForm[attrs.Definer] {
//
//		var f modelforms.ModelForm[attrs.Definer] = modelforms.NewBaseModelForm[attrs.Definer](m)
//		return forms.Initialize(
//			f, forms.WithFields(
//				fields.EmailField(
//					fields.Label("Email"),
//					fields.HelpText("Enter your email"),
//					fields.Name("Email"),
//					fields.Required(true),
//					fields.MinLength(5),
//					fields.MaxLength(250),
//				),
//				fields.CharField(
//					fields.Label("Name"),
//					fields.HelpText("Enter your name"),
//					fields.Name("Name"),
//					fields.Required(true),
//					fields.Regex(`^[a-zA-Z\s+]+$`),
//					fields.MinLength(2),
//					fields.MaxLength(50),
//				),
//				auth.NewPasswordField(
//					fields.Label("Password"),
//					fields.HelpText("Enter your password"),
//					fields.Name("Password"),
//					fields.Required(true),
//					fields.MinLength(8),
//					fields.MaxLength(50),
//				),
//			),
//		)
//	}

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
		// Fields:              []string{"Email", "Name", "Password", "Age", "Data", "Block"},
		Labels: map[string]func() string{
			"Email":    fields.S("Object Email"),
			"Name":     fields.S("Object Name"),
			"Password": fields.S("Object Password"),
			"Age":      fields.S("Object Age"),
			"Data":     fields.S("Object Data"),
			"Block":    fields.S("Object Block"),
		},
		AddView: admin.FormViewOptions{
			ViewOptions: admin.ViewOptions{
				Fields: []string{
					"Email",
					"Name",
					"Age",
					"Password",
					"Editor",
				},
			},
			Panels: []admin.Panel{
				admin.TitlePanel(
					admin.FieldPanel("Email"),
				),
				admin.MultiPanel(
					admin.FieldPanel("Name"),
					admin.FieldPanel("Age"),
				),
				admin.FieldPanel("Password"),
				admin.FieldPanel("Editor"),
				//admin.MultiPanel(
				//	admin.FieldPanel("CreatedAt"),
				//	admin.FieldPanel("UpdatedAt"),
				//),
			},
		},
		EditView: admin.FormViewOptions{
			ViewOptions: admin.ViewOptions{
				Fields: []string{
					"Email",
					"Editor",
					"Name",
					"Password",
					"Age",
					"Block",
				},
			},
		},
		ListView: admin.ListViewOptions{
			Format: map[string]func(interface{}) interface{}{
				"Age": func(v any) interface{} {
					return fmt.Sprintf("%d years old", v)
				},
				"Password": func(v any) interface{} {
					return "********"
				},
				"Data": func(v any) interface{} {
					var data = &editor.EditorJSData{}
					if err := json.Unmarshal(v.(json.RawMessage), &data); err != nil {
						return fmt.Sprintf("Error: %s", err)
					}
					var d, _ = editor.ValueToGo(nil, *data)
					return d.Render()
				},
				"Block": func(v any) interface{} {
					return "Block"
				},
				"Editor": func(v any) interface{} {
					if v == nil {
						return "No data"
					}
					var data = v.(*editor.EditorJSBlockData).Render()
					return template.HTML(data)
				},
			},
			Columns: map[string]list.ListColumn[attrs.Definer]{
				"Editor": list.HTMLColumn(
					fields.S("Editor Data"),
					func(defs attrs.Definitions, row attrs.Definer) interface{} {
						return row.(*MainStruct).Editor.Render()
					},
				),
			},
		},
		Model: &MainStruct{},
		GetForID: func(identifier any) (attrs.Definer, error) {
			var id, ok = identifier.(string)
			if !ok {
				return nil, fmt.Errorf("Invalid identifier: %T", identifier)
			}
			var email, err = mail.ParseAddress(id)
			if err != nil {
				return nil, err
			}
			m, ok := mainStructMap[email.Address]
			if !ok {
				return nil, fmt.Errorf("No object found with id: %s", email.Address)
			}
			return m, nil
		},
		GetList: func(amount, offset uint, fields []string) ([]attrs.Definer, error) {
			var items = make([]attrs.Definer, 0, len(mainStructMap))
			for _, v := range mainStructMap {
				items = append(items, v)
			}
			slices.SortFunc(items, func(a, b attrs.Definer) int {
				return strings.Compare(attrs.Get[string](a, "Email"), attrs.Get[string](b, "Email"))
			})

			if len(items) > 0 {
				fmt.Printf("Items: %+v\n", items[0])
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
			fields.JSONField[json.RawMessage](
				fields.Label("Data"),
				fields.Name("data"),
				fields.Required(true),
				fields.Widget(
					editor.NewEditorJSWidget(
						"paragraph",
						"header",
						"delimiter",
					// "list",
					),
				),
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
			fmt.Println("Form is finalized:", f.ErrorList())
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

		var blockData = s.Data
		var block = &editor.EditorJSData{}
		if err := json.Unmarshal(blockData, block); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		d, err := editor.ValueToGo(nil, *block)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		s.Editor = d

		fmt.Printf("%+v\n", s)

		s.Save(context.Background())
	}

	var context = ctx.RequestContext(r)
	context.Set("Form", form)

	if err := tpl.FRender(w, context, "core", "core/index.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
