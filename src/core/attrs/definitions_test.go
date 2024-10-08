package attrs_test

import (
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type TestModelDefinitions struct {
	ID      int
	Name    string
	Objects []int64
}

func (f *TestModelDefinitions) FieldDefs() attrs.Definitions {
	return attrs.Define(f,
		attrs.NewField(f, "ID", nil),
		attrs.NewField(f, "Name", nil),
		attrs.NewField(f, "Objects", nil),
	)
}

func TestModelFieldDefinitionsGet(t *testing.T) {
	var m = &TestModelDefinitions{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var defs = m.FieldDefs().(*attrs.ObjectDefinitions)

	if m.ID != defs.Get("ID").(int) {
		t.Errorf("expected %d, got %d", m.ID, defs.Get("ID"))
	}

	if m.Name != defs.Get("Name").(string) {
		t.Errorf("expected %q, got %q", m.Name, defs.Get("Name"))
	}

	if len(m.Objects) != len(defs.Get("Objects").([]int64)) {
		t.Errorf("expected %d, got %d", len(m.Objects), len(defs.Get("Objects").([]int64)))
	}
}

func TestModelFieldDefinitionsSet(t *testing.T) {
	var m = &TestModelDefinitions{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var defs = m.FieldDefs().(*attrs.ObjectDefinitions)

	defs.Set("ID", 2)
	defs.Set("Name", "new name")
	defs.Set("Objects", []int64{4, 5, 6})

	if m.ID != 2 {
		t.Errorf("expected %d, got %d", 2, m.ID)
	}

	if m.Name != "new name" {
		t.Errorf("expected %q, got %q", "new name", m.Name)
	}

	if len(m.Objects) != 3 {
		t.Errorf("expected %d, got %d", 3, len(m.Objects))
	}

	if m.Objects[0] != 4 {
		t.Errorf("expected %d, got %d", 4, m.Objects[0])
	}
}
