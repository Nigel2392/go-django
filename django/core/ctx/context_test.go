package ctx_test

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/Nigel2392/django/core/ctx"
)

type context_struct struct {
	data map[string]any
}

type iface_struct struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

func getInterfaceValue(i ctx.Context) *context_struct {
	var iface = (*iface_struct)(unsafe.Pointer(&i))
	return (*context_struct)(iface.data)
}

func TestGetInterfaceValue(t *testing.T) {
	var c = ctx.NewContext(map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})
	var data = getInterfaceValue(c)
	if data.data == nil {
		t.Errorf("expected not nil")
	}

	if data.data["key1"] != "value1" {
		t.Errorf("expected %q, got %q", "value1", data.data["key1"])
	}

	if data.data["key2"] != "value2" {
		t.Errorf("expected %q, got %q", "value2", data.data["key2"])
	}

	if data.data["key3"] != "value3" {
		t.Errorf("expected %q, got %q", "value3", data.data["key3"])
	}
}

func TestContextSet(t *testing.T) {
	var c = ctx.NewContext(nil)

	c.Set("key", "value")

	var data = getInterfaceValue(c)
	if data.data["key"] != "value" {
		t.Errorf("expected %q, got %q", "value", data.data["key"])
	}
}

func TestContextGet(t *testing.T) {
	var c = ctx.NewContext(map[string]any{"key": "value"})
	var value = c.Get("key")
	if value != "value" {
		t.Errorf("expected %q, got %q", "value", value)
	}
}

type context_editor struct {
	set interface{}
}

func (c *context_editor) EditContext(key string, context ctx.Context) {
	context.Set(key, c.set)
	context.Set(fmt.Sprintf("%s_key", key), c.set)
}

func TestContextEditContext(t *testing.T) {
	var c = ctx.NewContext(nil)
	var e = &context_editor{set: "value"}

	c.Set("object_key", e)

	var data = getInterfaceValue(c)
	if data.data["object_key"] != "value" {
		t.Errorf("expected %q, got %q", "value", data.data["key"])
	}

	if data.data["object_key_key"] != "value" {
		t.Errorf("expected %q, got %q", "value", data.data["object_key_key"])
	}
}
