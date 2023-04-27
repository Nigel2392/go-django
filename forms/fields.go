package forms

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/forms/validators"
)

type ElementInterface interface {
	String() string
	HTML() template.HTML
}

type FormElement interface {
	// Get the name of the field.
	GetName() string

	// Whether the field has a label.
	HasLabel() bool
	// Get the label for the field.
	Label() ElementInterface

	// Get the field element.
	Field() ElementInterface

	// Get, set or clear the value of the field.
	SetValue(string)
	SetFile(filename string, file io.ReadSeekCloser) error
	Value() *FormData
	Clear()

	// Validate the field.
	Validate() error

	// Errors
	Errors() []FormError
	AddError(error)
	HasError() bool

	// Relevant attributes to set.
	SetReadOnly(bool)
	SetDisabled(bool)
	SetRequired(bool)
	SetHidden(bool)

	IsFile() bool
}

const (
	TypeText     = "text"
	TypePassword = "password"
	TypeEmail    = "email"
	TypeNumber   = "number"
	TypeRange    = "range"
	TypeTextArea = "textarea"
	TypeCheck    = "checkbox"
	TypeRadio    = "radio"
	TypeSelect   = "select"
	TypeHidden   = "hidden"
	TypeFile     = "file"
)

type Element string

func (e Element) String() string {
	return string(e)
}

func (e Element) HTML() template.HTML {
	return template.HTML(e)
}

type Option struct {
	Value *FormData
	Text  *FormData
}

type FormData struct {
	Val      string
	FileName string
	Reader   io.ReadSeekCloser
}

func (f *FormData) IsFile() bool {
	if f == nil {
		return false
	}
	return f.Reader != nil && f.FileName != ""
}

func (f *FormData) Value() string {
	if f == nil {
		return ""
	}
	return f.Val
}

func (f *FormData) File() (string, io.ReadSeekCloser) {
	if f == nil {
		return "", nil
	}
	return f.FileName, f.Reader
}

type Field struct {
	LabelText    string
	LabelClass   string
	ID           string
	Class        string
	Placeholder  string
	Type         string
	Name         string
	FormValue    *FormData
	Max          int
	Min          int
	Required     bool
	Disabled     bool
	Options      []*Option
	Autocomplete string

	// FORMAT: "%s is required"
	ErrorMessageFieldRequired string
	// FORMAT: "%s is too long"
	ErrorMessageFieldMax string
	// FORMAT: "%s is too short"
	ErrorMessageFieldMin string
	// FORMAT: "%s is not a valid number (%s)"
	ErrorMessageNaN string

	Validators []validators.Validator

	FormErrors FormErrors

	// Render function
	RenderLabel func(f *Field) Element
	Render      func(f *Field) Element
}

func (f *Field) IsFile() bool {
	return f.Type == TypeFile
}

func (f *Field) SetFile(filename string, file io.ReadSeekCloser) error {
	if f.Type != TypeFile {
		return errors.New("field is not a file field")
	}
	f.FormValue = &FormData{
		FileName: filename,
		Reader:   file,
	}
	return nil
}

func (f *Field) GetName() string {
	return f.Name
}

func (f *Field) HasLabel() bool {
	return f.LabelText != ""
}

func (f *Field) Errors() []FormError {
	return f.FormErrors
}

func (f *Field) AddError(err error) {
	f.FormErrors = append(f.FormErrors, FormError{
		Name:     f.Name,
		FieldErr: err,
	})
}

func (f *Field) HasError() bool {
	return len(f.FormErrors) > 0
}

func (f *Field) SetValue(value string) {
	f.FormValue = &FormData{
		Val: value,
	}
}

func (f *Field) Value() *FormData {
	return f.FormValue
}

func (f *Field) Clear() {
	f.FormValue = &FormData{}
}

func (f *Field) SetReadOnly(readOnly bool) {
	f.Disabled = readOnly
}

func (f *Field) SetDisabled(disabled bool) {
	f.Disabled = disabled
}

func (f *Field) SetRequired(required bool) {
	f.Required = required
}

func (f *Field) SetHidden(hidden bool) {
	f.Type = TypeHidden
}

func (f *Field) String() string {
	return string(f.Label().HTML()) + string(f.Field().HTML())
}

