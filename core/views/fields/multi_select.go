package fields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
)

type Option struct {
	Val  string `json:"value"`
	Text string `json:"text"`
}

func AutoOption(v any) Option {
	var o Option
	switch v.(type) {
	case string:
		o = Option{Val: v.(string), Text: v.(string)}
	case int, int8, int16, int32, int64:
		o = Option{Val: fmt.Sprintf("%d", v), Text: fmt.Sprintf("%d", v)}
	case float32, float64:
		o = Option{Val: fmt.Sprintf("%f", v), Text: fmt.Sprintf("%f", v)}
	case bool:
		o = Option{Val: fmt.Sprintf("%t", v), Text: fmt.Sprintf("%t", v)}
	default:
		o = Option{Val: fmt.Sprintf("%v", v), Text: fmt.Sprintf("%v", v)}
	}
	return o
}

func (o Option) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

func (o *Option) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &o.Val)
}

func (o Option) Value() string {
	return o.Val
}

func (o Option) Label() string {
	return o.Text
}

// When using this field with the forms, you should fill the right and left values.
//
// The left values will be stored on the database, and the values on the right will not be stored.
type DoubleMultipleSelectField struct {
	Left  []interfaces.Option/* Selected */ `json:"left"` // Left values will be stored on the database!
	Right []interfaces.Option/* Unselected */ `json:"-"`  // Right will not be stored in the database!
}

func (i *DoubleMultipleSelectField) Initial(r *request.Request, model any, fieldName string) {
	var getOptionsFuncName = fmt.Sprintf("Get%sOptions", fieldName)
	var valueOf = reflect.ValueOf(model)
	var method = valueOf.MethodByName(getOptionsFuncName)
	if !method.IsValid() {
		panic(fmt.Sprintf("Method %s does not exist on model %s", getOptionsFuncName, valueOf.Type().Name()))
	}

	switch m := method.Interface().(type) {
	case func() ([]interfaces.Option, []interfaces.Option):
		i.Left, i.Right = m()
	case func(r *request.Request) ([]interfaces.Option, []interfaces.Option):
		i.Left, i.Right = m(r)
	case func(r *request.Request, model any) ([]interfaces.Option, []interfaces.Option):
		i.Left, i.Right = m(r, model)
	case func(r *request.Request, model any, fieldName string) ([]interfaces.Option, []interfaces.Option):
		i.Left, i.Right = m(r, model, fieldName)
	default:
		panic(fmt.Sprintf("Method %s on model %s does not have the correct signature for DoubleMultipleSelectField: %T", getOptionsFuncName, valueOf.Type().Name(), method.Interface()))
	}
}

func (i DoubleMultipleSelectField) Values() []string {
	var s []string
	for _, v := range i.Left {
		s = append(s, v.Value())
	}
	return s
}

func (i *DoubleMultipleSelectField) Scan(src interface{}) error {
	var s string
	switch src.(type) {
	case []byte:
		s = string(src.([]byte))
	case string:
		s = src.(string)
	}
	var err = json.Unmarshal([]byte(s), &i.Left)
	return err
}

func (i DoubleMultipleSelectField) Value() (driver.Value, error) {
	var b, err = json.Marshal(i.Left)
	return string(b), err
}

func (i *DoubleMultipleSelectField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	for _, vv := range v {
		i.Left = append(i.Left, Option{Val: vv})
	}
	return nil
}

func (i DoubleMultipleSelectField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display_text))
}

