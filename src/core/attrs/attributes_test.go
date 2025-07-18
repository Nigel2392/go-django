package attrs_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type ModelTest struct {
	S           string
	I8          int8
	I16         int16
	I32         int32
	I64         int64
	U8          uint8
	U16         uint16
	U32         uint32
	U64         uint64
	F32         float32
	F64         float64
	NullBool    sql.NullBool
	NullInt64   sql.NullInt64
	NullInt32   sql.NullInt32
	NullInt16   sql.NullInt16
	NullFloat64 sql.NullFloat64
	NullString  sql.NullString
	NullTime    sql.NullTime
	B           bool
	M           map[string]interface{}
	A           []interface{}
	BT          []byte
}

func (f *ModelTest) Identifier() string {
	return "ModelTest"
}

func (f *ModelTest) FieldDefs() attrs.Definitions {
	return attrs.Define(f,
		attrs.NewField(f, "S", nil),
		attrs.NewField(f, "I8", nil),
		attrs.NewField(f, "I16", nil),
		attrs.NewField(f, "I32", nil),
		attrs.NewField(f, "I64", nil),
		attrs.NewField(f, "U8", nil),
		attrs.NewField(f, "U16", nil),
		attrs.NewField(f, "U32", nil),
		attrs.NewField(f, "U64", nil),
		attrs.NewField(f, "F32", nil),
		attrs.NewField(f, "F64", nil),
		attrs.NewField(f, "B", nil),
		attrs.NewField(f, "M", nil),
		attrs.NewField(f, "A", nil),
		attrs.NewField(f, "BT", nil),
		attrs.NewField(f, "NullBool", nil),
		attrs.NewField(f, "NullInt64", nil),
		attrs.NewField(f, "NullInt32", nil),
		attrs.NewField(f, "NullInt16", nil),
		attrs.NewField(f, "NullFloat64", nil),
		attrs.NewField(f, "NullString", nil),
		attrs.NewField(f, "NullTime", nil),
	)
}

func now() time.Time {
	var t, _ = time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	return t
}