func (f *Field) Field() ElementInterface {
	if f.Render != nil {
		return f.Render(f)
	}
	var attrStringBuilder = strings.Builder{}
	if f.Type == "" {
		attrStringBuilder.WriteString(` type="text"`)
	} else {
		attrStringBuilder.WriteString(` type="` + f.Type + `"`)
	}
	if f.ID != "" {
		attrStringBuilder.WriteString(` id="` + f.ID + `"`)
	} else {
		attrStringBuilder.WriteString(` id="` + f.Name + `"`)
	}
	if f.Name != "" {
		attrStringBuilder.WriteString(` name="` + f.Name + `"`)
	}
	if f.Placeholder != "" {
		attrStringBuilder.WriteString(` placeholder="` + f.Placeholder + `"`)
	}
	if f.Class != "" {
		attrStringBuilder.WriteString(` class="` + f.Class + `"`)
	}
	if f.FormValue != nil && f.Type != TypeFile && f.FormValue.Val != "" {
		attrStringBuilder.WriteString(` value="` + f.FormValue.Val + `"`)
	}
	if f.Max > 0 {
		attrStringBuilder.WriteString(` max="` + strconv.Itoa(f.Max) + `"`)
	}
	if f.Min > 0 {
		attrStringBuilder.WriteString(` min="` + strconv.Itoa(f.Min) + `"`)
	}
	if f.Required {
		attrStringBuilder.WriteString(` required`)
	}
	if f.Disabled {
		attrStringBuilder.WriteString(` disabled`)
	}
	if f.Autocomplete != "" {
		attrStringBuilder.WriteString(` autocomplete="` + f.Autocomplete + `"`)
	}
	var attrs = attrStringBuilder.String()
	switch f.Type {

	case "text", "password", "email", "number", "range", "hidden":
		return Element(`<input` + attrs + `>` + "\r\n")
	case "file":
		if f.FormValue != nil && f.FormValue.Val != "" {
			var b strings.Builder
			b.WriteString(`<p class="form-control">`)
			b.WriteString(f.FormValue.Val)
			b.WriteString(`</p>`)
			b.WriteString(`<input` + attrs + `>` + "\r\n")
			return Element(b.String())
		} else {
			return Element(`<input` + attrs + `>` + "\r\n")
		}
	case "textarea":
		if f.FormValue != nil && f.FormValue.Val != "" {
			return Element(`<textarea` + attrs + `>` + f.FormValue.Val + `</textarea>` + "\r\n")
		}
		return Element(`<textarea` + attrs + `>` + `</textarea>` + "\r\n")

	case "checkbox":
		if f.FormValue != nil && f.FormValue.Val != "" && strings.ToLower(f.FormValue.Val) == "on" || strings.ToLower(f.FormValue.Val) == "true" {
			return Element(`<input` + attrs + ` checked>` + "\r\n")
		}
		return Element(`<input` + attrs + `>` + "\r\n")

	case "radio":
		var b = Element(`<input` + attrs + `>` + "\r\n")
		return b

	case "select":
		var b = Element(`<select` + attrs + `>`)
		for _, option := range f.Options {
			b += Element(`<option value="` + option.Value.Value() + `">` + option.Text.Value() + `</option>`)
		}
		b += Element(`</select>`)
		return b
	}
	return Element("")
}

func (f *Field) Label() ElementInterface {
	if f.RenderLabel != nil {
		return f.RenderLabel(f)
	}
	if f.LabelText == "" {
		return Element("")
	}
	var LabelClass = ""
	if f.LabelClass != "" {
		LabelClass = ` class="` + f.LabelClass + `"`
	}
	if f.ID == "" {
		f.ID = f.Name
	}
	return Element(`<label for="` + f.ID + `"` + LabelClass + `>` + f.LabelText + `</label>` + "\r\n")
}

func (f *Field) Validate() error {
	// VALIDATE REQUIRED
	if f.Required && f.FormValue == nil || f.Required && f.FormValue != nil && f.FormValue.Val == "" {
		if f.ErrorMessageFieldRequired != "" {
			return fmt.Errorf(f.ErrorMessageFieldRequired, f.LabelText)
		}
		return fmt.Errorf("%s is required", f.LabelText)
	} else if f.FormValue == nil {
		return nil
	}

	// VALIDATE LENGTH
	switch f.Type {
	case "number", "range":
		var v string
		if f.FormValue == nil && f.FormValue.Val == "" {
			v = "0"
		} else if f.FormValue != nil {
			v = f.FormValue.Val
		} else {
			v = "0"
		}
		var i, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%s is not a valid number (%s)", f.LabelText, f.FormValue)
		}

		if f.Max > 0 && i > f.Max {
			if f.ErrorMessageFieldMax != "" {
				return fmt.Errorf(f.ErrorMessageFieldMax, f.LabelText)
			}
			return fmt.Errorf("%s is too large", f.LabelText)
		}

		if f.Min > 0 && i < f.Min {
			if f.ErrorMessageFieldMin != "" {
				return fmt.Errorf(f.ErrorMessageFieldMin, f.LabelText)
			}
			return fmt.Errorf("%s is too small", f.LabelText)
		}
	case "file":
	default:
		var v string
		if f.FormValue != nil && f.FormValue.Val != "" {
			v = f.FormValue.Val
		} else {
			v = f.FormValue.Val
		}
		if f.Max > 0 && len(v) > f.Max {
			if f.ErrorMessageFieldMax != "" {
				return fmt.Errorf(f.ErrorMessageFieldMax, f.LabelText)
			}
			return fmt.Errorf("%s is too long by %d characters", f.LabelText, len(v)-f.Max)
		}
		if f.Min > 0 && len(v) < f.Min {
			if f.ErrorMessageFieldMin != "" {
				return fmt.Errorf(f.ErrorMessageFieldMin, f.LabelText)
			}
			return fmt.Errorf("%s is too short by %d characters", f.LabelText, f.Min-len(v))
		}
	}

	if f.Validators != nil {
		for _, validator := range f.Validators {
			if err := validator(f.FormValue); err != nil {
				return err
			}
		}
	}

	return nil
}

