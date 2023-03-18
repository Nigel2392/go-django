package forms

import (
	"fmt"
	"html/template"
	"strings"
)

func (f *Form) Render() template.HTML {
	var formBuilder strings.Builder
	for _, field := range f.Fields {
		field.Render(&formBuilder)
	}
	return template.HTML(formBuilder.String())
}

func (f *FormField) Render(fieldBuilder *strings.Builder) {
	if f.Type == "m2m" {
		f.renderM2M(fieldBuilder)
		return
	}
	labelsClasses := strings.Join(f.LabelClasses, " ")
	divClasses := strings.Join(f.DivClasses, " ")
	if divClasses != "" {
		fmt.Fprintf(fieldBuilder, `<div class="%s">`, divClasses)
	} else {
		fmt.Fprintf(fieldBuilder, `<div>`)
	}
	if labelsClasses != "" {
		fmt.Fprintf(fieldBuilder, `<label for="form_%s" class="%s">%s</label>`, f.Name, labelsClasses, f.Label)
	} else {
		fmt.Fprintf(fieldBuilder, `<label for="form_%s">%s</label>`, f.Name, f.Label)
	}
	switch f.Type {
	case "checkbox", "datetime":
		f.renderInput(fieldBuilder)
	case "duration":
		f.Type = "text"
		f.renderInput(fieldBuilder)
	case "select", "fk":
		f.renderSelect(fieldBuilder)
	case "textarea":
		f.renderTextarea(fieldBuilder)
	default:
		f.renderInput(fieldBuilder)
	}
	fieldBuilder.WriteString(`</div>`)
}

func (f *FormField) renderM2M(fieldBuilder *strings.Builder) {
	if len(f.Options) != 2 {
		panic("m2m field must have 2 options.")
	}
	// Write container
	var unselected, selected *FormField
	selected = f.Options[0]
	selected.Multiple = true
	unselected = f.Options[1]
	unselected.Multiple = true

	fmt.Fprintf(fieldBuilder, `<div class="m2m-container">`)

	// Write selected
	fmt.Fprintf(fieldBuilder, `<div class="m2m-select">`)
	fmt.Fprintf(fieldBuilder, `<h3>`)
	fmt.Fprintf(fieldBuilder, `Selected `)
	fmt.Fprintf(fieldBuilder, f.Name)
	fmt.Fprintf(fieldBuilder, `</h3>`)
	fmt.Fprintf(fieldBuilder, `<input type="hidden" name="form_%s" value="-">`, f.Name)
	selected.renderSelect(fieldBuilder)
	fmt.Fprintf(fieldBuilder, `</div>`)

	// Write buttons
	fmt.Fprintf(fieldBuilder, `<div class="m2m-buttons">`)
	fmt.Fprintf(fieldBuilder, `<button type="button" class="m2m-add-all add">Add All</button>`)
	fmt.Fprintf(fieldBuilder, `<button type="button" class="m2m-remove-all remove">Remove All</button>`)
	fmt.Fprintf(fieldBuilder, `</div>`)

	// Write unselected
	fmt.Fprintf(fieldBuilder, `<div class="m2m-select">`)
	fmt.Fprintf(fieldBuilder, `<h3>Other</h3>`)
	unselected.renderSelect(fieldBuilder)
	fmt.Fprintf(fieldBuilder, `</div>`)

	fmt.Fprintf(fieldBuilder, `</div>`)
}

func (f *FormField) renderInput(fieldBuilder *strings.Builder) {
	var extra = f.getExtra()
	fmt.Fprintf(fieldBuilder, `<input type="%s" id="form_%s" name="form_%s" %s>`, f.Type, f.Name, f.Name, extra)
}

func (f *FormField) renderSelect(fieldBuilder *strings.Builder) {
	var extra = f.getExtra()
	fmt.Fprintf(fieldBuilder, `<select id="form_%s" name="form_%s" %s>`, f.Name, f.Name, extra)
	for _, option := range f.Options {
		var extra string
		if option.Selected {
			extra += " selected"
		}
		fmt.Fprintf(fieldBuilder, `<option value="%s" %s>%s</option>`, option.Value, extra, option.Label)
	}
	fmt.Fprintf(fieldBuilder, `</select>`)
}

func (f *FormField) renderTextarea(fieldBuilder *strings.Builder) {
	var extra = f.getExtra()
	fmt.Fprintf(fieldBuilder, `<textarea id="form_%s" name="form_%s" %s>%s</textarea>`, f.Name, f.Name, extra, f.Value)
}

func (f *FormField) getExtra() string {
	var extra string
	if f.Multiple {
		extra = " multiple"
	}
	if f.Required {
		extra += " required"
	}
	if (f.ReadOnly && !f.readonlyfull) || (f.NeedsAdmin && !f.isAdmin) {
		extra += " readonly"
	} else if f.readonlyfull {
		if f.Value != "" {
			extra += " readonly"
		}
	}
	if (f.Disabled && !f.disabledfull) || (f.NeedsAdmin && !f.isAdmin) {
		extra += " disabled"
	} else if f.disabledfull {
		if f.Value != "" {
			extra += " disabled"
		}
	}
	if f.Checked {
		extra += " checked"
	}
	if f.Selected {
		extra += " selected"
	}
	if f.Autocomplete != "" {
		extra += fmt.Sprintf(` autocomplete="%s"`, f.Autocomplete)
	}
	if f.Value != "" {
		extra += fmt.Sprintf(` value="%s"`, f.Value)
	}
	if f.Custom != "" {
		extra += " " + f.Custom
	}
	if len(f.Classes) > 0 {
		extra += fmt.Sprintf(` class="%s"`, strings.Join(f.Classes, " "))
	}
	return extra
}
