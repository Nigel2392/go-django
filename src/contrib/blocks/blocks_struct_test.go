package blocks_test

import (
	"context"
	"maps"
	"net/mail"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
)

func NewSimpleStructBlock() *blocks.StructBlock {
	var b = blocks.NewStructBlock()

	b.AddField("name", blocks.CharBlock())
	b.AddField("age", blocks.NumberBlock())
	b.AddField("email", blocks.EmailBlock())
	b.AddField("password", blocks.PasswordBlock())
	b.AddField("date", blocks.DateBlock())
	b.AddField("datetime", blocks.DateTimeBlock())

	return b
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

var structBlockDataRaw = url.Values{
	"test_block-name":     {"John Doe"},
	"test_block-age":      {"30"},
	"test_block-email":    {"test@localhost"},
	"test_block-password": {"password"},
	"test_block-date":     {"2021-01-01"},
	"test_block-datetime": {"2021-01-01T00:00:00"},
}

var structBlockstructBlockDataRawCmp = &blocks.StructBlockValue{
	V: map[string]interface{}{
		"name":     "John Doe",
		"age":      "30",
		"email":    "test@localhost",
		"password": "password",
		"date":     "2021-01-01",
		"datetime": "2021-01-01T00:00:00",
	},
}

var structBlockDataGo = &blocks.StructBlockValue{
	V: map[string]interface{}{
		"name":     "John Doe",
		"age":      30,
		"email":    must(mail.ParseAddress("test@localhost")),
		"password": "password",
		"date":     must(time.Parse("2006-01-02", "2021-01-01")),
		"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-01T00:00:00")),
	},
}

