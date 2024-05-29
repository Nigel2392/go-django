package attrs_test

import (
	"testing"

	"github.com/Nigel2392/django/core/attrs"
)

type TestModelAutoFieldDefs struct {
	ID      int
	Name    string
	Objects []int64
}

func (f *TestModelAutoFieldDefs) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f)
}

func TestModelAutoFieldDefinitions(t *testing.T) {
	var m = &TestModelAutoFieldDefs{}

	var fieldDefs = m.FieldDefs().(*attrs.ObjectDefinitions)
	if len(fieldDefs.ObjectFields) != 3 {
		t.Errorf("expected %d, got %d", 3, len(fieldDefs.ObjectFields))
	}

	if _, ok := fieldDefs.ObjectFields["ID"]; !ok {
		t.Errorf("expected field %q", "ID")
	}

	if _, ok := fieldDefs.ObjectFields["Name"]; !ok {
		t.Errorf("expected field %q", "Name")
	}

	if _, ok := fieldDefs.ObjectFields["Objects"]; !ok {
		t.Errorf("expected field %q", "Objects")
	}
}

func TestModelAutoFieldDefinitionsGet(t *testing.T) {
	var m = &TestModelAutoFieldDefs{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		ID      = attrs.Get[int](m, "ID")
		Name    = attrs.Get[string](m, "Name")
		Objects = attrs.Get[[]int64](m, "Objects")
	)

	if ID != 1 {
		t.Errorf("expected %d, got %d", 1, ID)
	}

	if Name != "name" {
		t.Errorf("expected %q, got %q", "name", Name)
	}

	if len(Objects) != 3 {
		t.Errorf("expected %d, got %d", 3, len(Objects))
	}
}

func TestModelAutoFieldDefinitionsSet(t *testing.T) {
	var m = &TestModelAutoFieldDefs{}

	attrs.Set(m, "ID", 1)
	attrs.Set(m, "Name", "name")
	attrs.Set(m, "Objects", []int64{1, 2, 3})

	if m.ID != 1 {
		t.Errorf("expected %d, got %d", 1, m.ID)
	}

	if m.Name != "name" {
		t.Errorf("expected %q, got %q", "name", m.Name)
	}

	if len(m.Objects) != 3 {
		t.Errorf("expected %d, got %d", 3, len(m.Objects))
	}
}

type TestModelAutoFieldDefsInclude struct {
	TestModelAutoFieldDefs
}

func (f *TestModelAutoFieldDefsInclude) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f, "Name")
}

func TestModelAutoFieldDefinitionsInclude(t *testing.T) {
	var m = &TestModelAutoFieldDefsInclude{}

	var fieldDefs = m.FieldDefs().(*attrs.ObjectDefinitions)
	if len(fieldDefs.ObjectFields) != 1 {
		t.Errorf("expected %d, got %d", 1, len(fieldDefs.ObjectFields))
	}

	if _, ok := fieldDefs.ObjectFields["Name"]; !ok {
		t.Errorf("expected field %q", "Name")
	}

	if _, ok := fieldDefs.ObjectFields["ID"]; ok {
		t.Errorf("unexpected field %q", "ID")
	}

	if _, ok := fieldDefs.ObjectFields["Objects"]; ok {
		t.Errorf("unexpected field %q", "Objects")
	}
}

func TestModelAutoFieldDefinitionsIncludeGet(t *testing.T) {
	var m = &TestModelAutoFieldDefsInclude{
		TestModelAutoFieldDefs: TestModelAutoFieldDefs{
			ID:      1,
			Name:    "name",
			Objects: []int64{1, 2, 3},
		},
	}

	var Name = attrs.Get[string](m, "Name")

	if Name != "name" {
		t.Errorf("expected %q, got %q", "name", Name)
	}
}

func TestModelAutoFieldDefinitionsIncludeGetExcluded(t *testing.T) {
	var m = &TestModelAutoFieldDefsInclude{
		TestModelAutoFieldDefs: TestModelAutoFieldDefs{
			ID:      1,
			Name:    "name",
			Objects: []int64{1, 2, 3},
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	attrs.Get[string](m, "ID")
}

func TestModelAutoFieldDefinitionsIncludeSet(t *testing.T) {
	var m = &TestModelAutoFieldDefsInclude{
		TestModelAutoFieldDefs: TestModelAutoFieldDefs{
			Name: "not-name",
		},
	}

	attrs.Set(m, "Name", "name")

	if m.Name != "name" {
		t.Errorf("expected %q, got %q", "name", m.Name)
	}
}

func TestModelAutoFieldDefinitionsIncludeSetExcluded(t *testing.T) {
	var m = &TestModelAutoFieldDefsInclude{}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	attrs.Set(m, "ID", 1)
}
