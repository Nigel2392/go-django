package forms

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/forms/validators"
)

const (
	TypeText     = "text"
	TypePassword = "password"
	TypeEmail    = "email"
	TypeNumer    = "number"
	TypeRange    = "range"
	TypeTextArea = "textarea"
	TypeCheck    = "checkbox"
	TypeRadio    = "radio"
	TypeSelect   = "select"
	TypeHidden   = "hidden"
)

type Element string

func (e Element) String() string {
	return string(e)
}

func (e Element) HTML() template.HTML {
	return template.HTML(e)
}

type Option struct {
	Value string
	Text  string
}

type Field struct {
	LabelText    string
	LabelClass   string
	ID           string
	Class        string
	Placeholder  string
	Type         string
	Name         string
	Value        string
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

	Errors FormErrors
}

func (f *Field) String() string {
	return string(f.Label().HTML()) + string(f.Field().HTML())
}

func (f *Field) Field() Element {
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
	if f.Value != "" {
		attrStringBuilder.WriteString(` value="` + f.Value + `"`)
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

	case "text", "password", "email", "number", "range":
		return Element(`<input` + attrs + `>` + "\r\n")

	case "textarea":
		return Element(`<textarea` + attrs + `>` + f.Value + `</textarea>` + "\r\n")

	case "checkbox":
		if strings.ToLower(f.Value) == "on" || strings.ToLower(f.Value) == "true" {
			return Element(`<input` + attrs + ` checked>` + "\r\n")
		}
		return Element(`<input` + attrs + `>` + "\r\n")

	case "radio":
		var b = Element(`<input` + attrs + `>` + "\r\n")
		return b

	case "select":
		var b = Element(`<select` + attrs + `>`)
		for _, option := range f.Options {
			b += Element(`<option value="` + option.Value + `">` + option.Text + `</option>`)
		}
		b += Element(`</select>`)
		return b

	default:
		return ""
	}
}

func (f *Field) Label() Element {
	if f.LabelText == "" {
		return ""
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
	if f.Required && f.Value == "" {
		if f.ErrorMessageFieldRequired != "" {
			return fmt.Errorf(f.ErrorMessageFieldRequired, f.LabelText)
		}
		return fmt.Errorf("%s is required", f.LabelText)
	} else if f.Value == "" {
		return nil
	}

	// VALIDATE LENGTH
	switch f.Type {
	case "number", "range":
		var i, err = strconv.Atoi(f.Value)
		if err != nil {
			return fmt.Errorf("%s is not a valid number (%s)", f.LabelText, f.Value)
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

	default:
		if f.Max > 0 && len(f.Value) > f.Max {
			if f.ErrorMessageFieldMax != "" {
				return fmt.Errorf(f.ErrorMessageFieldMax, f.LabelText)
			}
			return fmt.Errorf("%s is too long by %d characters", f.LabelText, len(f.Value)-f.Max)
		}
		if f.Min > 0 && len(f.Value) < f.Min {
			if f.ErrorMessageFieldMin != "" {
				return fmt.Errorf(f.ErrorMessageFieldMin, f.LabelText)
			}
			return fmt.Errorf("%s is too short by %d characters", f.LabelText, f.Min-len(f.Value))
		}
	}

	if f.Validators != nil {
		for _, validator := range f.Validators {
			if err := validator(f.Value); err != nil {
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
				f.Value = switchTyp(fv)
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
				f.Value = ""
			}
		}

		fields = append(fields, &f)
	}
	return fields, nil
}

func switchTyp(t any) string {
	switch val := t.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case string:
		return val
	case []byte:
		return string(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}