func TestStructBlock_ValueAtPath(t *testing.T) {
	sb := blocks.NewStructBlock()
	sb.SetName("person")
	sb.AddField("name", blocks.CharBlock())
	sb.AddField("age", blocks.NumberBlock())

	// Build a *StructBlockValue with Bound children by using ValueFromDB.
	raw := map[string]any{
		"name": "Ada",
		"age":  37,
	}
	boundAny, err := sb.ValueFromDB(jraw(t, raw))
	if err != nil {
		t.Fatalf("ValueFromDB: %v", err)
	}
	bound := boundAny.(blocks.BoundBlockValue)

	t.Run("OK_SingleLevel", func(t *testing.T) {
		got, err := sb.ValueAtPath(bound, []string{"name"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.(string) != "Ada" {
			t.Fatalf("want Ada, got %#v", got)
		}
	})

	t.Run("OK_NestedStruct", func(t *testing.T) {
		outer := blocks.NewStructBlock()
		outer.SetName("outer")
		outer.AddField("person", sb)

		rawNested := map[string]any{
			"person": raw, // reuse above
		}
		bAny, err := outer.ValueFromDB(jraw(t, rawNested))
		if err != nil {
			t.Fatalf("ValueFromDB nested: %v", err)
		}

		got, err := outer.ValueAtPath(bAny.(blocks.BoundBlockValue), []string{"person", "age"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.(int) != 37 {
			t.Fatalf("want 37, got %#v", got)
		}
	})

	t.Run("Err_TypeMismatch", func(t *testing.T) {
		// Pass a FieldBlockValue as the "bound" to StructBlock.ValueAtPath.
		wrong := &blocks.FieldBlockValue{V: "not-a-struct"}
		_, err := sb.ValueAtPath(wrong, []string{"name"})
		if err == nil {
			t.Fatalf("expected type mismatch error")
		}
		if !strings.Contains(err.Error(), "TypeMismatch") && !strings.Contains(err.Error(), "value must be a *StructBlockValue") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Err_FieldNotFound", func(t *testing.T) {
		_, err := sb.ValueAtPath(bound, []string{"nope"})
		if err == nil {
			t.Fatalf("expected field not found error")
		}
		if !strings.Contains(err.Error(), "no such field") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestStructBlock_ValueFromDB(t *testing.T) {
	sb := NewSimpleStructBlock()
	sb.SetName("person")

	// 1) Empty/null input -> nil, nil
	t.Run("Empty", func(t *testing.T) {
		v, err := sb.ValueFromDB(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if v != nil {
			t.Fatalf("expected nil, got %T:%v", v, v)
		}
	})

	// 2) Valid JSON -> Go-typed map
	t.Run("OK", func(t *testing.T) {
		raw := map[string]any{
			"name":     "John Doe",
			"age":      30,
			"email":    mustAddr(t, "john@localhost"),
			"password": "hunter2",
			"date":     "2021-01-01",
			"datetime": "2021-01-01T00:00:00",
		}
		got, err := sb.ValueFromDB(jraw(t, raw))
		if err != nil {
			s, _ := err.(*blocks.BaseBlockValidationError[string]).MarshalJSON()
			t.Log(string(s))
			t.Fatalf("unexpected err: %v", err)
		}
		m, ok := got.(*blocks.StructBlockValue)
		if !ok {
			t.Fatalf("expected *StructBlockValue, got %T", got)
		}

		want := map[string]interface{}{
			"name":     &blocks.FieldBlockValue{V: "John Doe"},
			"age":      &blocks.FieldBlockValue{V: 30},
			"email":    &blocks.FieldBlockValue{V: mustAddr(t, "john@localhost")},
			"password": &blocks.FieldBlockValue{V: "hunter2"},
			"date":     &blocks.FieldBlockValue{V: mustDate(t, "2006-01-02", "2021-01-01")},
			"datetime": &blocks.FieldBlockValue{V: mustDate(t, "2006-01-02T15:04:05", "2021-01-01T00:00:00")},
		}
		if !deepEqual(m.V, want) {
			t.Fatalf("mismatch\nwant = %#v\ngot = %#v", want, m)
		}
	})

	// 3) Aggregated field errors (invalid email & invalid date), but partial data returned
	t.Run("AggregatedErrors", func(t *testing.T) {
		raw := map[string]any{
			"name":     "Jane Doe",
			"age":      25,
			"email":    "not-an-email", // invalid
			"password": "pw",
			"date":     "2021-01-00", // invalid date
			"datetime": "2021-01-01T00:00:00",
		}
		got, err := sb.ValueFromDB(jraw(t, raw))
		if err == nil {
			t.Fatalf("expected aggregated errors, got nil")
		}
		m, ok := got.(*blocks.StructBlockValue)
		if !ok {
			t.Fatalf("expected *StructBlockValue, got %T", got)
		}

		// Valid parts should still be typed
		if m.V["name"].(blocks.BoundBlockValue).Data() != "Jane Doe" || m.V["age"].(blocks.BoundBlockValue).Data() != 25 || m.V["password"].(blocks.BoundBlockValue).Data() != "pw" {
			t.Fatalf("partial data mismatch: %#v", m)
		}
		if _, ok := m.V["email"]; ok {
			t.Fatalf("email should not be present on partial success")
		}
		if _, ok := m.V["date"]; ok {
			t.Fatalf("date should not be present on partial success")
		}
		if _, ok := m.V["datetime"].(blocks.BoundBlockValue).Data().(time.Time); !ok {
			t.Fatalf("datetime should be parsed to time.Time: %#v", m.V["datetime"])
		}

		var errs = err.(*blocks.BaseBlockValidationError[string])
		if len(errs.Errors) != 2 {
			t.Fatalf("expected 2 errors, got %d: %#v", len(errs.Errors), errs.Errors)
		}
		if _, ok := errs.Errors["email"]; !ok {
			t.Fatalf("expected email error, got: %#v", errs.Errors)
		}
		if _, ok := errs.Errors["date"]; !ok {
			t.Fatalf("expected date error, got: %#v", errs.Errors)
		}
	})
}

func TestStructBlock(t *testing.T) {
	var b = NewSimpleStructBlock()

	b.SetName("test_block")

	t.Run("ValueFromDataDict", func(t *testing.T) {
		var data, err = b.ValueFromDataDict(context.Background(), structBlockDataRaw, nil, "test_block")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if data == nil {
			t.Errorf("Expected data, got nil")
		}

		var d = data.(*blocks.StructBlockValue)

		for k, v := range d.V {
			if !deepEqual(v, structBlockstructBlockDataRawCmp.V[k]) {
				t.Errorf("Expected %v, got %v", structBlockstructBlockDataRawCmp.V[k], v)
			}
		}

		if len(d.V) != len(structBlockstructBlockDataRawCmp.V) {
			t.Errorf("Expected length %d, got %d", len(structBlockstructBlockDataRawCmp.V), len(d.V))
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		var data, err = b.ValueToGo(structBlockstructBlockDataRawCmp)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var d = data.(*blocks.StructBlockValue)

		for k, v := range d.V {
			if !deepEqual(v, structBlockDataGo.V[k]) {
				t.Errorf("Expected %v, got %v", structBlockDataGo.V[k], v)
			}
		}

		if len(d.V) != len(structBlockDataGo.V) {
			t.Errorf("Expected length %d, got %d", len(structBlockDataGo.V), len(d.V))
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		var (
			data = b.ValueToForm(structBlockDataGo)
			d    = data.(*blocks.StructBlockValue)
		)

		for k, v := range d.V {
			if !deepEqual(v, structBlockstructBlockDataRawCmp.V[k]) {
				t.Errorf("Expected %v, got %v", structBlockstructBlockDataRawCmp.V[k], v)
			}
		}

		if len(d.V) != len(structBlockstructBlockDataRawCmp.V) {
			t.Errorf("Expected length %d, got %d", len(structBlockstructBlockDataRawCmp.V), len(d.V))
		}
	})

	t.Run("ConversionsEqual", func(t *testing.T) {
		var data, err = b.ValueToGo(structBlockstructBlockDataRawCmp)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var (
			data2 = b.ValueToForm(data)
			d1    = data.(*blocks.StructBlockValue)
			d2    = data2.(*blocks.StructBlockValue)
		)

		for k, v := range d2.V {
			if v != structBlockstructBlockDataRawCmp.V[k] {
				t.Errorf("Expected %v, got %v", structBlockstructBlockDataRawCmp.V[k], v)
			}
		}

		if len(d1.V) != len(d2.V) {
			t.Errorf("Expected length %d, got %d", len(d2.V), len(d1.V))
		}
	})

	b.AddField("test_nested", NewSimpleStructBlock())

	var (
		nestedStructBlockDataRaw = maps.Clone(structBlockDataRaw)
		nestedStructBlockDataCmp = &blocks.StructBlockValue{V: maps.Clone(structBlockstructBlockDataRawCmp.V)}
		nestedStructBlockDataGo  = &blocks.StructBlockValue{V: maps.Clone(structBlockDataGo.V)}
	)

	nestedStructBlockDataRaw.Set("test_block-test_nested-name", "Jane Doe")
	nestedStructBlockDataRaw.Set("test_block-test_nested-age", "25")
	nestedStructBlockDataRaw.Set("test_block-test_nested-email", "test2@localhost")
	nestedStructBlockDataRaw.Set("test_block-test_nested-password", "password2")
	nestedStructBlockDataRaw.Set("test_block-test_nested-date", "2021-01-02")
	nestedStructBlockDataRaw.Set("test_block-test_nested-datetime", "2021-01-02T00:00:00")

	nestedStructBlockDataCmp.V["test_nested"] = &blocks.StructBlockValue{
		V: map[string]interface{}{
			"name":     "Jane Doe",
			"age":      "25",
			"email":    "test2@localhost",
			"password": "password2",
			"date":     "2021-01-02",
			"datetime": "2021-01-02T00:00:00",
		},
	}

	nestedStructBlockDataGo.V["test_nested"] = &blocks.StructBlockValue{
		V: map[string]interface{}{
			"name":     "Jane Doe",
			"age":      25,
			"email":    must(mail.ParseAddress("test2@localhost")),
			"password": "password2",
			"date":     must(time.Parse("2006-01-02", "2021-01-02")),
			"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-02T00:00:00")),
		},
	}

	t.Run("NestedStructBlock", func(t *testing.T) {

		t.Run("ValueFromDataDict", func(t *testing.T) {
			var data, err = b.ValueFromDataDict(context.Background(), nestedStructBlockDataRaw, nil, "test_block")

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if data == nil {
				t.Errorf("Expected data, got nil")
			}

			var d = data.(*blocks.StructBlockValue)

			for k, v := range d.V {
				if !deepEqual(v, nestedStructBlockDataCmp.V[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataCmp.V[k], v)
				}
			}

			if len(d.V) != len(nestedStructBlockDataCmp.V) {
				t.Errorf("Expected length %d, got %d", len(nestedStructBlockDataCmp.V), len(d.V))
			}
		})

		t.Run("ValueToGo", func(t *testing.T) {
			var data, err = b.ValueToGo(nestedStructBlockDataCmp)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			var d = data.(*blocks.StructBlockValue)

			for k, v := range d.V {
				if !deepEqual(v, nestedStructBlockDataGo.V[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataGo.V[k], v)
				}
			}

			if len(d.V) != len(nestedStructBlockDataGo.V) {
				t.Errorf("Expected length %d, got %d", len(nestedStructBlockDataGo.V), len(d.V))
			}
		})

		t.Run("ValueToForm", func(t *testing.T) {
			var (
				data = b.ValueToForm(nestedStructBlockDataGo)
				d    = data.(*blocks.StructBlockValue)
			)

			for k, v := range d.V {
				if !deepEqual(v, nestedStructBlockDataCmp.V[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataCmp.V[k], v)
				}
			}

			if len(d.V) != len(nestedStructBlockDataCmp.V) {
				t.Errorf("Expected length %d, got %d", len(nestedStructBlockDataCmp.V), len(d.V))
			}
		})

		t.Run("ConversionsEqual", func(t *testing.T) {
			var data, err = b.ValueToGo(nestedStructBlockDataCmp)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			var (
				data2 = b.ValueToForm(data)
				d1    = data.(*blocks.StructBlockValue)
				d2    = data2.(*blocks.StructBlockValue)
			)

			for k, v := range d2.V {
				if !deepEqual(v, nestedStructBlockDataCmp.V[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataCmp.V[k], v)
				}
			}

			if len(d1.V) != len(d2.V) {
				t.Errorf("Expected length %d, got %d", len(d2.V), len(d1.V))
			}
		})
	})
}