type FormValue interface {
	FormValue(name string) string
}

// Generate fields from a struct. The struct must have the following tags:
// `form:"name:VALUE,(params)"` - The name of the field
// `form:"type:VALUE,(params)"` - The type of the field (text, password, email, number, range, textarea, checkbox, radio, select, date, time, datetime)
// `form:"label:VALUE,(params)"` - The label text for the field
// `form:"placeholder:VALUE,(params)"` - The placeholder text for the field
// `form:"class:VALUE,(params)"` - The class for the field
// `form:"required:VALUE,(params)"` - Whether the field is required
// `form:"min:VALUE,(params)"` - The minimum length of the field
// `form:"max:VALUE,(params)"` - The maximum length of the field
// `form:"regex:VALUE,(params)"` - The regex to validate the field against

func GenerateFieldsFromStruct(s interface{}) ([]*Field, error) {
	var fields = make([]*Field, 0)
	var value = reflect.ValueOf(s)
	var typ = reflect.TypeOf(s)
	if typ.Kind() == reflect.Ptr {
		value = value.Elem()
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return fields, errors.New("not a struct")
	}
	for i := 0; i < typ.NumField(); i++ {
		var field = typ.Field(i)
		var value = value.Field(i)
		var name = field.Tag.Get("form")
		if name == "" {
			continue
		}
		var pieces = strings.Split(name, ";")
		var f = Field{}
		f.Name = field.Name
		for _, piece := range pieces {
			var parts = strings.Split(piece, ":")
			if len(parts) < 2 {
				continue
			}

			parts[0] = strings.TrimSpace(parts[0])
			parts[1] = strings.TrimSpace(parts[1])

			if !value.CanInterface() {
				continue
			}
			// Check if it implements a FormValue interface
			if value.Interface() != nil {
				var fv = value.Interface()
				f.FormValue = switchTyp(fv)
			}
			switch strings.ToLower(parts[0]) {
			case "type":
				f.Type = parts[1]
			case "label":
				f.LabelText = parts[1]
			case "placeholder":
				f.Placeholder = parts[1]
			case "class":
				f.Class = parts[1]
			case "required":
				f.Required = true
			case "min":
				var i, err = strconv.Atoi(parts[1])
				if err != nil {
					return fields, err
				}
				f.Min = i
			case "max":
				var i, err = strconv.Atoi(parts[1])
				if err != nil {
					return fields, err
				}
				f.Max = i
			case "regex":
				if f.Validators == nil {
					f.Validators = make([]validators.Validator, 0)
				}
				f.Validators = append(f.Validators, validators.Regex(parts[1]))
			}
		}

		if f.Type == "" {
			var kind = value.Kind()
			switch kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				f.Type = "number"
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				f.Type = "number"
			case reflect.Float32, reflect.Float64:
				f.Type = "number"
			case reflect.Bool:
				f.Type = "checkbox"
			case reflect.String:
				f.Type = "text"
			case reflect.Slice:
				f.Type = "select"
				// Set the options
				var options = make([]*Option, 0)
				for i := 0; i < value.Len(); i++ {
					var v = value.Index(i)
					var o = Option{}
					if v.CanInterface() {
						var fv = v.Interface()
						o.Value = switchTyp(fv)
						o.Text = switchTyp(fv)
					}
					options = append(options, &o)
				}
				f.Options = options
				f.FormValue = &FormData{Val: ""}
			}
		}

		fields = append(fields, &f)
	}
	return fields, nil
}

func switchTyp(t any) *FormData {
	switch val := t.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return &FormData{Val: fmt.Sprintf("%d", val)}
	case float32, float64:
		return &FormData{Val: fmt.Sprintf("%f", val)}
	case bool:
		return &FormData{Val: fmt.Sprintf("%t", val)}
	case string:
		return &FormData{Val: val}
	case []byte:
		return &FormData{Val: string(val)}
	case time.Time:
		return &FormData{Val: val.Format("2006-01-02 15:04:05")}
	case fmt.Stringer:
		return &FormData{Val: val.String()}
	default:
		return &FormData{Val: fmt.Sprintf("%v", val)}
	}
}
