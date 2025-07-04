package drivers_test

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// dummy type for RegisterGoType test
type Dummy struct{}

func TestRegisterGoType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when registering nil type, got none")
		}
	}()

	// should panic with nil
	drivers.RegisterGoType("nilType", nil)

	// register a real type
	typ := reflect.TypeOf(Dummy{})
	drivers.RegisterGoType("Dummy", typ)

	got, ok := drivers.TypeFromString("github.com/Nigel2392/go-django/queries/src/drivers_test.Dummy")
	if !ok || got != typ {
		t.Errorf("expected %v, got %v", typ, got)
	}
}

func TestStringForType(t *testing.T) {
	var val *int
	expected := "*int"
	if got := drivers.StringForType(val); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	typ := reflect.TypeOf(&Dummy{})
	expected = "*github.com/Nigel2392/go-django/queries/src/drivers_test.Dummy"
	if got := drivers.StringForType(typ); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestTypeFromString(t *testing.T) {
	dbtype.Lock() // required by internal guard
	typ := reflect.TypeOf(decimal.Decimal{})
	drivers.RegisterGoType(drivers.StringForType(typ), typ)

	str := drivers.StringForType(typ)
	got, ok := drivers.TypeFromString(str)
	if !ok {
		t.Errorf("type not found: %s", str)
	}
	if got != typ {
		t.Errorf("expected %v, got %v", typ, got)
	}
}

func TestRegisteredDBTypeMappings(t *testing.T) {
	tests := []struct {
		val  any
		want dbtype.Type
	}{
		{drivers.Text(""), dbtype.Text},
		{drivers.String(""), dbtype.String},
		{drivers.Char(""), dbtype.Char},
		{drivers.Int(0), dbtype.Int},
		{drivers.Uint(0), dbtype.Uint},
		{drivers.Float(0.0), dbtype.Float},
		{drivers.Bool(false), dbtype.Bool},
		{drivers.Bytes(nil), dbtype.Bytes},
		{drivers.BLOB(nil), dbtype.BLOB},
		{drivers.UUID(uuid.UUID{}), dbtype.UUID},
		{drivers.Timestamp{}, dbtype.Timestamp},
		{drivers.LocalTime{}, dbtype.LocalTime},
		{drivers.DateTime{}, dbtype.DateTime},
		{drivers.Email{}, dbtype.String},
		{decimal.Decimal{}, dbtype.Decimal},

		{new(any), dbtype.JSON},
		{*new(string), dbtype.String},
		{*new([]byte), dbtype.Bytes},
		{*new(int), dbtype.Int},
		{*new(int8), dbtype.Int},
		{*new(int16), dbtype.Int},
		{*new(int32), dbtype.Int},
		{*new(int64), dbtype.Int},
		{*new(uint), dbtype.Uint},
		{*new(uint8), dbtype.Uint},
		{*new(uint16), dbtype.Uint},
		{*new(uint32), dbtype.Uint},
		{*new(uint64), dbtype.Uint},
		{*new(float32), dbtype.Float},
		{*new(float64), dbtype.Float},
		{*new(bool), dbtype.Bool},
		{*new(uuid.UUID), dbtype.UUID},
		{*new(time.Time), dbtype.DateTime},

		{sql.NullString{}, dbtype.Text},
		{sql.NullFloat64{}, dbtype.Float},
		{sql.NullInt64{}, dbtype.Int},
		{sql.NullInt32{}, dbtype.Int},
		{sql.NullInt16{}, dbtype.Int},
		{sql.NullBool{}, dbtype.Bool},
		{sql.NullByte{}, dbtype.Bytes},
		{sql.NullTime{}, dbtype.DateTime},
		{decimal.NullDecimal{}, dbtype.Decimal},

		{sql.Null[drivers.Text]{}, dbtype.Text},
		{sql.Null[drivers.String]{}, dbtype.String},
		{sql.Null[drivers.Char]{}, dbtype.Char},
		{sql.Null[drivers.Int]{}, dbtype.Int},
		{sql.Null[drivers.Uint]{}, dbtype.Uint},
		{sql.Null[drivers.Float]{}, dbtype.Float},
		{sql.Null[drivers.Bool]{}, dbtype.Bool},
		{sql.Null[drivers.Bytes]{}, dbtype.Bytes},
		{sql.Null[drivers.UUID]{}, dbtype.UUID},
		{sql.Null[drivers.Timestamp]{}, dbtype.Timestamp},
		{sql.Null[drivers.LocalTime]{}, dbtype.LocalTime},
		{sql.Null[drivers.DateTime]{}, dbtype.DateTime},
		{sql.Null[drivers.Email]{}, dbtype.String},

		{sql.Null[any]{}, dbtype.JSON},
		{sql.Null[string]{}, dbtype.String},
		{sql.Null[[]byte]{}, dbtype.Bytes},
		{sql.Null[int]{}, dbtype.Int},
		{sql.Null[int8]{}, dbtype.Int},
		{sql.Null[int16]{}, dbtype.Int},
		{sql.Null[int32]{}, dbtype.Int},
		{sql.Null[int64]{}, dbtype.Int},
		{sql.Null[uint]{}, dbtype.Uint},
		{sql.Null[uint8]{}, dbtype.Uint},
		{sql.Null[uint16]{}, dbtype.Uint},
		{sql.Null[uint32]{}, dbtype.Uint},
		{sql.Null[uint64]{}, dbtype.Uint},
		{sql.Null[float32]{}, dbtype.Float},
		{sql.Null[float64]{}, dbtype.Float},
		{sql.Null[bool]{}, dbtype.Bool},
		{sql.Null[uuid.UUID]{}, dbtype.UUID},
		{sql.Null[time.Time]{}, dbtype.DateTime},
		{sql.Null[decimal.Decimal]{}, dbtype.Decimal},
	}

	for _, tt := range tests {

		var val = reflect.TypeOf(tt.val)
		if val == nil {
			t.Errorf("test value is nil: %v", tt.val)
			continue
		}

		// dereference pointers to get the underlying type
		//
		// this is necessary because dbtype.For expects the concrete type,
		// not a reference type.
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		t.Run(val.String(), func(t *testing.T) {
			got, exists := dbtype.For(val)
			if !exists {
				t.Errorf("dbtype.TYPES.For(%v) not found", val)
				return
			}

			if got != tt.want {
				t.Errorf("FromType(%v) = %v, want %v", reflect.TypeOf(tt.val), got, tt.want)
			}
		})
	}
}
