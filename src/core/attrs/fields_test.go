package attrs_test

import (
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/goldcrest"
)

type TestModelFields struct {
	ID      int
	Name    string
	Objects []int64
}

type customTestWidget struct {
	*widgets.BaseWidget
}

func (f *TestModelFields) FieldDefs() attrs.Definitions {
	return attrs.Define(f,
		attrs.NewField(f, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(f, "Name", nil),
		attrs.NewField(f, "Objects", &attrs.FieldConfig{ReadOnly: true}),
	)
}

type TestEmbeddedModelFields struct {
	ID   int
	Name string
	Test *TestModelFields
}

func (f *TestEmbeddedModelFields) FieldDefs() attrs.Definitions {
	return attrs.Define(f,
		attrs.NewField(f, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(f, "Name", nil),
		attrs.NewField(f, "Test", nil),
	)
}

func init() {

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &TestModelFields{},
		GetInstance: func(i interface{}) (interface{}, error) {
			var id = i.(int)
			return &TestModelFields{
				ID:      int(id),
				Name:    "name",
				Objects: []int64{4, 5, 6},
			}, nil
		},
	})

}

func TestModelFieldsGet(t *testing.T) {
	var m = &TestModelDefinitions{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", nil)
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
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", nil)
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
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", &attrs.FieldConfig{ReadOnly: true})
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
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", &attrs.FieldConfig{ReadOnly: true})
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

func TestModelFieldsScannable(t *testing.T) {
	var m = &TestModelFields{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", nil)
	)

	defID.Scan(uint64(2))
	defName.Scan("new name")
	defObjects.Scan([]int64{4, 5, 6})

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

	if m.Objects[1] != 5 {
		t.Errorf("expected %d, got %d", 5, m.Objects[1])
	}

	if m.Objects[2] != 6 {
		t.Errorf("expected %d, got %d", 6, m.Objects[2])
	}

	var testEmbeddedModelFields = &TestEmbeddedModelFields{
		ID:   1,
		Name: "name",
	}

	var (
		defTestID   = attrs.NewField(testEmbeddedModelFields, "ID", nil)
		defTestName = attrs.NewField(testEmbeddedModelFields, "Name", nil)
		defTest     = attrs.NewField(testEmbeddedModelFields, "Test", nil)
	)

	defTestID.Scan(uint64(2))
	defTestName.Scan("new name")
	defTest.Scan(2)

	if testEmbeddedModelFields.ID != 2 {
		t.Errorf("expected %d, got %d", 2, testEmbeddedModelFields.ID)
	}

	if testEmbeddedModelFields.Name != "new name" {
		t.Errorf("expected %q, got %q", "new name", testEmbeddedModelFields.Name)
	}

	if testEmbeddedModelFields.Test.ID != 2 {
		t.Errorf("expected %d, got %d", 2, testEmbeddedModelFields.Test.ID)
	}

	//	if testEmbeddedModelFields.Test.Name != "name" {
	//		t.Errorf("expected %q, got %q", "name", testEmbeddedModelFields.Test.Name)
	//	}
	//
	//	if len(testEmbeddedModelFields.Test.Objects) != 3 {
	//		t.Errorf("expected %d, got %d", 3, len(testEmbeddedModelFields.Test.Objects))
	//	}
	//
	//	if testEmbeddedModelFields.Test.Objects[0] != 4 {
	//		t.Errorf("expected %d, got %d", 1, testEmbeddedModelFields.Test.Objects[0])
	//	}
	//
	//	if testEmbeddedModelFields.Test.Objects[1] != 5 {
	//		t.Errorf("expected %d, got %d", 2, testEmbeddedModelFields.Test.Objects[1])
	//	}
	//
	//	if testEmbeddedModelFields.Test.Objects[2] != 6 {
	//		t.Errorf("expected %d, got %d", 3, testEmbeddedModelFields.Test.Objects[2])
	//	}
}

func TestModelFieldsValuer(t *testing.T) {
	var m = &TestEmbeddedModelFields{
		ID:   1,
		Name: "name",
		Test: &TestModelFields{ID: 1, Name: "name", Objects: []int64{1, 2, 3}},
	}

	var (
		defID   = attrs.NewField(m, "ID", nil)
		defName = attrs.NewField(m, "Name", nil)
		defTest = attrs.NewField(m, "Test", nil)
	)

	var v any
	var err error

	v, err = defID.Value()
	if err != nil {
		t.Errorf("expected %v, got %v", nil, err)
	}

	if v.(int) != 1 {
		t.Errorf("expected %d, got %d", 1, v.(int))
	}

	v, err = defName.Value()
	if err != nil {
		t.Errorf("expected %v, got %v", nil, err)
	}

	if v.(string) != "name" {
		t.Errorf("expected %q, got %q", "name", v.(string))
	}

	v, err = defTest.Value()
	if err != nil {
		t.Errorf("expected %v, got %v", nil, err)
	}

	if v == nil {
		t.Errorf("expected %v, got %v", nil, v)
	}

	if v.(uint64) != 1 {
		t.Errorf("expected %d, got %d", 1, v.(int))
	}

}

func TestModelFormFields(t *testing.T) {
	var m = &TestModelFields{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", nil)
	)

	var (
		formfieldID      = defID.FormField()
		formfieldName    = defName.FormField()
		formfieldObjects = defObjects.FormField()
	)

	if v, ok := formfieldID.(*fields.BaseField); !ok {
		t.Errorf("expected %t, got %t", true, ok)
	} else {
		if v.Name() != "ID" {
			t.Errorf("expected %q, got %q", "ID", v.Name())
		}

		if _, ok := v.Widget().(*widgets.NumberWidget[int]); !ok {
			t.Errorf("expected %t, got %t", true, ok)
		}
	}

	if v, ok := formfieldName.(*fields.BaseField); !ok {
		t.Errorf("expected %t, got %t", true, ok)
	} else {
		if v.Name() != "Name" {
			t.Errorf("expected %q, got %q", "Name", v.Name())
		}

		if _, ok := v.Widget().(*widgets.BaseWidget); !ok {
			t.Errorf("expected %t, got %t", true, ok)
		}
	}

	if v, ok := formfieldObjects.(*fields.BaseField); !ok {
		t.Errorf("expected %t, got %t", true, ok)
	} else {
		if v.Name() != "Objects" {
			t.Errorf("expected %q, got %q", "Objects", v.Name())
		}

		if _, ok := v.Widget().(*widgets.BaseWidget); !ok {
			t.Errorf("expected %t, got %t", true, ok)
		}
	}
}

func TestModelFormFieldsCustomType(t *testing.T) {
	var m = &TestModelFields{
		ID:      1,
		Name:    "name",
		Objects: []int64{1, 2, 3},
	}

	var (
		defID      = attrs.NewField(m, "ID", nil)
		defName    = attrs.NewField(m, "Name", nil)
		defObjects = attrs.NewField(m, "Objects", nil)
	)

	goldcrest.Register(
		attrs.HookFormFieldForType,
		0, attrs.FormFieldGetter(func(f attrs.Field, t reflect.Type, v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
			if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Int64 {
				var newF = fields.JSONField[[]int64](opts...)
				newF.FormWidget = &customTestWidget{widgets.NewBaseWidget(
					"custom", "", nil,
				)}
				return newF, true
			}
			return nil, false
		}),
	)

	var (
		formfieldID      = defID.FormField()
		formfieldName    = defName.FormField()
		formfieldObjects = defObjects.FormField()
	)

	if v, ok := formfieldID.(*fields.BaseField); !ok {
		t.Errorf("expected %t, got %t", true, ok)
	} else {
		if v.Name() != "ID" {
			t.Errorf("expected %q, got %q", "ID", v.Name())
		}

		if _, ok := v.Widget().(*widgets.NumberWidget[int]); !ok {
			t.Errorf("expected %t, got %t", true, ok)
		}
	}

	if v, ok := formfieldName.(*fields.BaseField); !ok {
		t.Errorf("expected %t, got %t", true, ok)
	} else {
		if v.Name() != "Name" {
			t.Errorf("expected %q, got %q", "Name", v.Name())
		}

		if _, ok := v.Widget().(*widgets.BaseWidget); !ok {
			t.Errorf("expected %t, got %t", true, ok)
		}
	}

	if v, ok := formfieldObjects.(*fields.JSONFormField[[]int64]); !ok {
		t.Errorf("expected %t, got %t", true, ok)
	} else {
		if v.Name() != "Objects" {
			t.Errorf("expected %q, got %q", "Objects", v.Name())
		}

		if _, ok := v.Widget().(*customTestWidget); !ok {
			t.Errorf("expected %t, got %t (%T)", true, ok, v.Widget())
		}
	}

	goldcrest.Unregister(
		attrs.HookFormFieldForType,
	)
}
