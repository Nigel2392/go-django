package widgets_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/pkg/errors"
)

func init() {
	forms.InitTemplateLibrary()
}

func TestNewTextInput(t *testing.T) {
	attrs := map[string]string{"placeholder": "Enter text", "class": "text-input"}
	textInput := widgets.NewTextInput(nil)

	// Check widget type
	if textInput.FormType() != "text" {
		t.Errorf("Expected widget type 'text', but got '%s'", textInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := textInput.Render(&buffer, "text-input-id", "text-input-name", "", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="text"`) {
		t.Errorf("Rendered HTML should contain 'type=\"text\"' attribute")
	}
	if !strings.Contains(output, `placeholder="Enter text"`) {
		t.Errorf("Rendered HTML should contain 'placeholder=\"Enter text\"' attribute")
	}
	if !strings.Contains(output, `class="text-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"text-input\"' attribute")
	}
	if !strings.Contains(output, `id="text-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"text-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="text-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"text-input-name\"' attribute")
	}
}

func TestNewTextarea(t *testing.T) {
	attrs := map[string]string{"placeholder": "Enter text", "class": "textarea"}
	textarea := widgets.NewTextarea(nil)

	// Check widget type
	if textarea.FormType() != "textarea" {
		t.Errorf("Expected widget type 'textarea', but got '%s'", textarea.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := textarea.Render(&buffer, "textarea-id", "textarea-name", "", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper textarea element with attributes
	if !strings.Contains(output, `<textarea`) {
		t.Errorf("Rendered HTML should contain a <textarea> element")
	}
	if !strings.Contains(output, `placeholder="Enter text"`) {
		t.Errorf("Rendered HTML should contain 'placeholder=\"Enter text\"' attribute")
	}
	if !strings.Contains(output, `class="textarea"`) {
		t.Errorf("Rendered HTML should contain 'class=\"textarea\"' attribute")
	}
	if !strings.Contains(output, `id="textarea-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"textarea-id\"' attribute")
	}
	if !strings.Contains(output, `name="textarea-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"textarea-name\"' attribute")
	}
}

func TestNewEmailInput(t *testing.T) {
	attrs := map[string]string{"class": "email-input"}
	emailInput := widgets.NewEmailInput(nil)

	// Check widget type
	if emailInput.FormType() != "email" {
		t.Errorf("Expected widget type 'email', but got '%s'", emailInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := emailInput.Render(&buffer, "email-input-id", "email-input-name", "", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="email"`) {
		t.Errorf("Rendered HTML should contain 'type=\"email\"' attribute")
	}
	if !strings.Contains(output, `class="email-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"email-input\"' attribute")
	}
	if !strings.Contains(output, `id="email-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"email-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="email-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"email-input-name\"' attribute")
	}
}

func TestNewPasswordInput(t *testing.T) {
	attrs := map[string]string{"class": "password-input"}
	passwordInput := widgets.NewPasswordInput(nil)

	// Check widget type
	if passwordInput.FormType() != "password" {
		t.Errorf("Expected widget type 'password', but got '%s'", passwordInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := passwordInput.Render(&buffer, "password-input-id", "password-input-name", "", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="password"`) {
		t.Errorf("Rendered HTML should contain 'type=\"password\"' attribute")
	}
	if !strings.Contains(output, `class="password-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"password-input\"' attribute")
	}
	if !strings.Contains(output, `id="password-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"password-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="password-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"password-input-name\"' attribute")
	}
}

func TestNewHiddenInput(t *testing.T) {
	attrs := map[string]string{"value": "secret-value"}
	hiddenInput := widgets.NewHiddenInput(nil)

	// Check widget type
	if hiddenInput.FormType() != "hidden" {
		t.Errorf("Expected widget type 'hidden', but got '%s'", hiddenInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := hiddenInput.Render(&buffer, "hidden-input-id", "hidden-input-name", "secret-value", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="hidden"`) {
		t.Errorf("Rendered HTML should contain 'type=\"hidden\"' attribute")
	}
	if !strings.Contains(output, `value="secret-value"`) {
		t.Errorf("Rendered HTML should contain 'value=\"secret-value\"' attribute")
	}
	if !strings.Contains(output, `id="hidden-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"hidden-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="hidden-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"hidden-input-name\"' attribute")
	}
}

func TestNewNumberInput(t *testing.T) {
	attrs := map[string]string{"min": "1", "max": "10", "step": "1", "class": "number-input"}
	numberInput := widgets.NewNumberInput[int](nil) // Using int as the type for testing

	// Check widget type
	if numberInput.FormType() != "number" {
		t.Errorf("Expected widget type 'number', but got '%s'", numberInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := numberInput.Render(&buffer, "number-input-id", "number-input-name", 5, attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="number"`) {
		t.Errorf("Rendered HTML should contain 'type=\"number\"' attribute")
	}
	if !strings.Contains(output, `min="1"`) {
		t.Errorf("Rendered HTML should contain 'min=\"1\"' attribute")
	}
	if !strings.Contains(output, `max="10"`) {
		t.Errorf("Rendered HTML should contain 'max=\"10\"' attribute")
	}
	if !strings.Contains(output, `step="1"`) {
		t.Errorf("Rendered HTML should contain 'step=\"1\"' attribute")
	}
	if !strings.Contains(output, `class="number-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"number-input\"' attribute")
	}
	if !strings.Contains(output, `id="number-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"number-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="number-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"number-input-name\"' attribute")
	}
	if !strings.Contains(output, `value="5"`) {
		t.Errorf("Rendered HTML should contain 'value=\"5\"' attribute")
	}
}

