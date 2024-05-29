package ctx_test

import (
	"testing"

	"github.com/Nigel2392/django/core/ctx"
)

type TestStruct struct {
	Name string
	Age  int
}

type TestStructContextInterface struct {
	Data map[string]interface{}
}

func (t *TestStructContextInterface) Set(key string, value interface{}) {
	t.Data[key] = value
}

func (t *TestStructContextInterface) Get(key string) interface{} {
	return t.Data[key]
}

func TestStructContextGet(t *testing.T) {
	var obj = &TestStruct{Name: "John", Age: 30}
	var context = ctx.NewStructContext(obj)

	var value = context.Get("Name")
	if value != "John" {
		t.Errorf("expected %q, got %q", "John", value)
	}

	value = context.Get("Age")
	if value != 30 {
		t.Errorf("expected %d, got %d", 30, value)
	}
}

func TestStructContextSet(t *testing.T) {
	var obj = &TestStruct{Name: "John", Age: 30}
	var context = ctx.NewStructContext(obj)

	context.Set("Name", "Jane")
	context.Set("Age", 25)

	if obj.Name != "Jane" {
		t.Errorf("expected %q, got %q", "Jane", obj.Name)
	}

	if obj.Age != 25 {
		t.Errorf("expected %d, got %d", 25, obj.Age)
	}
}

func TestStructContextMapSetGet(t *testing.T) {
	var obj = &TestStruct{Name: "John", Age: 30}
	var context = ctx.NewStructContext(obj)

	context.Set("ObjName", "Jane")
	context.Set("ObjAge", 25)

	if context.Get("ObjName") != "Jane" {
		t.Errorf("expected %q, got %q", "Jane", context.Get("ObjName"))
	}

	if context.Get("ObjAge") != 25 {
		t.Errorf("expected %d, got %d", 25, context.Get("ObjAge"))
	}

	if obj.Name != "John" {
		t.Errorf("expected %q, got %q", "John", obj.Name)
	}

	if obj.Age != 30 {
		t.Errorf("expected %d, got %d", 30, obj.Age)
	}
}

func TestStructContextInterfaceGet(t *testing.T) {
	var context = ctx.NewStructContext(
		&TestStructContextInterface{Data: map[string]interface{}{
			"Name": "John",
			"Age":  30,
		}},
	)

	if context.Get("Name") != "John" {
		t.Errorf("expected %q, got %q", "Jane", context.Get("Name"))
	}

	if context.Get("Age") != 30 {
		t.Errorf("expected %d, got %d", 25, context.Get("Age"))
	}
}

func TestStructContextInterfaceSet(t *testing.T) {

	var m = make(map[string]interface{})
	m["Name"] = "John"
	m["Age"] = 30

	var context = ctx.NewStructContext(
		&TestStructContextInterface{m},
	)

	context.Set("Name", "Jane")
	context.Set("Age", 25)

	if context.Get("Name") != "Jane" {
		t.Errorf("expected %v, got %v %T", "Jane", context.Get("Name"), context.Get("Name"))
	}

	if context.Get("Age") != 25 {
		t.Errorf("expected %v, got %v %T", 25, context.Get("Age"), context.Get("Age"))
	}

	var obj = context.(*ctx.StructContext).Object().(*TestStructContextInterface)
	if obj.Data["Name"] != "Jane" {
		t.Errorf("expected %v, got %v %T", "Jane", obj.Data["Name"], obj.Data["Name"])
	}

	if obj.Data["Age"] != 25 {
		t.Errorf("expected %v, got %v %T", 25, obj.Data["Age"], obj.Data["Age"])
	}
}
