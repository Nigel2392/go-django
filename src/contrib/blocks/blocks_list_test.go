package blocks_test

import (
	"context"
	"net/mail"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/google/uuid"
)

func NewListBlock() *blocks.ListBlock {
	var b = blocks.NewListBlock(NewSimpleStructBlock())
	b.SetName("test_list_block")
	return b
}

var (
	ListBlockDataRaw = map[string][]string{
		"test_list_block--total":     {"2"},
		"test_list_block-id-0":       {uuid.Nil.String()},
		"test_list_block-order-0":    {"0"},
		"test_list_block-0-name":     {"John Doe"},
		"test_list_block-0-age":      {"30"},
		"test_list_block-0-email":    {"test@localhost"},
		"test_list_block-0-password": {"password"},
		"test_list_block-0-date":     {"2021-01-01"},
		"test_list_block-0-datetime": {"2021-01-01T00:00:00"},
		"test_list_block-id-1":       {uuid.Nil.String()},
		"test_list_block-order-1":    {"1"},
		"test_list_block-1-name":     {"Jane Doe"},
		"test_list_block-1-age":      {"25"},
		"test_list_block-1-email":    {"test2@localhost"},
		"test_list_block-1-password": {"password2"},
		"test_list_block-1-date":     {"2021-01-02"},
		"test_list_block-1-datetime": {"2021-01-02T00:00:00"},
	}

	ListBlockDataRawCmp = blocks.ListBlockData{
		{
			ID:    uuid.Nil,
			Order: 0,
			Data: map[string]interface{}{
				"name":     "John Doe",
				"age":      "30",
				"email":    "test@localhost",
				"password": "password",
				"date":     "2021-01-01",
				"datetime": "2021-01-01T00:00:00",
			},
		},
		{
			ID:    uuid.Nil,
			Order: 1,
			Data: map[string]interface{}{
				"name":     "Jane Doe",
				"age":      "25",
				"email":    "test2@localhost",
				"password": "password2",
				"date":     "2021-01-02",
				"datetime": "2021-01-02T00:00:00",
			},
		},
	}

	ListBlockDataGo = blocks.ListBlockData{
		{
			ID:    uuid.Nil,
			Order: 0,
			Data: map[string]interface{}{
				"name":     "John Doe",
				"age":      30,
				"email":    must(mail.ParseAddress("test@localhost")),
				"password": "password",
				"date":     must(time.Parse("2006-01-02", "2021-01-01")),
				"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-01T00:00:00")),
			},
		},
		{
			ID:    uuid.Nil,
			Order: 1,
			Data: map[string]interface{}{
				"name":     "Jane Doe",
				"age":      25,
				"email":    must(mail.ParseAddress("test2@localhost")),
				"password": "password2",
				"date":     must(time.Parse("2006-01-02", "2021-01-02")),
				"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-02T00:00:00")),
			},
		},
	}
)