func TestNewBooleanInput(t *testing.T) {
	attrs := map[string]string{"class": "checkbox-input"}
	booleanInput := widgets.NewBooleanInput(nil)

	// Check widget type
	if booleanInput.FormType() != "checkbox" {
		t.Errorf("Expected widget type 'checkbox', but got '%s'", booleanInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := booleanInput.Render(&buffer, "boolean-input-id", "boolean-input-name", true, attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="checkbox"`) {
		t.Errorf("Rendered HTML should contain 'type=\"checkbox\"' attribute")
	}
	if !strings.Contains(output, `class="checkbox-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"checkbox-input\"' attribute")
	}
	if !strings.Contains(output, `id="boolean-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"boolean-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="boolean-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"boolean-input-name\"' attribute")
	}
	if !strings.Contains(output, `checked`) {
		t.Errorf("Rendered HTML should contain 'checked' attribute when value is true")
	}
}

func TestNewDateInput(t *testing.T) {
	attrs := map[string]string{"class": "date-input"}
	dateInput := widgets.NewDateInput(nil, widgets.DateWidgetTypeDate) // Assuming `DateWidgetTypeDate` is available

	// Check widget type
	if dateInput.FormType() != "date" {
		t.Errorf("Expected widget type 'date', but got '%s'", dateInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := dateInput.Render(&buffer, "date-input-id", "date-input-name", "2024-01-01", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="date"`) {
		t.Errorf("Rendered HTML should contain 'type=\"date\"' attribute")
	}
	if !strings.Contains(output, `class="date-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"date-input\"' attribute")
	}
	if !strings.Contains(output, `id="date-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"date-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="date-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"date-input-name\"' attribute")
	}
	if !strings.Contains(output, `value="2024-01-01"`) {
		t.Errorf("Rendered HTML should contain 'value=\"2024-01-01\"' attribute")
	}
}

