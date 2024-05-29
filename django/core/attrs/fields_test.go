package attrs_test

import (
	"testing"

	"github.com/Nigel2392/django/core/attrs"
)

type TestModelFields struct {
	ID      int
	Name    string
	Objects []int64
}

func TestModelFieldsGet(t *testing.T) {
	var m = &TestModelDefinitions{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", false, true)
		defName    = attrs.NewField(m, "Name", false, true)
		defObjects = attrs.NewField(m, "Objects", false, true)
	)

	if m.ID != defID.GetValue().(int) {
		t.Errorf("expected %d, got %d", m.ID, defID.GetValue())
	}

	if m.Name != defName.GetValue().(string) {
		t.Errorf("expected %q, got %q", m.Name, defName.GetValue())
	}

	if len(m.Objects) != len(defObjects.GetValue().([]int64)) {
		t.Errorf("expected %d, got %d", len(m.Objects), len(defObjects.GetValue().([]int64)))
	}
}

func TestModelFieldFieldsSet(t *testing.T) {
	var m = &TestModelDefinitions{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", false, true)
		defName    = attrs.NewField(m, "Name", false, true)
		defObjects = attrs.NewField(m, "Objects", false, true)
	)

	defID.SetValue(2, false)
	defName.SetValue("new name", false)
	defObjects.SetValue([]int64{4, 5, 6}, false)

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

func TestModelFieldFieldsSetReadOnly(t *testing.T) {
	var m = &TestModelFields{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", false, true)
		defName    = attrs.NewField(m, "Name", false, true)
		defObjects = attrs.NewField(m, "Objects", false, false)
	)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}

		if m.Objects[0] != 1 {
			t.Errorf("expected %d, got %d", 1, m.Objects[0])
		}

		if m.ID != 2 {
			t.Errorf("expected %d, got %d", 1, m.ID)
		}

		if m.Name != "new name" {
			t.Errorf("expected %q, got %q", "name", m.Name)
		}
	}()

	defID.SetValue(2, false)
	defName.SetValue("new name", false)
	defObjects.SetValue([]int64{4, 5, 6}, false)
}

func TestModelFieldFieldsForceSetReadOnly(t *testing.T) {
	var m = &TestModelFields{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", false, true)
		defName    = attrs.NewField(m, "Name", false, true)
		defObjects = attrs.NewField(m, "Objects", false, false)
	)

	defID.SetValue(2, true)
	defName.SetValue("new name", true)
	defObjects.SetValue([]int64{4, 5, 6}, true)

	if m.ID != 2 {
		t.Errorf("expected %d, got %d", 2, m.ID)
	}

	if m.Name != "new name" {
		t.Errorf("expected %q, got %q", "new name", m.Name)
	}

	if m.Objects[0] != 4 {
		t.Errorf("expected %d, got %d", 4, m.Objects[0])
	}
}
