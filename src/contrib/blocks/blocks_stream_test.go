package blocks_test

import (
	"context"
	"net/mail"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/google/uuid"
)

func NewStreamPersonBlock() *blocks.StreamBlock {
	sb := blocks.NewStreamBlock()
	sb.SetName("test_stream")
	sb.AddField("person", NewSimpleStructBlock())
	return sb
}

func TestStreamBlock(t *testing.T) {
	t.Helper()

	// Helpers (local to avoid package-level name collisions)
	mustAddr := func(s string) *mail.Address {
		a, err := mail.ParseAddress(s)
		if err != nil {
			t.Fatalf("parse address %q: %v", s, err)
		}
		return a
	}
	mustDate := func(layout, s string) time.Time {
		d, err := time.Parse(layout, s)
		if err != nil {
			t.Fatalf("parse time %q: %v", s, err)
		}
		return d
	}

	b := NewStreamPersonBlock()

	// ---------- Raw form data ----------
	raw := url.Values{
		"test_stream--total": {"2"},

		"test_stream-id-0":       {uuid.Nil.String()},
		"test_stream-order-0":    {"0"},
		"test_stream-0--type":    {"person"},
		"test_stream-0-name":     {"John Doe"},
		"test_stream-0-age":      {"30"},
		"test_stream-0-email":    {"test@localhost"},
		"test_stream-0-password": {"password"},
		"test_stream-0-date":     {"2021-01-01"},
		"test_stream-0-datetime": {"2021-01-01T00:00:00"},

		"test_stream-id-1":       {uuid.Nil.String()},
		"test_stream-order-1":    {"1"},
		"test_stream-1--type":    {"person"},
		"test_stream-1-name":     {"Jane Doe"},
		"test_stream-1-age":      {"25"},
		"test_stream-1-email":    {"test2@localhost"},
		"test_stream-1-password": {"password2"},
		"test_stream-1-date":     {"2021-01-02"},
		"test_stream-1-datetime": {"2021-01-02T00:00:00"},
	}

	// ---------- Expected: "form" representation (strings) ----------
	formItem0 := map[string]interface{}{
		"name":     "John Doe",
		"age":      "30",
		"email":    "test@localhost",
		"password": "password",
		"date":     "2021-01-01",
		"datetime": "2021-01-01T00:00:00",
	}
	formItem1 := map[string]interface{}{
		"name":     "Jane Doe",
		"age":      "25",
		"email":    "test2@localhost",
		"password": "password2",
		"date":     "2021-01-02",
		"datetime": "2021-01-02T00:00:00",
	}
	formValueCmp := &blocks.StreamBlockValue{
		Blocks: []*blocks.StreamBlockData{
			{ID: uuid.Nil, Type: "person", Data: formItem0, Order: 0},
			{ID: uuid.Nil, Type: "person", Data: formItem1, Order: 1},
		},
	}

	// ---------- Expected: Go representation ----------
	goItem0 := map[string]interface{}{
		"name":     "John Doe",
		"age":      30,
		"email":    mustAddr("test@localhost"),
		"password": "password",
		"date":     mustDate("2006-01-02", "2021-01-01"),
		"datetime": mustDate("2006-01-02T15:04:05", "2021-01-01T00:00:00"),
	}
	goItem1 := map[string]interface{}{
		"name":     "Jane Doe",
		"age":      25,
		"email":    mustAddr("test2@localhost"),
		"password": "password2",
		"date":     mustDate("2006-01-02", "2021-01-02"),
		"datetime": mustDate("2006-01-02T15:04:05", "2021-01-02T00:00:00"),
	}
	goValueCmp := &blocks.StreamBlockValue{
		Blocks: []*blocks.StreamBlockData{
			{ID: uuid.Nil, Type: "person", Data: goItem0, Order: 0},
			{ID: uuid.Nil, Type: "person", Data: goItem1, Order: 1},
		},
	}

	t.Run("ValueFromDataDict", func(t *testing.T) {
		data, errs := b.ValueFromDataDict(context.Background(), raw, nil, "test_stream")
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
		if data == nil {
			t.Fatalf("expected data, got nil")
		}

		v, ok := data.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected *StreamBlockValue, got %T", data)
		}

		if len(v.Blocks) != len(formValueCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(formValueCmp.Blocks), len(v.Blocks))
		}

		for i := range v.Blocks {
			got := v.Blocks[i]
			exp := formValueCmp.Blocks[i]
			if got.ID != exp.ID || got.Type != exp.Type || got.Order != exp.Order {
				t.Errorf("meta mismatch at %d: got (id=%s,type=%s,order=%d) exp (id=%s,type=%s,order=%d)",
					i, got.ID, got.Type, got.Order, exp.ID, exp.Type, exp.Order)
			}
			if !reflect.DeepEqual(got.Data, exp.Data) {
				t.Errorf("data mismatch at %d: exp %v, got %v", i, exp.Data, got.Data)
			}
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		gotAny, err := b.ValueToGo(formValueCmp)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		got, ok := gotAny.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected StreamBlockValue, got %T", gotAny)
		}

		if len(got.Blocks) != len(goValueCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(goValueCmp.Blocks), len(got.Blocks))
		}

		for i := range got.Blocks {
			g := got.Blocks[i]
			e := goValueCmp.Blocks[i]
			if g.ID != e.ID || g.Type != e.Type || g.Order != e.Order {
				t.Errorf("meta mismatch at %d: got (id=%s,type=%s,order=%d) exp (id=%s,type=%s,order=%d)",
					i, g.ID, g.Type, g.Order, e.ID, e.Type, e.Order)
			}
			if !reflect.DeepEqual(g.Data, e.Data) {
				t.Errorf("data mismatch at %d: exp %v, got %v", i, e.Data, g.Data)
			}
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		gotAny := b.ValueToForm(goValueCmp)
		got, ok := gotAny.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected StreamBlockValue, got %T", gotAny)
		}

		if len(got.Blocks) != len(formValueCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(formValueCmp.Blocks), len(got.Blocks))
		}

		for i := range got.Blocks {
			g := got.Blocks[i]
			e := formValueCmp.Blocks[i]
			if g.ID != e.ID || g.Type != e.Type || g.Order != e.Order {
				t.Errorf("meta mismatch at %d: got (id=%s,type=%s,order=%d) exp (id=%s,type=%s,order=%d)",
					i, g.ID, g.Type, g.Order, e.ID, e.Type, e.Order)
			}
			if !reflect.DeepEqual(g.Data, e.Data) {
				t.Errorf("data mismatch at %d: exp %v, got %v", i, e.Data, g.Data)
			}
		}
	})

	t.Run("ConversionsEqual", func(t *testing.T) {
		goAny, err := b.ValueToGo(formValueCmp)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		formAgain := b.ValueToForm(goAny)

		// We expect the final form representation to match the original "formValueCmp".
		got, ok := formAgain.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected StreamBlockValue, got %T", formAgain)
		}
		if len(got.Blocks) != len(formValueCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(formValueCmp.Blocks), len(got.Blocks))
		}
		for i := range got.Blocks {
			if !reflect.DeepEqual(got.Blocks[i], formValueCmp.Blocks[i]) {
				t.Errorf("roundtrip mismatch at %d: exp %v, got %v", i, formValueCmp.Blocks[i], got.Blocks[i])
			}
		}
	})
}

