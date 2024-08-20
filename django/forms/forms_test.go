package forms_test

import (
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/Nigel2392/django/core/errs"
	forms "github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
)

type boolean bool

func (c *boolean) setter(f forms.Form) {
	*c = true
}

type field interface {
	Field() template.HTML
}

type fakeField struct {
	html string
}

func (f fakeField) Field() template.HTML {
	return template.HTML(f.html)
}

func inputEquals(widget field, type_, name, value string, attrs map[string]string) error {
	var (
		id              = fmt.Sprintf("id_%s", name)
		html            = string(widget.Field())
		tokens []string = make([]string, 0, 3+len(attrs))
	)

	if !strings.HasPrefix(html, "<input") {
		return errors.New("Input should start with '<input'")
	}

	if !strings.HasSuffix(html, "/>") {
		return errors.New("Input should end with '/>'")
	}

	html = strings.TrimPrefix(html, "<input")
	html = strings.TrimSuffix(html, "/>")

	tokens = append(tokens, []string{
		fmt.Sprintf("type=\"%s\"", type_),
		fmt.Sprintf("id=\"%s\"", id),
		fmt.Sprintf("name=\"%s\"", name),
		fmt.Sprintf("value=\"%s\"", value),
	}...)

	for k, v := range attrs {
		tokens = append(tokens, fmt.Sprintf("%s=\"%s\"", k, v))
	}

	var attributeRegex = `(\w+)=("[^"]*")`
	var obj = regexp.MustCompile(attributeRegex)
	var matches = obj.FindAllStringSubmatch(html, -1)

	if len(matches) != len(tokens) {
		return fmt.Errorf("Number of attributes do not match: %d != expected (%d) %v %v", len(matches), len(tokens), matches, tokens)
	}

	for _, token := range tokens {
		var found bool
		for _, match := range matches {
			if match[0] == token {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Token not found: %s", token)
		}
	}

	return nil
}

func GetTestForm() forms.Form {
	var form forms.Form = forms.Initialize(
		forms.NewBaseForm(),
		forms.WithFields(
			fields.EmailField(
				fields.Label("Email"),
				fields.Name("email"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(250),
			),
			fields.CharField(
				fields.Label("Name"),
				fields.Name("name"),
				fields.Required(true),
				fields.Regex(`^[a-zA-Z]+$`),
				fields.MinLength(2),
				fields.MaxLength(50),
			),
			fields.NumberField[int](
				fields.Label("Age"),
				fields.Name("age"),
				fields.Required(true),
			),
		),
	)

	return form
}

func TestInputEquals(t *testing.T) {
	var attrs = map[string]string{
		"minlength": "5",
		"maxlength": "250",
		"required":  "",
	}
	var inputHtml = `<input type="text" id="id_email" name="email" value="test@localhost" minlength="5" maxlength="250" required="" />`
	t.Run("Valid", func(t *testing.T) {
		if err := inputEquals(fakeField{inputHtml}, "text", "email", "test@localhost", attrs); err != nil {
			t.Errorf("Input HTML does not match: %s", err)
		}
	})
	t.Run("Invalid", func(t *testing.T) {
		t.Run("Type", func(t *testing.T) {
			if err := inputEquals(fakeField{inputHtml}, "number", "email", "test@localhost", attrs); err == nil {
				t.Error("Input HTML should not match.")
			}
		})
		t.Run("Name", func(t *testing.T) {
			if err := inputEquals(fakeField{inputHtml}, "text", "name", "test@localhost", attrs); err == nil {
				t.Error("Input HTML should not match.")
			}
		})
		t.Run("Value", func(t *testing.T) {
			if err := inputEquals(fakeField{inputHtml}, "text", "email", "test@localhost1", attrs); err == nil {
				t.Error("Input HTML should not match.")
			}
		})
		t.Run("Attributes", func(t *testing.T) {
			if err := inputEquals(fakeField{inputHtml}, "text", "email", "test@localhost", map[string]string{
				"minlength": "6",
				"maxlength": "249",
				"required":  "",
			}); err == nil {
				t.Error("Input HTML should not match.")
			}
		})
	})
}

func TestFormRequired(t *testing.T) {
	var form = forms.NewBaseForm()
	form = forms.Initialize(
		form,
		forms.WithFields(
			fields.CharField(
				fields.Name("first_name"),
				fields.Required(true),
			),
			fields.CharField(
				fields.Name("last_name"),
				fields.Required(true),
			),
		),
	)

	var addFormData = func(m map[string]string) {
		var urlValues = make(url.Values)
		for k, v := range m {
			urlValues[k] = []string{v}
		}
		form.WithData(
			urlValues,
			nil,
			nil,
		)
	}

	t.Run("RequiredAttributePresent", func(t *testing.T) {
		var (
			fields       = form.BoundFields()
			firstName, _ = fields.Get("first_name")
			lastName, _  = fields.Get("last_name")
		)

		firstName.(*forms.BoundFormField).FormValue = "Firstname"
		lastName.(*forms.BoundFormField).FormValue = "Lastname"

		if err := inputEquals(firstName, "text", "first_name", "Firstname", map[string]string{
			"required": "",
		}); err != nil {
			t.Errorf("Input HTML does not match for field 'first_name': %s", err)
		}

		if err := inputEquals(lastName, "text", "last_name", "Lastname", map[string]string{
			"required": "",
		}); err != nil {
			t.Errorf("Input HTML does not match for field 'last_name': %s", err)
		}
	})

	t.Run("Valid", func(t *testing.T) {
		addFormData(map[string]string{
			"first_name": "Firstname",
			"last_name":  "Lastname",
		})

		if !form.IsValid() {
			t.Error("Empty required field should not be valid.")
		}

		var (
			cleaned   = form.CleanedData()
			firstName interface{}
			lastName  interface{}
			ok        bool
		)

		t.Run("DataPresent", func(t *testing.T) {
			if firstName, ok = cleaned["first_name"]; !ok {
				t.Error("first_name not found in cleaned data")
			}
			if lastName, ok = cleaned["last_name"]; !ok {
				t.Error("last_name not found in cleaned data")
			}
		})

		t.Run("DataMatches", func(t *testing.T) {
			if firstName != "Firstname" {
				t.Errorf("first_name does not match expected: %s != %s", firstName, "FirstName")
			}
			if lastName != "Lastname" {
				t.Errorf("last_name does not match expected: %s != %s", lastName, "Lastname")
			}
		})
	})

	t.Run("Invalid", func(t *testing.T) {
		addFormData(map[string]string{
			"first_name": "",
			"last_name":  "",
		})

		if form.IsValid() {
			t.Error("Empty required field should not be valid.")
		}

		var (
			formErrors   = form.BoundErrors()
			firstNameErr []error
			lastNameErr  []error
			ok           bool
		)
		t.Run("ErrorsPresent", func(t *testing.T) {
			firstNameErr, ok = formErrors.Get("first_name")
			if !ok {
				t.Error("Required field first_name should have an error.")
			}
			lastNameErr, ok = formErrors.Get("last_name")
			if !ok {
				t.Error("Required field last_name should have an error.")
			}

		})
		t.Run("ErrorsTypeCheck", func(t *testing.T) {
			if !errors.Is(firstNameErr[0], errs.ErrFieldRequired) {
				t.Error("Required field first_name did not give an error of type FieldRequired")
			}
			if !errors.Is(lastNameErr[0], errs.ErrFieldRequired) {
				t.Error("Required field last_name did not give an error of type FieldRequired")
			}
		})

		t.Run("ErrorsNotEquals", func(t *testing.T) {
			if errors.Is(firstNameErr[0], lastNameErr[0]) {
				t.Errorf("Two field-specific ValidationErrors should not be equal: %T == %T", firstNameErr[0], lastNameErr[0])
			}
		})

	})
}

func HTMLEqual(s1, s2 string) bool {
	return true
}

func TestFormValidation(t *testing.T) {
	var (
		form             forms.Form = GetTestForm()
		isvalidCalled    boolean
		invalidCalled    boolean
		onfinalizeCalled boolean
	)

	form = forms.Initialize(
		form,
		forms.OnValid(isvalidCalled.setter),
		forms.OnInvalid(invalidCalled.setter),
		forms.OnFinalize(onfinalizeCalled.setter),
	)

	var raiseInvalid = func(t *testing.T, title string) {
		var b = new(strings.Builder)
		b.WriteString(title)
		var errs = form.BoundErrors()
		for head := errs.Front(); head != nil; head = head.Next() {
			for _, err := range head.Value {
				b.WriteString(err.Error())
			}
		}
		t.Error(b.String())
	}

	var raiseValid = func(t *testing.T, title string) {
		var b = new(strings.Builder)
		b.WriteString(title)
		var cleaned = form.CleanedData()
		for k, v := range cleaned {
			fmt.Fprintf(b, "%s: %v %T\n", k, v, v)
		}

		t.Error(b.String())
	}

	var TestOnValid = func(t *testing.T) {
		if !form.IsValid() {
			raiseInvalid(t, "Form is invalid:")
		}
	}

	var TestOnInvalid = func(t *testing.T) {
		if form.IsValid() {
			raiseValid(t, "Form is valid:")
		}
	}

	var TestOnValidCallback = func(t *testing.T) {
		if invalidCalled {
			raiseInvalid(t, "Form should not call OnInvalid:")
		}

		if !isvalidCalled {
			raiseValid(t, "Form should have called OnValid:")
		}

	}

	var TestOnInvalidCallback = func(t *testing.T) {
		if isvalidCalled {
			raiseValid(t, "Form should not call OnValid:")
		}

		if !invalidCalled {
			raiseValid(t, "Form should have called OnInvalid:")
		}
	}

	var TestFinalizeCalled = func(t *testing.T) {
		if !onfinalizeCalled {
			t.Error("Form OnFinalize should have been called ")
		}
	}

	var TestBoundIds = func(t *testing.T, email, name, age forms.BoundField) func(t *testing.T) {
		return func(t *testing.T) {
			if email.ID() != "id_email" {
				t.Error("ID does not match for field 'email': ", email.ID())
			}
			if name.ID() != "id_name" {
				t.Error("ID does not match for field 'age': ", age.ID())
			}
			if age.ID() != "id_age" {
				t.Error("ID does not match for field 'age': ", age.ID())
			}
		}
	}

	var TestBoundLabels = func(t *testing.T, email, name, age forms.BoundField) func(t *testing.T) {
		return func(t *testing.T) {
			if string(email.Label()) != fmt.Sprintf("<label for=\"%s\">%s</label>", "id_email", "Email") {
				t.Error("Label HTML does not match for field 'email': ", email.Label())
			}
			if string(name.Label()) != fmt.Sprintf("<label for=\"%s\">%s</label>", "id_name", "Name") {
				t.Error("Label HTML does not match for field 'age': ", age.Label())
			}
			if string(age.Label()) != fmt.Sprintf("<label for=\"%s\">%s</label>", "id_age", "Age") {
				t.Error("Label HTML does not match for field 'age': ", age.Label())
			}
		}
	}

	var TestBoundFieldsRender = func(t *testing.T, email, name, age forms.BoundField, expected map[string]string) func(t *testing.T) {
		return func(t *testing.T) {
			if err := inputEquals(email, "email", "email", expected["email"], map[string]string{
				"minlength": "5",
				"maxlength": "250",
				"required":  "",
			}); err != nil {
				t.Errorf("Input HTML does not match for field 'email': %s", err)
			}
			if err := inputEquals(name, "text", "name", expected["name"], map[string]string{
				"minlength": "2",
				"maxlength": "50",
				"required":  "",
			}); err != nil {
				t.Errorf("Input HTML does not match for field 'name': %s", err)
			}
			if err := inputEquals(age, "number", "age", expected["age"], map[string]string{
				"required": "",
			}); err != nil {
				t.Errorf("Input HTML does not match for field 'age': %s", err)
			}

		}
	}

	var TestBoundValues = func(t *testing.T, email, name, age forms.BoundField, expected map[string]string) func(t *testing.T) {
		return func(t *testing.T) {
			if email.Value() != expected["email"] {
				t.Errorf("BoundField 'email' does not match expected value: %v != %v, %T != %T", email.Value(), expected["email"], email.Value(), expected["email"])
			}
			if name.Value() != expected["name"] {
				t.Errorf("BoundField 'name' does not match expected value: %v != %v, %T != %T", name.Value(), expected["name"], name.Value(), expected["name"])
			}
			if age.Value() != expected["age"] {
				t.Errorf("BoundField 'age' does not match expected value: %v != %v, %T != %T", age.Value(), expected["age"], age.Value(), expected["age"])
			}
		}
	}

	var TestBoundFields = func(t *testing.T, email, name, age forms.BoundField) {
		t.Run("IDs", TestBoundIds(t, email, name, age))
		t.Run("Labels", TestBoundLabels(t, email, name, age))
	}

	var TestBoundValidErrors = func(t *testing.T, email, name, age forms.BoundField) func(t *testing.T) {
		return func(t *testing.T) {
			if len(email.Errors()) > 0 {
				t.Error("Valid BoundField 'email' should not have errors.")
			}
			if len(name.Errors()) > 0 {
				t.Error("Valid BoundField 'name' should not have errors.")
			}
			if len(age.Errors()) > 0 {
				t.Error("Valid BoundField 'age' should not have errors.")
			}
		}
	}

	var TestValidBoundFields = func(t *testing.T) {
		var (
			fields   = form.BoundFields()
			email, _ = fields.Get("email")
			name, _  = fields.Get("name")
			age, _   = fields.Get("age")
		)

		TestBoundFields(t, email, name, age)

		var data = map[string]string{
			"email": "test@localhost",
			"name":  "John",
			"age":   "18",
		}

		t.Run("Fields", TestBoundFieldsRender(t, email, name, age, data))
		t.Run("Errors", TestBoundValidErrors(t, email, name, age))
		t.Run("Values", TestBoundValues(t, email, name, age, data))
	}

	var TestBoundInvalidErrors = func(t *testing.T, email, name, age forms.BoundField) func(t *testing.T) {
		return func(t *testing.T) {
			if !(len(email.Errors()) > 0) {
				t.Error("Invalid BoundField 'email' should have errors.")
			}
			if len(name.Errors()) > 0 {
				t.Error("Valid BoundField 'name' should not have errors when another field is invalid.")
			}
			if len(age.Errors()) > 0 {
				t.Error("Valid BoundField 'age' should not have errors when another field is invalid.")
			}
		}
	}

	var TestInvalidBoundFields = func(t *testing.T) {
		var (
			fields   = form.BoundFields()
			email, _ = fields.Get("email")
			name, _  = fields.Get("name")
			age, _   = fields.Get("age")
		)

		TestBoundFields(t, email, name, age)

		var data = map[string]string{
			"email": "test/localhost",
			"name":  "John",
			"age":   "18",
		}

		t.Run("Fields", TestBoundFieldsRender(t, email, name, age, data))
		t.Run("Errors", TestBoundInvalidErrors(t, email, name, age))
		t.Run("Values", TestBoundValues(t, email, name, age, data))
	}

	t.Run("Valid", func(t *testing.T) {
		var data = url.Values{
			"email": []string{"test@localhost"},
			"name":  []string{"John"},
			"age":   []string{"18"},
		}

		form.WithData(data, nil, nil)

		TestOnValid(t)

		t.Run("Callback", TestOnValidCallback)
		t.Run("Finalize", TestFinalizeCalled)
		t.Run("BoundFields", TestValidBoundFields)
	})

	isvalidCalled = false
	invalidCalled = false
	onfinalizeCalled = false

	t.Run("Invalid", func(t *testing.T) {
		var data = url.Values{
			"email": []string{"test/localhost"},
			"name":  []string{"John"},
			"age":   []string{"18"},
		}

		form.WithData(data, nil, nil)

		TestOnInvalid(t)

		t.Run("Callback", TestOnInvalidCallback)
		t.Run("Finalize", TestFinalizeCalled)
		t.Run("BoundFields", TestInvalidBoundFields)
	})
}