func TestModelGet(t *testing.T) {
	var m = &ModelTest{
		S:           "string",
		I8:          8,
		I16:         16,
		I32:         32,
		I64:         64,
		U8:          8,
		U16:         16,
		U32:         32,
		U64:         64,
		F32:         32.32,
		F64:         64.64,
		NullBool:    sql.NullBool{Bool: true, Valid: true},
		NullInt64:   sql.NullInt64{Int64: 64, Valid: true},
		NullInt32:   sql.NullInt32{Int32: 32, Valid: true},
		NullInt16:   sql.NullInt16{Int16: 16, Valid: true},
		NullFloat64: sql.NullFloat64{Float64: 64.64, Valid: true},
		NullString:  sql.NullString{String: "string", Valid: true},
		NullTime:    sql.NullTime{Time: now(), Valid: true},
		B:           true,
		M:           map[string]interface{}{"key": "value"},
		A:           []interface{}{"a", "b", "c"},
		BT:          []byte("byte"),
	}

	var (
		S           = attrs.Get[string](m, "S")
		I8          = attrs.Get[int8](m, "I8")
		I16         = attrs.Get[int16](m, "I16")
		I32         = attrs.Get[int32](m, "I32")
		I64         = attrs.Get[int64](m, "I64")
		U8          = attrs.Get[uint8](m, "U8")
		U16         = attrs.Get[uint16](m, "U16")
		U32         = attrs.Get[uint32](m, "U32")
		U64         = attrs.Get[uint64](m, "U64")
		F32         = attrs.Get[float32](m, "F32")
		F64         = attrs.Get[float64](m, "F64")
		NullBool    = attrs.Get[sql.NullBool](m, "NullBool")
		NullInt64   = attrs.Get[sql.NullInt64](m, "NullInt64")
		NullInt32   = attrs.Get[sql.NullInt32](m, "NullInt32")
		NullInt16   = attrs.Get[sql.NullInt16](m, "NullInt16")
		NullFloat64 = attrs.Get[sql.NullFloat64](m, "NullFloat64")
		NullString  = attrs.Get[sql.NullString](m, "NullString")
		NullTime    = attrs.Get[sql.NullTime](m, "NullTime")
		B           = attrs.Get[bool](m, "B")
		M           = attrs.Get[map[string]interface{}](m, "M")
		A           = attrs.Get[[]interface{}](m, "A")
		BT          = attrs.Get[[]byte](m, "BT")

		S_INTERFACE           = attrs.Get[any](m, "S")
		I8_INTERFACE          = attrs.Get[any](m, "I8")
		I16_INTERFACE         = attrs.Get[any](m, "I16")
		I32_INTERFACE         = attrs.Get[any](m, "I32")
		I64_INTERFACE         = attrs.Get[any](m, "I64")
		U8_INTERFACE          = attrs.Get[any](m, "U8")
		U16_INTERFACE         = attrs.Get[any](m, "U16")
		U32_INTERFACE         = attrs.Get[any](m, "U32")
		U64_INTERFACE         = attrs.Get[any](m, "U64")
		F32_INTERFACE         = attrs.Get[any](m, "F32")
		F64_INTERFACE         = attrs.Get[any](m, "F64")
		B_INTERFACE           = attrs.Get[any](m, "B")
		NullBool_INTERFACE    = attrs.Get[any](m, "NullBool")
		NullInt64_INTERFACE   = attrs.Get[any](m, "NullInt64")
		NullInt32_INTERFACE   = attrs.Get[any](m, "NullInt32")
		NullInt16_INTERFACE   = attrs.Get[any](m, "NullInt16")
		NullFloat64_INTERFACE = attrs.Get[any](m, "NullFloat64")
		NullString_INTERFACE  = attrs.Get[any](m, "NullString")
		NullTime_INTERFACE    = attrs.Get[any](m, "NullTime")
	)

	if S != S_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", S_INTERFACE, S)
	}
	if I8 != I8_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", I8_INTERFACE, I8)
	}
	if I16 != I16_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", I16_INTERFACE, I16)
	}
	if I32 != I32_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", I32_INTERFACE, I32)
	}
	if I64 != I64_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", I64_INTERFACE, I64)
	}
	if U8 != U8_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", U8_INTERFACE, U8)
	}
	if U16 != U16_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", U16_INTERFACE, U16)
	}
	if U32 != U32_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", U32_INTERFACE, U32)
	}
	if U64 != U64_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", U64_INTERFACE, U64)
	}
	if F32 != F32_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", F32_INTERFACE, F32)
	}
	if F64 != F64_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", F64_INTERFACE, F64)
	}
	if B != B_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", B_INTERFACE, B)
	}
	if NullBool != NullBool_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullBool_INTERFACE, NullBool)
	}
	if NullInt64 != NullInt64_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullInt64_INTERFACE, NullInt64)
	}
	if NullInt32 != NullInt32_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullInt32_INTERFACE, NullInt32)
	}
	if NullInt16 != NullInt16_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullInt16_INTERFACE, NullInt16)
	}
	if NullFloat64 != NullFloat64_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullFloat64_INTERFACE, NullFloat64)
	}
	if NullString != NullString_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullString_INTERFACE, NullString)
	}
	if NullTime != NullTime_INTERFACE {
		t.Errorf("Interface %v does not equal regular value %v", NullTime_INTERFACE, NullTime)
	}

	if S != "string" {
		t.Errorf("expected %q, got %q", "string", S)
	}

	if I8 != 8 {
		t.Errorf("expected %d, got %d", 8, I8)
	}

	if I16 != 16 {
		t.Errorf("expected %d, got %d", 16, I16)
	}

	if I32 != 32 {
		t.Errorf("expected %d, got %d", 32, I32)
	}

	if I64 != 64 {
		t.Errorf("expected %d, got %d", 64, I64)
	}

	if U8 != 8 {
		t.Errorf("expected %d, got %d", 8, U8)
	}

	if U16 != 16 {
		t.Errorf("expected %d, got %d", 16, U16)
	}

	if U32 != 32 {
		t.Errorf("expected %d, got %d", 32, U32)
	}

	if U64 != 64 {
		t.Errorf("expected %d, got %d", 64, U64)
	}

	if F32 != 32.32 {
		t.Errorf("expected %f, got %f", 32.32, F32)
	}

	if F64 != 64.64 {
		t.Errorf("expected %f, got %f", 64.64, F64)
	}

	if B != true {
		t.Errorf("expected %t, got %t", true, B)
	}

	if M["key"] != "value" {
		t.Errorf("expected %q, got %q", "value", M["key"])
	}

	if len(A) != 3 {
		t.Errorf("expected %d, got %d", 3, len(A))
	}

	if string(BT) != "byte" {
		t.Errorf("expected %q, got %q", "byte", string(BT))
	}

	if NullBool.Bool != true {
		t.Errorf("expected %t, got %t", true, NullBool.Bool)
	}

	if NullInt64.Int64 != 64 {
		t.Errorf("expected %d, got %d", 64, NullInt64.Int64)
	}

	if NullInt32.Int32 != 32 {
		t.Errorf("expected %d, got %d", 32, NullInt32.Int32)
	}

	if NullInt16.Int16 != 16 {
		t.Errorf("expected %d, got %d", 16, NullInt16.Int16)
	}

	if NullFloat64.Float64 != 64.64 {
		t.Errorf("expected %f, got %f", 64.64, NullFloat64.Float64)
	}

	if NullString.String != "string" {
		t.Errorf("expected %q, got %q", "string", NullString.String)
	}
}