func TestNewFileInput(t *testing.T) {
	attrs := map[string]string{"class": "file-input"}

	// Validator to simulate file validation
	mockValidator := func(filename string, file io.Reader) error {
		if filename != "testfile.txt" {
			return errors.New("invalid filename")
		}

		// Check if file content is not empty
		buf := make([]byte, 4)
		_, err := file.Read(buf)
		if err != nil {
			return errors.New("error reading file")
		}

		if string(buf) != "file" {
			return errors.New("invalid file content")
		}

		return nil
	}

	// Create the file input widget with the mock validator
	fileInput := widgets.NewFileInput(nil, mockValidator)

	// Check widget type
	if fileInput.FormType() != "file" {
		t.Errorf("Expected widget type 'file', but got '%s'", fileInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := fileInput.Render(&buffer, "file-input-id", "file-input-name", nil, attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains proper input element with attributes
	if !strings.Contains(output, `<input`) {
		t.Errorf("Rendered HTML should contain an <input> element")
	}
	if !strings.Contains(output, `type="file"`) {
		t.Errorf("Rendered HTML should contain 'type=\"file\"' attribute")
	}
	if !strings.Contains(output, `class="file-input"`) {
		t.Errorf("Rendered HTML should contain 'class=\"file-input\"' attribute")
	}
	if !strings.Contains(output, `id="file-input-id"`) {
		t.Errorf("Rendered HTML should contain 'id=\"file-input-id\"' attribute")
	}
	if !strings.Contains(output, `name="file-input-name"`) {
		t.Errorf("Rendered HTML should contain 'name=\"file-input-name\"' attribute")
	}

	// Now simulate passing a file to the validator
	mockFile := bytes.NewReader([]byte("file content")) // Simulates a file with content
	err = mockValidator("testfile.txt", mockFile)
	if err != nil {
		t.Errorf("Expected no error for valid filename and file, but got: %v", err)
	}

	// Test with invalid file name
	err = mockValidator("invalidfile.txt", mockFile)
	if err == nil || err.Error() != "invalid filename" {
		t.Errorf("Expected 'invalid filename' error, but got: %v", err)
	}

	// Test for valid file but reading error
	emptyFile := bytes.NewReader([]byte{}) // Simulates an empty file
	err = mockValidator("testfile.txt", emptyFile)
	if err == nil || err.Error() != "error reading file" {
		t.Errorf("Expected 'error reading file' error, but got: %v", err)
	}
}

// Mock options for testing
func mockOptions() []widgets.Option {
	return []widgets.Option{
		widgets.NewOption("option1", "Option 1", "1"),
		widgets.NewOption("option2", "Option 2", "2"),
	}
}

func TestNewCheckboxInput(t *testing.T) {
	attrs := map[string]string{"class": "checkbox-input"}
	checkboxInput := options.NewCheckboxInput(attrs, mockOptions)

	// Check widget type
	if checkboxInput.FormType() != "checkbox" {
		t.Errorf("Expected widget type 'checkbox', but got '%s'", checkboxInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := checkboxInput.Render(&buffer, "checkbox-input-id", "checkbox-input-name", "1", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains checkbox inputs and associated labels
	if !strings.Contains(output, `<input type="checkbox"`) {
		t.Errorf("Rendered HTML should contain <input type=\"checkbox\"> elements")
	}
	if !strings.Contains(output, `checked`) {
		t.Errorf("Rendered HTML should contain 'checked' attribute for the selected option")
	}
	if !strings.Contains(output, `value="1"`) || !strings.Contains(output, `value="2"`) {
		t.Errorf("Rendered HTML should contain the correct choice values '1' and '2'")
	}
	if !strings.Contains(output, `Option 1`) || !strings.Contains(output, `Option 2`) {
		t.Errorf("Rendered HTML should contain the correct choice labels 'Option 1' and 'Option 2'")
	}
}

func TestNewRadioInput(t *testing.T) {
	attrs := map[string]string{"class": "radio-input"}
	radioInput := options.NewRadioInput(attrs, mockOptions)

	// Check widget type
	if radioInput.FormType() != "radio" {
		t.Errorf("Expected widget type 'radio', but got '%s'", radioInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := radioInput.Render(&buffer, "radio-input-id", "radio-input-name", "1", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains radio inputs and associated labels
	if !strings.Contains(output, `<input type="radio"`) {
		t.Errorf("Rendered HTML should contain <input type=\"radio\"> elements")
	}
	if !strings.Contains(output, `checked`) {
		t.Errorf("Rendered HTML should contain 'checked' attribute for the selected option")
	}
	if !strings.Contains(output, `value="1"`) || !strings.Contains(output, `value="2"`) {
		t.Errorf("Rendered HTML should contain the correct choice values '1' and '2'")
	}
	if !strings.Contains(output, `Option 1`) || !strings.Contains(output, `Option 2`) {
		t.Errorf("Rendered HTML should contain the correct choice labels 'Option 1' and 'Option 2'")
	}
}

func TestNewSelectInput(t *testing.T) {
	attrs := map[string]string{"class": "select-input"}
	selectInput := options.NewSelectInput(attrs, mockOptions)

	// Check widget type
	if selectInput.FormType() != "select" {
		t.Errorf("Expected widget type 'select', but got '%s'", selectInput.FormType())
	}

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := selectInput.Render(&buffer, "select-input-id", "select-input-name", "1", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains a select element and option elements
	if !strings.Contains(output, `<select`) {
		t.Errorf("Rendered HTML should contain a <select> element")
	}
	if !strings.Contains(output, `selected`) {
		t.Errorf("Rendered HTML should contain 'selected' attribute for the selected option")
	}
	if !strings.Contains(output, `<option value="1"`) || !strings.Contains(output, `<option value="2"`) {
		t.Errorf("Rendered HTML should contain the correct <option> elements for values '1' and '2'")
	}
	if !strings.Contains(output, `Option 1`) || !strings.Contains(output, `Option 2`) {
		t.Errorf("Rendered HTML should contain the correct option labels 'Option 1' and 'Option 2'")
	}
}

func TestNewSelectInputWithBlankOption(t *testing.T) {
	attrs := map[string]string{"class": "select-input"}
	selectInput := options.NewSelectInput(attrs, mockOptions)
	// Set the option to include a blank option
	selectInput.IncludeBlank = true

	// Render the widget and check the output HTML
	var buffer bytes.Buffer
	err := selectInput.Render(&buffer, "select-input-id", "select-input-name", "1", attrs)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buffer.String()

	// Check that HTML contains a select element and a blank option
	if !strings.Contains(output, `<select`) {
		t.Errorf("Rendered HTML should contain a <select> element")
	}
	if !strings.Contains(output, `<option value=""`) {
		t.Errorf("Rendered HTML should contain a blank <option> element")
	}
	if !strings.Contains(output, `---------`) { // Default blank label
		t.Errorf("Rendered HTML should contain the default blank label '---------'")
	}
}