func (i DoubleMultipleSelectField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	var nameLeft = fmt.Sprintf("%s-left", name)
	var nameRight = fmt.Sprintf("%s-right", name)
	b.WriteString("<div class=\"m2m-container\">\n")
	b.WriteString("\t<div class=\"left m2m-select\">\n")
	b.WriteString("\t\t<select multiple=\"multiple\" name=\"")
	// This is left as name, so when the form is submitted the values are passed under the right name.
	b.WriteString(name)
	b.WriteString("\" id=\"")
	b.WriteString(nameLeft)
	b.WriteString("\" ")
	b.WriteString(TagMapToElementAttributes(tags, AllTagsInput...))
	b.WriteString(">\n")
	for _, v := range i.Left {
		b.WriteString("\t\t\t<option value=\"")
		b.WriteString(v.Value())
		b.WriteString("\">")
		b.WriteString(v.Label())
		b.WriteString("</option>\n")
	}
	b.WriteString("\t\t</select>\n")
	b.WriteString("\t</div>\n")
	b.WriteString("\t<div class=\"m2m-buttons\">\n")
	b.WriteString("\t\t<button type=\"button\" class=\"m2m-add-all add\">Add All</button>\n")
	b.WriteString("\t\t<button type=\"button\" class=\"m2m-remove-all remove\">Remove All</button>\n")
	b.WriteString("\t</div>\n")
	b.WriteString("\t<div class=\"right m2m-select\">\n")
	b.WriteString("\t\t<select multiple=\"multiple\" name=\"")
	b.WriteString(nameRight)
	b.WriteString("\" id=\"")
	b.WriteString(nameRight)
	b.WriteString("\" ")
	b.WriteString(TagMapToElementAttributes(tags, AllTagsInput...))
	b.WriteString(">\n")
	for _, v := range i.Right {
		b.WriteString("\t\t\t<option value=\"")
		b.WriteString(v.Value())
		b.WriteString("\">")
		b.WriteString(v.Label())
		b.WriteString("</option>\n")
	}
	b.WriteString("\t\t</select>\n")
	b.WriteString("\t</div>\n")
	b.WriteString("</div>\n")
	return ElementType(b.String())
}

func (i DoubleMultipleSelectField) Script() (key string, value template.HTML) {
	return "m2m", `<script>
	var form = document.querySelector("form");
	// Check if form is disabled
	if (!form.classList.contains("disabled")) {
		var allBtn = document.querySelectorAll(".m2m-add-all");
		var removeAllBtn = document.querySelectorAll(".m2m-remove-all");
		var options = document.querySelectorAll(".m2m-select option");
		allBtn.forEach(function(btn) {
			btn.addEventListener("click", function() {
				var container = this.parentElement.parentElement;
				var selectors = container.querySelectorAll("select");
				var ownSelect = selectors[0];
				var relatedSelect = selectors[1];
				var options = relatedSelect.options;
				for (var i = 0; i < options.length; i++) {
					var option = document.createElement("option");
					option.value = options[i].value;
					option.text = options[i].text;
					option.selected = true;
					option.addEventListener("dblclick", switchSelected);
					ownSelect.appendChild(option);
				}
				relatedSelect.innerHTML = "";
				setSelected(selectors[0].querySelectorAll("option"));
			});
		});
		removeAllBtn.forEach(function(btn) {
			btn.addEventListener("click", function() {
				var container = this.parentElement.parentElement;
				var selectors = container.querySelectorAll("select");
				var ownSelect = selectors[0];
				var relatedSelect = selectors[1];
				var options = ownSelect.options;
				for (var i = 0; i < options.length; i++) {
					var option = document.createElement("option");
					option.value = options[i].value;
					option.text = options[i].text;
					option.addEventListener("dblclick", switchSelected);
					relatedSelect.appendChild(option);
				}
				ownSelect.innerHTML = "";
				setSelected(selectors[0].querySelectorAll("option"));
			});
		});

		// Add event listener, listen for double click
		options.forEach(function(option) {
			option.addEventListener("dblclick", switchSelected);
		});
		options.forEach(function(option) {
			option.addEventListener("click", function() {
				var container = this.parentElement.parentElement.parentElement;
				var selectors = container.querySelectorAll("select");
				setSelected(selectors[0].querySelectorAll("option"));
			});
		});
		// Switch selected option
		function switchSelected() {
			var container = this.parentElement.parentElement.parentElement;
			var selectors = container.querySelectorAll("select");
			if (this.parentElement === selectors[0]) {
				var ownSelect = selectors[0];
				var relatedSelect = selectors[1];
				var option = document.createElement("option");
				option.value = this.value;
				option.text = this.text;
				relatedSelect.appendChild(option);
				document.removeEventListener(this, switchSelected)
				this.parentElement.removeChild(this);
				option.addEventListener("dblclick", switchSelected)
				setSelected(selectors[0].querySelectorAll("option"));
			} else if (this.parentElement === selectors[1]) {
				var ownSelect = selectors[1];
				var relatedSelect = selectors[0];
				var option = document.createElement("option");
				option.value = this.value;
				option.text = this.text;
				option.selected = true;
				relatedSelect.appendChild(option);
				document.removeEventListener(this, switchSelected)
				this.parentElement.removeChild(this);
				option.addEventListener("dblclick", switchSelected)
				setSelected(selectors[0].querySelectorAll("option"));
			} 
		}
	}
	
function setSelected(options) {
	options.forEach(function(option) {
		option.selected = true;
	});
}
</script>
`
}