func TestListBlock(t *testing.T) {
	var b = NewListBlock()

	t.Run("ValueFromDataDict", func(t *testing.T) {
		var data, err = b.ValueFromDataDict(context.Background(), ListBlockDataRaw, nil, "test_list_block")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if data == nil {
			t.Errorf("Expected data, got nil")
		}

		var d = data.(blocks.ListBlockData)

		for i, v := range d {
			if !reflect.DeepEqual(*v, *ListBlockDataRawCmp[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataRawCmp[i], *v)
			}
		}

		if len(d) != len(ListBlockDataRawCmp) {
			t.Errorf("Expected length %d, got %d", len(ListBlockDataRawCmp), len(d))
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		var data, err = b.ValueToGo(ListBlockDataRawCmp)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var d = data.(blocks.ListBlockData)

		for i, v := range d {
			if !reflect.DeepEqual(*v, *ListBlockDataGo[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataGo[i], v)
			}
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		var (
			data = b.ValueToForm(ListBlockDataGo)
			d    = data.(blocks.ListBlockData)
		)

		for i, v := range d {
			if !reflect.DeepEqual(*v, *ListBlockDataRawCmp[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataRawCmp[i], *v)
			}
		}
	})

	t.Run("ConversionsEqual", func(t *testing.T) {
		var (
			data, err = b.ValueToGo(ListBlockDataRawCmp)
			data2     = b.ValueToForm(data)
		)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !reflect.DeepEqual(data2, ListBlockDataRawCmp) {
			t.Errorf("Expected %v, got %v", ListBlockDataRawCmp, data2)
		}
	})
}

func TestListBlock_ValueFromDB(t *testing.T) {
	child := NewSimpleStructBlock()
	lb := blocks.NewListBlock(child)
	lb.SetName("people")

	// 1) Empty -> nil, nil
	t.Run("Empty", func(t *testing.T) {
		v, err := lb.ValueFromDB(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if v != nil {
			t.Fatalf("expected nil, got %T:%v", v, v)
		}
	})

	type item struct {
		ID   uuid.UUID      `json:"id"`
		Data map[string]any `json:"data"`
	}

	// 2) OK
	t.Run("OK", func(t *testing.T) {
		id0 := uuid.New()
		id1 := uuid.New()

		data := []blocks.JSONListBlockValue{
			{
				ID: id0,
				// raw child data is JSON that the child block can parse from DB
				Data: jraw(t, map[string]any{
					"name":     "John Doe",
					"age":      30,
					"email":    mustAddr(t, "john@localhost"),
					"password": "hunter2",
					"date":     mustDate(t, "2006-01-02", "2021-01-01"),
					"datetime": mustDate(t, "2006-01-02T15:04:05", "2021-01-01T00:00:00"),
				}),
			},
			{
				ID: id1,
				Data: jraw(t, map[string]any{
					"name":     "Jane Doe",
					"age":      25,
					"email":    mustAddr(t, "jane@localhost"),
					"password": "pw2",
					"date":     mustDate(t, "2006-01-02", "2021-01-02"),
					"datetime": mustDate(t, "2006-01-02T15:04:05", "2021-01-02T00:00:00"),
				}),
			},
		}

		got, err := lb.ValueFromDB(jraw(t, data))
		if err != nil {
			errs := err.(*blocks.BaseBlockValidationError[int])
			j, _ := errs.MarshalJSON()
			t.Fatalf("unexpected error: %s", j)
		}

		lv, ok := got.(blocks.ListBlockData)
		if !ok {
			t.Fatalf("expected ListBlockData, got %T", got)
		}

		if len(lv) != 2 {
			t.Fatalf("expected 2 items, got %d", len(lv))
		}

		if lv[0] == nil || lv[1] == nil {
			t.Fatalf("expected non-nil items, got %#v", lv)
		}

		// Order is the index in the array
		if lv[0].ID != id0 || lv[0].Order != 0 {
			t.Fatalf("item0 meta mismatch: %#v", lv[0])
		}
		if lv[1].ID != id1 || lv[1].Order != 1 {
			t.Fatalf("item1 meta mismatch: %#v", lv[1])
		}

		// Typed data checks (spot-check a few types)
		m0 := lv[0].Data.(map[string]interface{})
		if m0["age"] != 30 || m0["name"] != "John Doe" {
			t.Fatalf("item0 data mismatch: %#v", m0)
		}
		if _, ok := m0["email"].(*mail.Address); !ok {
			t.Fatalf("item0 email not parsed: %#v", m0["email"])
		}
		if _, ok := m0["date"].(time.Time); !ok {
			t.Fatalf("item0 date not parsed: %#v", m0["date"])
		}
		if _, ok := m0["datetime"].(time.Time); !ok {
			t.Fatalf("item0 datetime not parsed: %#v", m0["datetime"])
		}
	})

	// 3) Aggregated child errors
	t.Run("AggregatedErrors", func(t *testing.T) {
		id := uuid.New()
		data := []blocks.JSONListBlockValue{
			{
				ID: id,
				Data: jraw(t, map[string]any{
					"name":     "Bad User",
					"age":      10,
					"email":    "not-an-email", // invalid
					"password": "x",
					"date":     "2021-99-01", // invalid
					"datetime": "2021-01-02T00:00:00",
				}),
			},
		}
		got, err := lb.ValueFromDB(jraw(t, data))
		if err == nil {
			t.Fatalf("expected aggregated error, got nil")
		}
		lv, ok := got.(blocks.ListBlockData)
		if !ok || len(lv) != 1 {
			t.Fatalf("expected ListBlockData with one entry, got %T:%v", got, got)
		}

		if lv[0] == nil {
			t.Fatalf("expected non-nil item, got %#v", lv)
		}

		// Partial success should include only valid typed fields
		m := lv[0].Data.(map[string]interface{})
		if m["name"] != "Bad User" || m["age"] != 10 {
			t.Fatalf("partial data mismatch: %#v", m)
		}
		if _, ok := m["email"]; ok {
			t.Fatalf("email should be missing on error")
		}
		if _, ok := m["date"]; ok {
			t.Fatalf("date should be missing on error")
		}
	})
}
