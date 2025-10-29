package blocks_test

import (
	"context"
	"net/mail"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

var __BLOCK = reflect.TypeOf((*blocks.Block)(nil)).Elem()

func deepEqual(expected, actual interface{}) bool {
	return cmp.Equal(expected, actual, cmp.FilterPath(func(p cmp.Path) bool {
		for i := 0; i < len(p); i++ {
			var ps = p.Index(i)
			if ps.Type() == __BLOCK || ps.Type().Implements(__BLOCK) {
				return true
			}
		}
		return p.Last().String() == "_rawData" || p.Last().String() == "._rawData"
	}, cmp.Ignore()))
}

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

	ListBlockDataRawCmp = &blocks.ListBlockValue{
		V: []*blocks.ListBlockData{
			{
				ID:    uuid.Nil,
				Order: 0,
				Data: &blocks.StructBlockValue{
					V: map[string]interface{}{
						"name":     "John Doe",
						"age":      "30",
						"email":    "test@localhost",
						"password": "password",
						"date":     "2021-01-01",
						"datetime": "2021-01-01T00:00:00",
					},
				},
			},
			{
				ID:    uuid.Nil,
				Order: 1,
				Data: &blocks.StructBlockValue{
					V: map[string]interface{}{
						"name":     "Jane Doe",
						"age":      "25",
						"email":    "test2@localhost",
						"password": "password2",
						"date":     "2021-01-02",
						"datetime": "2021-01-02T00:00:00",
					},
				},
			},
		},
	}

	ListBlockDataGo = &blocks.ListBlockValue{
		V: []*blocks.ListBlockData{
			{
				ID:    uuid.Nil,
				Order: 0,
				Data: &blocks.StructBlockValue{
					V: map[string]interface{}{
						"name":     "John Doe",
						"age":      30,
						"email":    must(mail.ParseAddress("test@localhost")),
						"password": "password",
						"date":     must(time.Parse("2006-01-02", "2021-01-01")),
						"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-01T00:00:00")),
					},
				},
			},
			{
				ID:    uuid.Nil,
				Order: 1,
				Data: &blocks.StructBlockValue{
					V: map[string]interface{}{
						"name":     "Jane Doe",
						"age":      25,
						"email":    must(mail.ParseAddress("test2@localhost")),
						"password": "password2",
						"date":     must(time.Parse("2006-01-02", "2021-01-02")),
						"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-02T00:00:00")),
					},
				},
			},
		},
	}
)

