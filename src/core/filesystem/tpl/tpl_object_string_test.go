package tpl_test

import (
	"bytes"
	"testing"

	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

const templateString = `{{define "test"}}Hello, {{ block "name" . }}{{.Name}}{{end}}!{{end}}`

func TestStringTemplateObject_Execute(t *testing.T) {
	tmpl := tpl.GetTemplateFromString("test", templateString, nil)
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string]interface{}{
		"Name": "World",
	})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "Hello, World!" {
		t.Errorf("unexpected output: %q", buf.String())
	}

	t.Logf("Template executed successfully: %s", buf.String())
}

func TestStringTemplateObject_Execute_Subtemplate(t *testing.T) {
	tmpl := tpl.GetTemplateFromString("name", templateString, nil)
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string]interface{}{
		"Name": "World",
	})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "World" {
		t.Errorf("unexpected output: %q", buf.String())
	}

	t.Logf("Subtemplate executed successfully: %s", buf.String())
}