func TestModelSet(t *testing.T) {
	var m = &ModelTest{
		S:           "string",
		I8:          8,
		I16:         16,
		I32:         32,
		I64:         64,
		U8:          8,
		U16:         16,
		U32:         32,
		U64:         64,
		F32:         32.32,
		F64:         64.64,
		NullBool:    sql.NullBool{Bool: true, Valid: true},
		NullInt64:   sql.NullInt64{Int64: 64, Valid: true},
		NullInt32:   sql.NullInt32{Int32: 32, Valid: true},
		NullInt16:   sql.NullInt16{Int16: 16, Valid: true},
		NullFloat64: sql.NullFloat64{Float64: 64.64, Valid: true},
		NullString:  sql.NullString{String: "string", Valid: true},
		NullTime:    sql.NullTime{Time: now(), Valid: true},
		B:           true,
		M:           map[string]interface{}{"key": "value"},
		A:           []interface{}{"a", "b", "c"},
		BT:          []byte("byte"),
	}

	attrs.Set(m, "S", "new string")
	attrs.Set(m, "I8", 88)
	attrs.Set(m, "I16", 1616)
	attrs.Set(m, "I32", 3232)
	attrs.Set(m, "I64", 6464)
	attrs.Set(m, "U8", 88)
	attrs.Set(m, "U16", 0)
	attrs.Set(m, "U32", 3232)
	attrs.Set(m, "U64", 6464)
	attrs.Set(m, "F32", 32.3232)
	attrs.Set(m, "F64", 64.6464)
	attrs.Set(m, "NullBool", sql.NullBool{Bool: false, Valid: false})
	attrs.Set(m, "NullInt64", sql.NullInt64{Int64: 6464, Valid: true})
	attrs.Set(m, "NullInt32", sql.NullInt32{Int32: 3232, Valid: true})
	attrs.Set(m, "NullInt16", sql.NullInt16{Int16: 1616, Valid: true})
	attrs.Set(m, "NullFloat64", sql.NullFloat64{Float64: 64.6464, Valid: true})
	attrs.Set(m, "NullString", sql.NullString{String: "new string", Valid: true})
	attrs.Set(m, "NullTime", sql.NullTime{Time: now(), Valid: true})
	attrs.Set(m, "B", false)
	attrs.Set(m, "M", map[string]interface{}{"new key": "new value"})
	attrs.Set(m, "A", []interface{}{"x", "y", "z"})
	attrs.Set(m, "BT", []byte("new byte"))

	var assertEqual = func(t *testing.T, a, b interface{}) {
		if a != b {
			t.Errorf("expected %v, got %v", b, a)
		}
	}

	assertEqual(t, m.S, "new string")
	assertEqual(t, m.I8, int8(88))
	assertEqual(t, m.I16, int16(1616))
	assertEqual(t, m.I32, int32(3232))
	assertEqual(t, m.I64, int64(6464))
	assertEqual(t, m.U8, uint8(88))
	assertEqual(t, m.U16, uint16(0))
	assertEqual(t, m.U32, uint32(3232))
	assertEqual(t, m.U64, uint64(6464))
	assertEqual(t, m.F32, float32(32.3232))
	assertEqual(t, m.F64, float64(64.6464))
	assertEqual(t, m.NullBool.Bool, false)
	assertEqual(t, m.NullInt64.Int64, int64(6464))
	assertEqual(t, m.NullInt32.Int32, int32(3232))
	assertEqual(t, m.NullInt16.Int16, int16(1616))
	assertEqual(t, m.NullFloat64.Float64, float64(64.6464))
	assertEqual(t, m.NullString.String, "new string")
	assertEqual(t, m.NullTime.Time, now())
	assertEqual(t, m.B, false)
	assertEqual(t, m.M["new key"], "new value")
	assertEqual(t, len(m.A), 3)
	assertEqual(t, string(m.BT), "new byte")
}