func TestListBlock_ValueAtPath(t *testing.T) {
	// Child is a simple struct
	child := blocks.NewStructBlock()
	child.AddField("name", blocks.CharBlock())
	child.AddField("age", blocks.NumberBlock())

	lb := blocks.NewListBlock(child)
	lb.SetName("people")

	// Build Bound struct values for list entries.
	p0, err := child.ValueFromDB(jraw(t, map[string]any{"name": "John", "age": 30}))
	if err != nil {
		t.Fatalf("child 0 from DB: %v", err)
	}
	p1, err := child.ValueFromDB(jraw(t, map[string]any{"name": "Jane", "age": 25}))
	if err != nil {
		t.Fatalf("child 1 from DB: %v", err)
	}

	bound := &blocks.ListBlockValue{
		V: []*blocks.ListBlockData{
			{Data: p0},
			{Data: p1},
		},
	}

	t.Run("OK_IndexOnly", func(t *testing.T) {
		got, err := lb.ValueAtPath(bound, []string{"0"})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if _, ok := got.(map[string]any); !ok {
			t.Fatalf("want map[string]any, got %T", got)
		}
	})

	t.Run("OK_IndexAndDeepPath", func(t *testing.T) {
		got, err := lb.ValueAtPath(bound, []string{"1", "name"})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.(string) != "Jane" {
			t.Fatalf("want Jane, got %#v", got)
		}
	})

	t.Run("Err_TypeMismatch", func(t *testing.T) {
		wrong := &blocks.FieldBlockValue{V: "not-a-list"}
		_, err := lb.ValueAtPath(wrong, []string{"0"})
		if err == nil {
			t.Fatalf("expected type mismatch error")
		}
		if !strings.Contains(err.Error(), "value must be a *ListBlockValue") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Err_InvalidIndex", func(t *testing.T) {
		_, err := lb.ValueAtPath(bound, []string{"x"})
		if err == nil {
			t.Fatalf("expected invalid index error")
		}
		if !strings.Contains(err.Error(), "invalid index") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Err_IndexOutOfRange", func(t *testing.T) {
		_, err := lb.ValueAtPath(bound, []string{"5"})
		if err == nil {
			t.Fatalf("expected out of range error")
		}
		if !strings.Contains(err.Error(), "index out of range") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

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

		var d = data.(*blocks.ListBlockValue)

		if len(d.V) != len(ListBlockDataRawCmp.V) {
			t.Errorf("Expected length %d, got %d", len(ListBlockDataRawCmp.V), len(d.V))
		}

		for i, v := range d.V {
			if !deepEqual(*v, *ListBlockDataRawCmp.V[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataRawCmp.V[i], *v)
			}
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		var data, err = b.ValueToGo(ListBlockDataRawCmp)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var d = data.(*blocks.ListBlockValue)
		for i, v := range d.V {
			if !deepEqual(*v, *ListBlockDataGo.V[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataGo.V[i], *v)
			}
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		var (
			data = b.ValueToForm(ListBlockDataGo)
			d    = data.(*blocks.ListBlockValue)
		)

		for i, v := range d.V {
			if !deepEqual(*v, *ListBlockDataRawCmp.V[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataRawCmp.V[i], *v)
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

		if !deepEqual(data2.(*blocks.ListBlockValue).V, ListBlockDataRawCmp.V) {
			t.Errorf("Expected %v, got %v", ListBlockDataRawCmp.V, data2.(*blocks.ListBlockValue).V)
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

	// 2) OK
	t.Run("OK", func(t *testing.T) {
		id0 := uuid.New()
		id1 := uuid.New()

		data := []blocks.JSONStreamBlockData{
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

		lv, ok := got.(*blocks.ListBlockValue)
		if !ok {
			t.Fatalf("expected *blocks.ListBlockValue, got %T", got)
		}

		if len(lv.V) != 2 {
			t.Fatalf("expected 2 items, got %d", len(lv.V))
		}

		if lv.V[0] == nil || lv.V[1] == nil {
			t.Fatalf("expected non-nil items, got %#v", lv)
		}

		// Order is the index in the array
		if lv.V[0].ID != id0 || lv.V[0].Order != 0 {
			t.Fatalf("item0 meta mismatch: %#v", lv.V[0])
		}
		if lv.V[1].ID != id1 || lv.V[1].Order != 1 {
			t.Fatalf("item1 meta mismatch: %#v", lv.V[1])
		}

		// Typed data checks (spot-check a few types)
		m0 := lv.V[0].Data.(*blocks.StructBlockValue).V
		if m0["age"].(blocks.BoundBlockValue).Data() != 30 || m0["name"].(blocks.BoundBlockValue).Data() != "John Doe" {
			t.Fatalf("item0 data mismatch: %#v", m0)
		}
		if _, ok := m0["email"].(blocks.BoundBlockValue).Data().(*mail.Address); !ok {
			t.Fatalf("item0 email not parsed: %#v", m0["email"])
		}
		if _, ok := m0["date"].(blocks.BoundBlockValue).Data().(time.Time); !ok {
			t.Fatalf("item0 date not parsed: %#v", m0["date"])
		}
		if _, ok := m0["datetime"].(blocks.BoundBlockValue).Data().(time.Time); !ok {
			t.Fatalf("item0 datetime not parsed: %#v", m0["datetime"])
		}
	})

	// 3) Aggregated child errors
	t.Run("AggregatedErrors", func(t *testing.T) {
		id := uuid.New()
		data := []blocks.JSONStreamBlockData{
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
		lv, ok := got.(*blocks.ListBlockValue)
		if !ok || len(lv.V) != 1 {
			t.Fatalf("expected ListBlockData with one entry, got %T:%v", got, got)
		}

		if lv.V[0] == nil {
			t.Fatalf("expected non-nil item, got %#v", lv)
		}

		// Partial success should include only valid typed fields
		m := lv.V[0].Data.(*blocks.StructBlockValue).V
		if m["name"].(blocks.BoundBlockValue).Data() != "Bad User" || m["age"].(blocks.BoundBlockValue).Data() != 10 {
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
