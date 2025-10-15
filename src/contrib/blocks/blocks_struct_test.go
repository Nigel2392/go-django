package blocks_test

import (
	"context"
	"encoding/json"
	"maps"
	"net/mail"
	"net/url"
	"reflect"
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

var structBlockstructBlockDataRawCmp = map[string]interface{}{
	"name":     "John Doe",
	"age":      "30",
	"email":    "test@localhost",
	"password": "password",
	"date":     "2021-01-01",
	"datetime": "2021-01-01T00:00:00",
}

var structBlockDataGo = map[string]interface{}{
	"name":     "John Doe",
	"age":      30,
	"email":    must(mail.ParseAddress("test@localhost")),
	"password": "password",
	"date":     must(time.Parse("2006-01-02", "2021-01-01")),
	"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-01T00:00:00")),
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
			"date":     mustDate(t, "2006-01-02", "2021-01-01"),
			"datetime": mustDate(t, "2006-01-02T15:04:05", "2021-01-01T00:00:00"),
		}
		got, err := sb.ValueFromDB(jraw(t, raw))
		if err != nil {
			s, _ := err.(*blocks.BaseBlockValidationError[string]).MarshalJSON()
			t.Log(string(s))
			t.Fatalf("unexpected err: %v", err)
		}
		m, ok := got.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", got)
		}

		want := map[string]interface{}{
			"name":     "John Doe",
			"age":      30,
			"email":    mustAddr(t, "john@localhost"),
			"password": "hunter2",
			"date":     mustDate(t, "2006-01-02", "2021-01-01"),
			"datetime": mustDate(t, "2006-01-02T15:04:05", "2021-01-01T00:00:00"),
		}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("mismatch\nwant=%#v\ngot =%#v", want, m)
		}
	})

	// 3) Aggregated field errors (invalid email & invalid date), but partial data returned
	t.Run("AggregatedErrors", func(t *testing.T) {
		raw := map[string]json.RawMessage{
			"name":     jraw(t, "Jane Doe"),
			"age":      jraw(t, 25),
			"email":    jraw(t, "not-an-email"), // invalid
			"password": jraw(t, "pw"),
			"date":     jraw(t, "2021-13-40"), // invalid date
			"datetime": jraw(t, "2021-01-02T01:02:03"),
		}
		got, err := sb.ValueFromDB(jraw(t, raw))
		if err == nil {
			t.Fatalf("expected aggregated errors, got nil")
		}
		m, ok := got.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", got)
		}
		// Valid parts should still be typed
		if m["name"] != "Jane Doe" || m["age"] != 25 || m["password"] != "pw" {
			t.Fatalf("partial data mismatch: %#v", m)
		}
		if _, ok := m["email"]; ok {
			t.Fatalf("email should not be present on partial success")
		}
		if _, ok := m["date"]; ok {
			t.Fatalf("date should not be present on partial success")
		}
		if _, ok := m["datetime"].(time.Time); !ok {
			t.Fatalf("datetime should be parsed to time.Time: %#v", m["datetime"])
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

		var d = data.(map[string]interface{})

		for k, v := range d {
			if !reflect.DeepEqual(v, structBlockstructBlockDataRawCmp[k]) {
				t.Errorf("Expected %v, got %v", structBlockstructBlockDataRawCmp[k], v)
			}
		}

		if len(d) != len(structBlockstructBlockDataRawCmp) {
			t.Errorf("Expected length %d, got %d", len(structBlockstructBlockDataRawCmp), len(d))
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		var data, err = b.ValueToGo(structBlockstructBlockDataRawCmp)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var d = data.(map[string]interface{})

		for k, v := range d {
			if !reflect.DeepEqual(v, structBlockDataGo[k]) {
				t.Errorf("Expected %v, got %v", structBlockDataGo[k], v)
			}
		}

		if len(d) != len(structBlockDataGo) {
			t.Errorf("Expected length %d, got %d", len(structBlockDataGo), len(d))
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		var (
			data = b.ValueToForm(structBlockDataGo)
			d    = data.(map[string]interface{})
		)

		for k, v := range d {
			if !reflect.DeepEqual(v, structBlockstructBlockDataRawCmp[k]) {
				t.Errorf("Expected %v, got %v", structBlockstructBlockDataRawCmp[k], v)
			}
		}

		if len(d) != len(structBlockstructBlockDataRawCmp) {
			t.Errorf("Expected length %d, got %d", len(structBlockstructBlockDataRawCmp), len(d))
		}
	})

	t.Run("ConversionsEqual", func(t *testing.T) {
		var data, err = b.ValueToGo(structBlockstructBlockDataRawCmp)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var (
			data2 = b.ValueToForm(data)
			d1    = data.(map[string]interface{})
			d2    = data2.(map[string]interface{})
		)

		for k, v := range d2 {
			if v != structBlockstructBlockDataRawCmp[k] {
				t.Errorf("Expected %v, got %v", structBlockstructBlockDataRawCmp[k], v)
			}
		}

		if len(d1) != len(d2) {
			t.Errorf("Expected length %d, got %d", len(d2), len(d1))
		}
	})

	b.AddField("test_nested", NewSimpleStructBlock())

	var (
		nestedStructBlockDataRaw = maps.Clone(structBlockDataRaw)
		nestedStructBlockDataCmp = maps.Clone(structBlockstructBlockDataRawCmp)
		nestedStructBlockDataGo  = maps.Clone(structBlockDataGo)
	)

	nestedStructBlockDataRaw.Set("test_block-test_nested-name", "Jane Doe")
	nestedStructBlockDataRaw.Set("test_block-test_nested-age", "25")
	nestedStructBlockDataRaw.Set("test_block-test_nested-email", "test2@localhost")
	nestedStructBlockDataRaw.Set("test_block-test_nested-password", "password2")
	nestedStructBlockDataRaw.Set("test_block-test_nested-date", "2021-01-02")
	nestedStructBlockDataRaw.Set("test_block-test_nested-datetime", "2021-01-02T00:00:00")

	nestedStructBlockDataCmp["test_nested"] = map[string]interface{}{
		"name":     "Jane Doe",
		"age":      "25",
		"email":    "test2@localhost",
		"password": "password2",
		"date":     "2021-01-02",
		"datetime": "2021-01-02T00:00:00",
	}

	nestedStructBlockDataGo["test_nested"] = map[string]interface{}{
		"name":     "Jane Doe",
		"age":      25,
		"email":    must(mail.ParseAddress("test2@localhost")),
		"password": "password2",
		"date":     must(time.Parse("2006-01-02", "2021-01-02")),
		"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-02T00:00:00")),
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

			var d = data.(map[string]interface{})

			for k, v := range d {
				if !reflect.DeepEqual(v, nestedStructBlockDataCmp[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataCmp[k], v)
				}
			}

			if len(d) != len(nestedStructBlockDataCmp) {
				t.Errorf("Expected length %d, got %d", len(nestedStructBlockDataCmp), len(d))
			}
		})

		t.Run("ValueToGo", func(t *testing.T) {
			var data, err = b.ValueToGo(nestedStructBlockDataCmp)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			var d = data.(map[string]interface{})

			for k, v := range d {
				if !reflect.DeepEqual(v, nestedStructBlockDataGo[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataGo[k], v)
				}
			}

			if len(d) != len(nestedStructBlockDataGo) {
				t.Errorf("Expected length %d, got %d", len(nestedStructBlockDataGo), len(d))
			}
		})

		t.Run("ValueToForm", func(t *testing.T) {
			var (
				data = b.ValueToForm(nestedStructBlockDataGo)
				d    = data.(map[string]interface{})
			)

			for k, v := range d {
				if !reflect.DeepEqual(v, nestedStructBlockDataCmp[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataCmp[k], v)
				}
			}

			if len(d) != len(nestedStructBlockDataCmp) {
				t.Errorf("Expected length %d, got %d", len(nestedStructBlockDataCmp), len(d))
			}
		})

		t.Run("ConversionsEqual", func(t *testing.T) {
			var data, err = b.ValueToGo(nestedStructBlockDataCmp)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			var (
				data2 = b.ValueToForm(data)
				d1    = data.(map[string]interface{})
				d2    = data2.(map[string]interface{})
			)

			for k, v := range d2 {
				if !reflect.DeepEqual(v, nestedStructBlockDataCmp[k]) {
					t.Errorf("Expected %v, got %v", nestedStructBlockDataCmp[k], v)
				}
			}

			if len(d1) != len(d2) {
				t.Errorf("Expected length %d, got %d", len(d2), len(d1))
			}
		})
	})
}