type ModelTestReadOnly struct {
	Name string
}

func (f *ModelTestReadOnly) FieldDefs() attrs.Definitions {
	return attrs.Define(f,
		attrs.NewField(f, "Name", &attrs.FieldConfig{ReadOnly: true}),
	)
}

func TestModelSetReadOnly(t *testing.T) {
	var m = &ModelTestReadOnly{
		Name: "name",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}

		if m.Name != "name" {
			t.Errorf("expected %q, got %q", "name", m.Name)
		}
	}()

	attrs.Set(m, "Name", "new name")
}

func TestModelForceSetReadOnly(t *testing.T) {
	var m = &ModelTestReadOnly{
		Name: "name",
	}

	attrs.ForceSet(m, "Name", "new name")

	if m.Name != "new name" {
		t.Errorf("expected %q, got %q", "new name", m.Name)
	}
}

type TestModelReadonlyStructTag struct {
	ID   int
	Name string `attrs:"readonly"`
}

func (f *TestModelReadonlyStructTag) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f)
}

func TestModelSetReadOnlyStructTag(t *testing.T) {
	var m = &TestModelReadonlyStructTag{
		ID:   1,
		Name: "name",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}

		if m.ID != 2 {
			t.Errorf("expected %d, got %d", 1, m.ID)
		}

		if m.Name != "name" {
			t.Errorf("expected %q, got %q", "name", m.Name)
		}
	}()

	attrs.Set(m, "ID", 2)
	attrs.Set(m, "Name", "new name")
}

func TestModelForceSetReadOnlyStructTag(t *testing.T) {
	var m = &TestModelReadonlyStructTag{
		ID:   1,
		Name: "name",
	}

	attrs.ForceSet(m, "ID", 2)
	attrs.ForceSet(m, "Name", "new name")

	if m.ID != 2 {
		t.Errorf("expected %d, got %d", 2, m.ID)
	}

	if m.Name != "new name" {
		t.Errorf("expected %q, got %q", "new name", m.Name)
	}
}

