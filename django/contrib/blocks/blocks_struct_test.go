package blocks_test

import (
	"maps"
	"net/mail"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/django/contrib/blocks"
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

func TestStructBlock(t *testing.T) {
	var b = NewSimpleStructBlock()

	b.SetName("test_block")

	t.Run("ValueFromDataDict", func(t *testing.T) {
		var data, err = b.ValueFromDataDict(structBlockDataRaw, nil, "test_block")

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
			var data, err = b.ValueFromDataDict(nestedStructBlockDataRaw, nil, "test_block")

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