// Build a StreamBlock that mixes several child block types.
func NewRichStreamBlock() *blocks.StreamBlock {
	sb := blocks.NewStreamBlock()
	sb.SetName("test_stream")

	// Rich set of children
	sb.AddField("person", NewSimpleStructBlock()) // from your earlier helper
	sb.AddField("title", blocks.CharBlock())
	sb.AddField("age", blocks.NumberBlock())
	sb.AddField("email", blocks.EmailBlock())
	sb.AddField("born", blocks.DateBlock())
	sb.AddField("when", blocks.DateTimeBlock())

	return sb
}

func TestStreamBlock_RichChildren(t *testing.T) {
	t.Helper()

	mustAddr := func(s string) *mail.Address {
		a, err := mail.ParseAddress(s)
		if err != nil {
			t.Fatalf("parse address %q: %v", s, err)
		}
		return a
	}
	mustTime := func(layout, s string) time.Time {
		d, err := time.Parse(layout, s)
		if err != nil {
			t.Fatalf("parse time %q: %v", s, err)
		}
		return d
	}

	b := NewRichStreamBlock()

	// ----- Raw form (what form posts look like) -----
	// We'll include:
	//  0: title (char)
	//  1: person (struct)
	//  2: age (number)
	//  3: email (email)
	//  4: born (date)
	//  5: when (datetime)
	raw := url.Values{
		"test_stream--total": {"6"},

		"test_stream-id-0":    {uuid.Nil.String()},
		"test_stream-order-0": {"0"},
		"test_stream-0--type": {"title"},
		"test_stream-0":       {"Hello world"},

		"test_stream-id-1":       {uuid.Nil.String()},
		"test_stream-order-1":    {"1"},
		"test_stream-1--type":    {"person"},
		"test_stream-1-name":     {"John Doe"},
		"test_stream-1-age":      {"30"},
		"test_stream-1-email":    {"john@localhost"},
		"test_stream-1-password": {"hunter2"},
		"test_stream-1-date":     {"2022-02-02"},
		"test_stream-1-datetime": {"2022-02-02T10:30:00"},

		"test_stream-id-2":    {uuid.Nil.String()},
		"test_stream-order-2": {"2"},
		"test_stream-2--type": {"age"},
		"test_stream-2":       {"42"},

		"test_stream-id-3":    {uuid.Nil.String()},
		"test_stream-order-3": {"3"},
		"test_stream-3--type": {"email"},
		"test_stream-3":       {"dev@localhost"},

		"test_stream-id-4":    {uuid.Nil.String()},
		"test_stream-order-4": {"4"},
		"test_stream-4--type": {"born"},
		"test_stream-4":       {"1999-12-31"},

		"test_stream-id-5":    {uuid.Nil.String()},
		"test_stream-order-5": {"5"},
		"test_stream-5--type": {"when"},
		"test_stream-5":       {"2025-01-15T23:59:59"},
	}

	// ----- Expected "form" (stringy) value -----
	formPerson := map[string]interface{}{
		"name":     "John Doe",
		"age":      "30",
		"email":    "john@localhost",
		"password": "hunter2",
		"date":     "2022-02-02",
		"datetime": "2022-02-02T10:30:00",
	}
	formCmp := &blocks.StreamBlockValue{
		Blocks: []*blocks.StreamBlockData{
			{ID: uuid.Nil, Type: "title", Order: 0, Data: "Hello world"},
			{ID: uuid.Nil, Type: "person", Order: 1, Data: formPerson},
			{ID: uuid.Nil, Type: "age", Order: 2, Data: "42"},
			{ID: uuid.Nil, Type: "email", Order: 3, Data: "dev@localhost"},
			{ID: uuid.Nil, Type: "born", Order: 4, Data: "1999-12-31"},
			{ID: uuid.Nil, Type: "when", Order: 5, Data: "2025-01-15T23:59:59"},
		},
	}

	// ----- Expected Go-typed value -----
	goPerson := map[string]interface{}{
		"name":     "John Doe",
		"age":      30,
		"email":    mustAddr("john@localhost"),
		"password": "hunter2",
		"date":     mustTime("2006-01-02", "2022-02-02"),
		"datetime": mustTime("2006-01-02T15:04:05", "2022-02-02T10:30:00"),
	}
	goCmp := &blocks.StreamBlockValue{
		Blocks: []*blocks.StreamBlockData{
			{ID: uuid.Nil, Type: "title", Order: 0, Data: "Hello world"},
			{ID: uuid.Nil, Type: "person", Order: 1, Data: goPerson},
			{ID: uuid.Nil, Type: "age", Order: 2, Data: 42},
			{ID: uuid.Nil, Type: "email", Order: 3, Data: mustAddr("dev@localhost")},
			{ID: uuid.Nil, Type: "born", Order: 4, Data: mustTime("2006-01-02", "1999-12-31")},
			{ID: uuid.Nil, Type: "when", Order: 5, Data: mustTime("2006-01-02T15:04:05", "2025-01-15T23:59:59")},
		},
	}

	t.Run("ValueFromDataDict", func(t *testing.T) {
		gotAny, errs := b.ValueFromDataDict(context.Background(), raw, nil, "test_stream")
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
		got, ok := gotAny.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected *StreamBlockValue, got %T", gotAny)
		}

		if len(got.Blocks) != len(formCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(formCmp.Blocks), len(got.Blocks))
		}
		for i := range got.Blocks {
			gb, eb := got.Blocks[i], formCmp.Blocks[i]
			if gb.ID != eb.ID || gb.Type != eb.Type || gb.Order != eb.Order {
				t.Errorf("meta mismatch #%d: got(id=%s,type=%s,order=%d) exp(id=%s,type=%s,order=%d)",
					i, gb.ID, gb.Type, gb.Order, eb.ID, eb.Type, eb.Order)
			}
			if !reflect.DeepEqual(gb.Data, eb.Data) {
				t.Errorf("data mismatch #%d: exp %v, got %v", i, eb.Data, gb.Data)
			}
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		gotAny, err := b.ValueToGo(formCmp)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		got, ok := gotAny.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected StreamBlockValue, got %T", gotAny)
		}

		if len(got.Blocks) != len(goCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(goCmp.Blocks), len(got.Blocks))
		}
		for i := range got.Blocks {
			gb, eb := got.Blocks[i], goCmp.Blocks[i]
			if gb.ID != eb.ID || gb.Type != eb.Type || gb.Order != eb.Order {
				t.Errorf("meta mismatch #%d: got(id=%s,type=%s,order=%d) exp(id=%s,type=%s,order=%d)",
					i, gb.ID, gb.Type, gb.Order, eb.ID, eb.Type, eb.Order)
			}
			if !reflect.DeepEqual(gb.Data, eb.Data) {
				t.Errorf("data mismatch #%d: exp %v, got %v", i, eb.Data, gb.Data)
			}
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		gotAny := b.ValueToForm(goCmp)
		got, ok := gotAny.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected StreamBlockValue, got %T", gotAny)
		}

		if len(got.Blocks) != len(formCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(formCmp.Blocks), len(got.Blocks))
		}
		for i := range got.Blocks {
			gb, eb := got.Blocks[i], formCmp.Blocks[i]
			if gb.ID != eb.ID || gb.Type != eb.Type || gb.Order != eb.Order {
				t.Errorf("meta mismatch #%d: got(id=%s,type=%s,order=%d) exp(id=%s,type=%s,order=%d)",
					i, gb.ID, gb.Type, gb.Order, eb.ID, eb.Type, eb.Order)
			}
			if !reflect.DeepEqual(gb.Data, eb.Data) {
				t.Errorf("data mismatch #%d: exp %v, got %v", i, eb.Data, gb.Data)
			}
		}
	})

	t.Run("ConversionsEqual", func(t *testing.T) {
		goAny, err := b.ValueToGo(formCmp)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		formAgainAny := b.ValueToForm(goAny)
		formAgain, ok := formAgainAny.(*blocks.StreamBlockValue)
		if !ok {
			t.Fatalf("expected StreamBlockValue, got %T", formAgainAny)
		}

		if len(formAgain.Blocks) != len(formCmp.Blocks) {
			t.Fatalf("expected %d blocks, got %d", len(formCmp.Blocks), len(formAgain.Blocks))
		}
		for i := range formAgain.Blocks {
			if !reflect.DeepEqual(formAgain.Blocks[i], formCmp.Blocks[i]) {
				t.Errorf("roundtrip mismatch #%d: exp %v, got %v", i, formCmp.Blocks[i], formAgain.Blocks[i])
			}
		}
	})
}

