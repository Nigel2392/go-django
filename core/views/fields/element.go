package fields

import (
	"html/template"
	"strings"

	"github.com/Nigel2392/go-django/core/httputils/tags"
	"github.com/Nigel2392/go-django/core/views/interfaces"
)

var (
	_ interfaces.FormField = (*IntField)(nil)
	_ interfaces.FormField = (*FloatField)(nil)
	_ interfaces.FormField = (*StringField)(nil)
	_ interfaces.FormField = (*TextField)(nil)
	_ interfaces.FormField = (*BoolField)(nil)
	_ interfaces.FormField = (*DateField)(nil)
	_ interfaces.FormField = (*DateTimeField)(nil)
	_ interfaces.FormField = (*SelectField)(nil)
	_ interfaces.FormField = (*CheckBoxField)(nil)

	_ interfaces.Field = (*IntField)(nil)
	_ interfaces.Field = (*FloatField)(nil)
	_ interfaces.Field = (*StringField)(nil)
	_ interfaces.Field = (*TextField)(nil)
	_ interfaces.Field = (*BoolField)(nil)
	_ interfaces.Field = (*DateField)(nil)
	_ interfaces.Field = (*DateTimeField)(nil)
	_ interfaces.Field = (*SelectField)(nil)
	_ interfaces.Field = (*CheckBoxField)(nil)
)

type ElementType string

func (e ElementType) String() string {
	return string(e)
}

func (e ElementType) HTML() template.HTML {
	return template.HTML(e)
}

// excludes the name, selected, value, type
var AllTagsInput = []string{
	"class",
	"id",
	"style",
	"placeholder",
	"rows",
	"cols",
	"selected",
	"multiple",
	"disabled",
	"readonly",
	"required",
	"autofocus",
	"autocomplete",
	"min",
	"max",
	"step",
}

var AllTagsLabel = []string{
	"labelclass",
	"labelid",
	"labelstyle",
}

func TagMapToElementAttributes(t tags.TagMap, fields ...string) string {
	var b strings.Builder
	for i, field := range fields {
		switch strings.ToLower(field) {
		case "labelclass":
			var classes, ok = t["labelclass"]
			writeIfOK(&b, ok, ` class="`, strings.Join(classes, " "), `"`)
		case "labelid":
			var id = t.GetSingle("labelid")
			writeIfOK(&b, id != "", ` id="`, id, `"`)
		case "labelstyle":
			var style, ok = t["labelstyle"]
			writeIfOK(&b, ok, ` style="`, strings.Join(style, ";"), `"`)
		case "class":
			var classes, ok = t["class"]
			writeIfOK(&b, ok, ` class="`, strings.Join(classes, " "), `"`)
		case "id":
			var id = t.GetSingle("id")
			writeIfOK(&b, id != "", ` id="`, id, `"`)
		case "style":
			var style, ok = t["style"]
			writeIfOK(&b, ok, ` style="`, strings.Join(style, ";"), `"`)
		case "name":
			var name = t.GetSingle("name")
			writeIfOK(&b, name != "", ` name="`, name, `"`)
		case "value":
			var value = t.GetSingle("value")
			writeIfOK(&b, value != "", ` value="`, value, `"`)
		case "type":
			var typ = t.GetSingle("type")
			writeIfOK(&b, typ != "", ` type="`, typ, `"`)
		case "placeholder":
			var placeholder = t.GetSingle("placeholder")
			writeIfOK(&b, placeholder != "", ` placeholder="`, placeholder, `"`)
		case "rows":
			var rows = t.GetSingle("rows")
			writeIfOK(&b, rows != "", ` rows="`, rows, `"`)
		case "cols":
			var cols = t.GetSingle("cols")
			writeIfOK(&b, cols != "", ` cols="`, cols, `"`)
		case "checked":
			var checked = t.Exists("checked")
			writeIfOK(&b, checked, `checked`)
		case "selected":
			var selected = t.Exists("selected")
			writeIfOK(&b, selected, `selected`)
		case "multiple":
			var multiple = t.Exists("multiple")
			writeIfOK(&b, multiple, `multiple`)
		case "disabled":
			var disabled = t.Exists("disabled")
			writeIfOK(&b, disabled, `disabled`)
		case "readonly":
			var readonly = t.Exists("readonly")
			writeIfOK(&b, readonly, `readonly`)
		case "required":
			var required = t.Exists("required")
			writeIfOK(&b, required, `required`)
		case "autofocus":
			var autofocus = t.Exists("autofocus")
			writeIfOK(&b, autofocus, `autofocus`)
		case "autocomplete":
			var autocomplete = t.GetSingle("autocomplete")
			writeIfOK(&b, autocomplete != "", ` autocomplete="`, autocomplete, `"`)
		case "min":
			var min = t.GetSingle("min")
			writeIfOK(&b, min != "", ` min="`, min, `"`)
		case "max":
			var max = t.GetSingle("max")
			writeIfOK(&b, max != "", ` max="`, max, `"`)
		case "step":
			var step = t.GetSingle("step")
			writeIfOK(&b, step != "", ` step="`, step, `"`)
		}
		if i < len(fields)-1 && b.Len() > 0 {
			b.WriteString(" ")
		}
	}
	return b.String()
}

func writeIfOK(b *strings.Builder, ok bool, s ...string) {
	var l = 0
	for _, str := range s {
		l += len(str)
	}
	if l == 0 {
		return
	}
	b.Grow(l)
	if ok {
		for _, str := range s {
			b.WriteString(str)
		}
	}
}
