package models_test

import (
	"testing"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	attrs.RegisterModel(&TestModel{})
}

type TestModel struct {
	models.Model
	Title       string
	Description string
}

func (m *TestModel) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.NewField(m, "Title", nil),
		attrs.NewField(m, "Description", nil),
	)
}

func TestModelFields(t *testing.T) {
	var m = &TestModel{
		Title:       "Test",
		Description: "Test description",
	}

	var (
		defs  = m.FieldDefs()
		title = defs.Get("Title")
		desc  = defs.Get("Description")
	)

	if title != "Test" {
		t.Errorf("Expected Title to be 'Test', got '%s'", title)
	}

	if desc != "Test description" {
		t.Errorf("Expected Description to be 'Test description', got '%s'", desc)
	}

	m.Title = "Updated Title"
	m.Description = "Updated description"

	title = defs.Get("Title")
	desc = defs.Get("Description")

	if title != "Updated Title" {
		t.Errorf("Expected Title to be 'Updated Title', got '%s'", title)
	}

	if desc != "Updated description" {
		t.Errorf("Expected Description to be 'Updated description', got '%s'", desc)
	}

	t.Logf("TestModel: %+v", m)
}

func TestModelFieldsSetValue(t *testing.T) {
	var m = &TestModel{
		Title:       "Test",
		Description: "Test description",
	}

	var (
		defs  = m.FieldDefs()
		title = defs.Get("Title")
		desc  = defs.Get("Description")
	)

	if title != "Test" {
		t.Errorf("Expected Title to be 'Test', got '%s'", title)
	}

	if desc != "Test description" {
		t.Errorf("Expected Description to be 'Test description', got '%s'", desc)
	}

	var (
		defs2         = m.FieldDefs()
		titleField, _ = defs2.Field("Title")
		descField, _  = defs2.Field("Description")
	)

	titleField.SetValue("Updated Title", true)
	descField.SetValue("Updated description", true)

	title = defs.Get("Title")
	desc = defs.Get("Description")

	if title != "Updated Title" {
		t.Errorf("Expected Title to be 'Updated Title', got '%s'", title)
	}

	if desc != "Updated description" {
		t.Errorf("Expected Description to be 'Updated description', got '%s'", desc)
	}
}

func TestExtractModel(t *testing.T) {
	var m = &TestModel{
		Title:       "Test",
		Description: "Test description",
	}

	var modelObj, err = models.ExtractModel(m)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if modelObj == nil {
		t.Errorf("Expected model object to be not nil")
	}
}