func TestStreamBlock_OmittedAndErrors(t *testing.T) {
	sb := NewRichStreamBlock()

	// Case 1: Omitted (no --total)
	omitted := sb.ValueOmittedFromData(context.Background(), url.Values{}, nil, "test_stream")
	if !omitted {
		t.Errorf("expected omitted when --total missing")
	}

	// Case 2: Present but marked deleted
	raw := url.Values{
		"test_stream--total":     {"1"},
		"test_stream-id-0":       {uuid.Nil.String()},
		"test_stream-order-0":    {"0"},
		"test_stream-0--type":    {"title"},
		"test_stream-0--deleted": {"1"},
		"test_stream-0":          {"Hello"},
	}
	omitted = sb.ValueOmittedFromData(context.Background(), raw, nil, "test_stream")
	if omitted {
		t.Errorf("expected not omitted when an item is present but deleted (still counted)")
	}

	// Case 3: Error paths: missing type, invalid uuid, invalid order, unknown type (should be skipped without failing)
	rawErr := url.Values{
		"test_stream--total": {"4"},

		// 0: missing --type -> error
		"test_stream-id-0":    {uuid.Nil.String()},
		"test_stream-order-0": {"0"},
		"test_stream-0":       {"Hello"},

		// 1: invalid UUID -> error
		"test_stream-id-1":    {"not-a-uuid"},
		"test_stream-order-1": {"1"},
		"test_stream-1--type": {"title"},
		"test_stream-1":       {"Hello2"},

		// 2: invalid order -> error
		"test_stream-id-2":    {uuid.Nil.String()},
		"test_stream-order-2": {"NaN"},
		"test_stream-2--type": {"title"},
		"test_stream-2":       {"Hello3"},

		// 3: unknown type -> warning and skip (no error)
		"test_stream-id-3":    {uuid.Nil.String()},
		"test_stream-order-3": {"3"},
		"test_stream-3--type": {"does_not_exist"},
	}

	_, errs := sb.ValueFromDataDict(context.Background(), rawErr, nil, "test_stream")
	if len(errs) == 0 {
		t.Fatalf("expected aggregated errors, got none")
	}
}
