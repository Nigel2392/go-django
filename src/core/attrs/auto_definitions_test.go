package attrs_test

import (
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
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
	if fieldDefs.ObjectFields.Len() != 3 {
		t.Errorf("expected %d, got %d", 3, fieldDefs.ObjectFields.Len())
	}

	if _, ok := fieldDefs.ObjectFields.Get("ID"); !ok {
		t.Errorf("expected field %q", "ID")
	}

	if _, ok := fieldDefs.ObjectFields.Get("Name"); !ok {
		t.Errorf("expected field %q", "Name")
	}

	if _, ok := fieldDefs.ObjectFields.Get("Objects"); !ok {
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
	if fieldDefs.ObjectFields.Len() != 1 {
		t.Errorf("expected %d, got %d", 1, fieldDefs.ObjectFields.Len())
	}

	if _, ok := fieldDefs.ObjectFields.Get("Name"); !ok {
		t.Errorf("expected field %q", "Name")
	}

	if _, ok := fieldDefs.ObjectFields.Get("ID"); ok {
		t.Errorf("unexpected field %q", "ID")
	}

	if _, ok := fieldDefs.ObjectFields.Get("Objects"); ok {
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

type PrimaryFieldTester struct {
	ID int `attrs:"primary;default=2"`
	I  int `attrs:"default=1"`
}

func (f *PrimaryFieldTester) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f)
}

func TestModelAutoFieldDefinitionsPrimary(t *testing.T) {
	var m = &PrimaryFieldTester{}

	var fieldDefs = m.FieldDefs().(*attrs.ObjectDefinitions)
	if fieldDefs.ObjectFields.Len() != 2 {
		t.Errorf("expected %d, got %d", 2, fieldDefs.ObjectFields.Len())
	}

	var primary = fieldDefs.Primary()

	if primary.Name() != "ID" {
		t.Errorf("expected %q, got %q", "ID", fieldDefs.Primary().Name())
	}

	if primary.GetDefault().(int) != 2 {
		t.Errorf("expected %d, got %d", 1, fieldDefs.Primary().GetDefault())
	}

	if primary.GetValue().(int) != 0 {
		t.Errorf("(GetValue())expected %d, got %d", 2, fieldDefs.Primary().GetValue())
	}

	var v, err = primary.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if v.(int) != 2 {
		t.Errorf("(Value()) expected %d, got %d", 2, v.(int))
	}

}

type DefaultsTester struct {
	Bool      bool     `attrs:"default=true"`
	Int       int      `attrs:"default=2"`
	Uint      uint     `attrs:"default=4"`
	Float     float64  `attrs:"default=6.0"`
	String    string   `attrs:"default=hello"`
	BoolPtr   *bool    `attrs:"default=true"`
	IntPtr    *int     `attrs:"default=8"`
	UintPtr   *uint    `attrs:"default=10"`
	FloatPtr  *float64 `attrs:"default=12.0"`
	StringPtr *string  `attrs:"default=hello"`
	Int16     int16    `attrs:"default=14"`
	Uint16    uint16   `attrs:"default=16"`
	Float32   float32  `attrs:"default=18.0"`
}

func (f *DefaultsTester) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f)
}

func TestModelAutoFieldDefinitionsDefaults(t *testing.T) {
	var m = &DefaultsTester{}
	var fieldDefs = m.FieldDefs().(*attrs.ObjectDefinitions)

	if fieldDefs.ObjectFields.Len() != 13 {
		t.Errorf("expected %d, got %d", 13, fieldDefs.ObjectFields.Len())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Bool"); !ok {
		t.Errorf("expected field %q", "Bool")
	} else if val.GetDefault().(bool) != true {
		t.Errorf("expected %v, got %v", true, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Int"); !ok {
		t.Errorf("expected field %q", "Int")
	} else if val.GetDefault().(int) != 2 {
		t.Errorf("expected %d, got %d", 2, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Uint"); !ok {
		t.Errorf("expected field %q", "Uint")
	} else if val.GetDefault().(uint) != 4 {
		t.Errorf("expected %d, got %d", 4, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Float"); !ok {
		t.Errorf("expected field %q", "Float")
	} else if val.GetDefault().(float64) != 6.0 {
		t.Errorf("expected %f, got %f", 6.0, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("String"); !ok {
		t.Errorf("expected field %q", "String")
	} else if val.GetDefault().(string) != "hello" {
		t.Errorf("expected %q, got %q", "hello", val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("BoolPtr"); !ok {
		t.Errorf("expected field %q", "BoolPtr")
	} else if *(val.GetDefault().(*bool)) != true {
		t.Errorf("expected %v, got %v", true, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("IntPtr"); !ok {
		t.Errorf("expected field %q", "IntPtr")
	} else if *(val.GetDefault().(*int)) != 8 {
		t.Errorf("expected %d, got %d", 8, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("UintPtr"); !ok {
		t.Errorf("expected field %q", "UintPtr")
	} else if *(val.GetDefault().(*uint)) != 10 {
		t.Errorf("expected %d, got %d", 10, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("FloatPtr"); !ok {
		t.Errorf("expected field %q", "FloatPtr")
	} else if *(val.GetDefault().(*float64)) != 12.0 {
		t.Errorf("expected %f, got %f", 12.0, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("StringPtr"); !ok {
		t.Errorf("expected field %q", "StringPtr")
	} else if *(val.GetDefault().(*string)) != "hello" {
		t.Errorf("expected %q, got %q", "hello", val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Int16"); !ok {
		t.Errorf("expected field %q", "Int16")
	} else if val.GetDefault().(int16) != 14 {
		t.Errorf("expected %d, got %d", 14, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Uint16"); !ok {
		t.Errorf("expected field %q", "Uint16")
	} else if val.GetDefault().(uint16) != 16 {
		t.Errorf("expected %d, got %d", 16, val.GetDefault())
	}

	if val, ok := fieldDefs.ObjectFields.Get("Float32"); !ok {
		t.Errorf("expected field %q", "Float32")
	} else if val.GetDefault().(float32) != 18.0 {
		t.Errorf("expected %f, got %f", 18.0, val.GetDefault())
	}
}