func BenchmarkUnpackFieldsFromArgs(b *testing.B) {
	var m = &ModelTest{
		S:           "string",
		I8:          8,
		I16:         16,
		I32:         32,
		I64:         64,
		U8:          8,
		U16:         16,
		U32:         32,
		U64:         64,
		F32:         32.32,
		F64:         64.64,
		NullBool:    sql.NullBool{Bool: true, Valid: true},
		NullInt64:   sql.NullInt64{Int64: 64, Valid: true},
		NullInt32:   sql.NullInt32{Int32: 32, Valid: true},
		NullInt16:   sql.NullInt16{Int16: 16, Valid: true},
		NullFloat64: sql.NullFloat64{Float64: 64.64, Valid: true},
		NullString:  sql.NullString{String: "string", Valid: true},
		NullTime:    sql.NullTime{Time: now(), Valid: true},
		B:           true,
		M:           map[string]interface{}{"key": "value"},
		A:           []interface{}{"a", "b", "c"},
	}

	b.ResetTimer()
	var fields = m.FieldDefs().Fields()
	for i := 0; i < b.N; i++ {
		var _, err = attrs.UnpackFieldsFromArgs[attrs.Definer, any](m, fields)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnpackFieldsFromArgsIter(b *testing.B) {
	var m = &ModelTest{
		S:           "string",
		I8:          8,
		I16:         16,
		I32:         32,
		I64:         64,
		U8:          8,
		U16:         16,
		U32:         32,
		U64:         64,
		F32:         32.32,
		F64:         64.64,
		NullBool:    sql.NullBool{Bool: true, Valid: true},
		NullInt64:   sql.NullInt64{Int64: 64, Valid: true},
		NullInt32:   sql.NullInt32{Int32: 32, Valid: true},
		NullInt16:   sql.NullInt16{Int16: 16, Valid: true},
		NullFloat64: sql.NullFloat64{Float64: 64.64, Valid: true},
		NullString:  sql.NullString{String: "string", Valid: true},
		NullTime:    sql.NullTime{Time: now(), Valid: true},
		B:           true,
		M:           map[string]interface{}{"key": "value"},
		A:           []interface{}{"a", "b", "c"},
	}

	b.ResetTimer()
	var fields = m.FieldDefs().Fields()
	for i := 0; i < b.N; i++ {
		var iterator = attrs.UnpackFieldsFromArgsIter[attrs.Definer, any](m, fields)
		var slice = make([]any, 0, 22)
		for field, err := range iterator {
			if err != nil {
				b.Fatal(err)
			}
			slice = append(slice, field)
		}
		if len(slice) != 22 {
			b.Fatalf("expected 22 fields, got %d", len(slice))
		}
	}
}

func BenchmarkUnpackFieldsFromArgsIterFunc(b *testing.B) {
	var m = &ModelTest{
		S:           "string",
		I8:          8,
		I16:         16,
		I32:         32,
		I64:         64,
		U8:          8,
		U16:         16,
		U32:         32,
		U64:         64,
		F32:         32.32,
		F64:         64.64,
		NullBool:    sql.NullBool{Bool: true, Valid: true},
		NullInt64:   sql.NullInt64{Int64: 64, Valid: true},
		NullInt32:   sql.NullInt32{Int32: 32, Valid: true},
		NullInt16:   sql.NullInt16{Int16: 16, Valid: true},
		NullFloat64: sql.NullFloat64{Float64: 64.64, Valid: true},
		NullString:  sql.NullString{String: "string", Valid: true},
		NullTime:    sql.NullTime{Time: now(), Valid: true},
		B:           true,
		M:           map[string]interface{}{"key": "value"},
		A:           []interface{}{"a", "b", "c"},
	}

	b.ResetTimer()
	var fields = m.FieldDefs().Fields()
	var fieldsFn = func(attrs.Definer) []attrs.Field {
		return fields
	}
	for i := 0; i < b.N; i++ {
		var iterator = attrs.UnpackFieldsFromArgsIter[attrs.Definer, any](m, fieldsFn)
		var slice = make([]any, 0, 22)
		for field, err := range iterator {
			if err != nil {
				b.Fatal(err)
			}
			slice = append(slice, field)
		}
		if len(slice) != 22 {
			b.Fatalf("expected 22 fields, got %d", len(slice))
		}
	}
}

func BenchmarkUnpackFieldsFromArgsIterNested(b *testing.B) {
	var m = &ModelTest{
		S:           "string",
		I8:          8,
		I16:         16,
		I32:         32,
		I64:         64,
		U8:          8,
		U16:         16,
		U32:         32,
		U64:         64,
		F32:         32.32,
		F64:         64.64,
		NullBool:    sql.NullBool{Bool: true, Valid: true},
		NullInt64:   sql.NullInt64{Int64: 64, Valid: true},
		NullInt32:   sql.NullInt32{Int32: 32, Valid: true},
		NullInt16:   sql.NullInt16{Int16: 16, Valid: true},
		NullFloat64: sql.NullFloat64{Float64: 64.64, Valid: true},
		NullString:  sql.NullString{String: "string", Valid: true},
		NullTime:    sql.NullTime{Time: now(), Valid: true},
		B:           true,
		M:           map[string]interface{}{"key": "value"},
		A:           []interface{}{"a", "b", "c"},
	}

	b.ResetTimer()
	var fields = m.FieldDefs().Fields()
	var fieldsFn = func(attrs.Definer) []attrs.Field {
		return fields
	}
	for i := 0; i < b.N; i++ {
		var iterator = attrs.UnpackFieldsFromArgsIter[attrs.Definer, any](m, attrs.UnpackFieldsFromArgsIter[attrs.Definer, any](m, fieldsFn))
		var slice = make([]any, 0, 22)
		for field, err := range iterator {
			if err != nil {
				b.Fatal(err)
			}
			slice = append(slice, field)
		}
		if len(slice) != 22 {
			b.Fatalf("expected 22 fields, got %d", len(slice))
		}
	}
}
