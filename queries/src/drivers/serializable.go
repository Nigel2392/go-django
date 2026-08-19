package drivers

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/internal/django_reflect"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_json_nullLiteral = []byte("null")

	_ json.Marshaler   = (*Value[any])(nil)
	_ json.Unmarshaler = (*Value[any])(nil)

	_ encoding.TextMarshaler   = (*Value[any])(nil)
	_ encoding.TextUnmarshaler = (*Value[any])(nil)
)

type valueJSON struct {
	GOType string          `json:"go_type"`
	Value  json.RawMessage `json:"value,omitempty"`
}

// The generic type is not used tor marshalling or unmarshalling,
// it is only provided for developer QOL
type Value[T any] struct {
	V T
}

func (s *Value[T]) IsZero() bool {
	av := any(s.V)
	if isZeroer, ok := av.(django_reflect.IsZeroer); ok {
		return isZeroer.IsZero()
	}

	return av == nil
}

func (s Value[T]) Value() (driver.Value, error) {
	if any(s.V) == nil {
		return nil, nil
	}

	//	if _, ok := any(*new(T)).(driver.Valuer); ok {
	//		return any(s.V).(driver.Valuer).Value()
	//	}

	str, err := json.Marshal(s)
	return string(str), err
}

func (s *Value[T]) Scan(src any) error {
	if src == nil {
		return nil
	}

	//	if _, ok := any(*new(T)).(sql.Scanner); ok {
	//		if isNil(reflect.ValueOf(s.V)) {
	//			s.V = newValue[T]()
	//		}
	//
	//		return any(s.V).(sql.Scanner).Scan(src)
	//	}

	var b io.Reader
	switch src := src.(type) {
	case []byte:
		b = bytes.NewReader(src)
	case string:
		b = strings.NewReader(src)
	case []rune:
		b = strings.NewReader(string(src))
	default:
		return errors.TypeMismatch.Wrapf(
			"%T is not of type ~string",
			src,
		)
	}

	var dec = json.NewDecoder(b)
	dec.DisallowUnknownFields()
	return dec.Decode(s)
}

func isNil(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return rv.IsNil()
	}
	return false
}

func (s Value[T]) MarshalJSON() (bytes []byte, err error) {
	rv := reflect.ValueOf(s.V)
	if isNil(rv) {
		return _json_nullLiteral, nil
	}

	var v any = s.V
	if val, ok := v.(driver.Valuer); ok {
		v, err = val.Value()
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve driver value: %w", err)
	}

	bytes, err = json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("could not marshal value: %w", err)
	}

	return json.Marshal(&valueJSON{
		GOType: StringForType(reflect.TypeOf(s.V)),
		Value:  bytes,
	})
}

func (s *Value[T]) UnmarshalJSON(b []byte) (err error) {
	if bytes.Equal(b, _json_nullLiteral) {
		return nil
	}

	var val = new(valueJSON)
	if err = json.Unmarshal(b, val); err != nil {
		return err
	}

	var isNullLiteral = bytes.Equal(val.Value, _json_nullLiteral)
	if isNullLiteral {
		return nil
	}

	var goType, ok = TypeFromString(val.GOType)
	if !ok {
		return errors.NotExists.Wrapf(
			"%q does not have a GO type registered",
			val.GOType,
		)
	}

	var isPtr = goType.Kind() == reflect.Pointer
	if isPtr {
		goType = goType.Elem()
	}

	var jsonData = val.Value
	var scanToVal = reflect.New(goType)
	var scanTo = scanToVal.Interface()

	switch sc := scanTo.(type) {
	case json.Unmarshaler:
	case sql.Scanner:
		var dst = newForSrcPtr(scanTo, goType)
		if err := json.Unmarshal(jsonData, dst.Interface()); err != nil {
			return fmt.Errorf("could not unmarshal value %T: %w %s", scanTo, err, string(jsonData))
		}

		if err := sc.Scan(dst.Elem().Interface()); err != nil {
			return fmt.Errorf("could not scan value %T: %w %s", scanTo, err, string(jsonData))
		}

		goto setValue
	}

	if err = json.Unmarshal(jsonData, scanTo); err != nil {
		return fmt.Errorf("could not unmarshal value %T: %w %s", scanTo, err, string(jsonData))
	}

setValue:
	return s.setIface(scanToVal, isPtr)
}

func (s Value[T]) MarshalText() (text []byte, err error) {
	var buf = new(bytes.Buffer)
	buf.WriteString(StringForType(reflect.TypeOf(s.V)))
	buf.WriteString(":")

	if any(s.V) == nil {
		buf.Write(_json_nullLiteral)
	}

	switch v := any(s.V).(type) {
	case nil:
		buf.Write(_json_nullLiteral)
	case encoding.TextAppender:
		return v.AppendText(buf.Bytes())
	case encoding.TextMarshaler:
		text, err = v.MarshalText()
	case encoding.BinaryAppender:
		return v.AppendBinary(buf.Bytes())
	case encoding.BinaryMarshaler:
		text, err = v.MarshalBinary()
	case attrs.Definer:
		pk := attrs.PrimaryKey(context.Background(), s.V)
		text = []byte(attrs.ToString(pk))
	default:
		t, err := json.Marshal(s.V)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(t, _json_nullLiteral) {
			buf.Write(_json_nullLiteral)
			return buf.Bytes(), nil
		}

		buf.Grow(base64.StdEncoding.EncodedLen(len(t)))
		enc := base64.NewEncoder(base64.StdEncoding, buf)
		enc.Write(t)
		err = enc.Close()
		return buf.Bytes(), err
	}

	buf.Write(text)

	return buf.Bytes(), err
}

func (s *Value[T]) UnmarshalText(text []byte) (err error) {
	var textParts = bytes.SplitN(text, []byte(":"), 2)
	if len(textParts) != 2 {

	}

	var (
		textTyp = textParts[0]
		textVal = textParts[1]
	)

	var goType, ok = TypeFromString(string(textTyp))
	if !ok {
		return errors.NotExists.Wrapf(
			"%q does not have a GO type registered",
			textTyp,
		)
	}

	var isPtr = goType.Kind() == reflect.Pointer
	if isPtr {
		goType = goType.Elem()
	}

	var scanToVal = reflect.New(goType)
	if bytes.Equal(textVal, _json_nullLiteral) {
		return s.setIface(scanToVal, isPtr)
	}

	var scanTo = scanToVal.Interface()
	switch scanner := scanTo.(type) {
	case encoding.TextUnmarshaler:
		err = scanner.UnmarshalText(textVal)
	case encoding.BinaryUnmarshaler:
		err = scanner.UnmarshalBinary(text)
	default:
		var n int
		textValDecoded := make([]byte, base64.StdEncoding.DecodedLen(len(textVal)))
		n, err = base64.StdEncoding.Decode(textValDecoded, textVal)
		if err != nil {
			return errors.ValueError.WithCause(err)
		}

		err = json.Unmarshal(textValDecoded[:n], scanTo)
	}

	if err != nil {
		return errors.ValueError.WithCause(err)
	}

	return s.setIface(scanToVal, isPtr)
}

func (s *Value[T]) setIface(scanToVal reflect.Value, isPtr bool) error {
	var v any
	var ok bool
	if isPtr {
		v = scanToVal.Interface()
	} else {
		v = scanToVal.Elem().Interface()
	}

	// skip untyped nil, ensure typed nil executes below cast
	if v == nil && reflect.TypeOf(v) == nil {
		return nil
	}

	s.V, ok = v.(T)
	if !ok {
		return fmt.Errorf("%T is not of type %s", v, reflect.TypeFor[T]())
	}

	return nil
}

var (
	__typ_int   = reflect.TypeFor[int64]()
	__typ_uint  = reflect.TypeFor[uint64]()
	__typ_float = reflect.TypeFor[float64]()
	__typ_str   = reflect.TypeFor[string]()
	__typ_bool  = reflect.TypeFor[bool]()
)

func newPtr[T any]() reflect.Value {
	var t = reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return reflect.New(t)
}

func newForSrcPtr(orig any, r reflect.Type) reflect.Value {
	switch orig.(type) {
	case *any:
		return newPtr[any]()

	case *[]byte:
		var b = make([]byte, 0)
		return reflect.ValueOf(&b)

	case *sql.NullString:
		return newPtr[string]()

	case *sql.NullInt64:
		return newPtr[int64]()

	case *sql.NullInt32:
		return newPtr[int32]()

	case *sql.NullInt16:
		return newPtr[int16]()

	case *sql.NullByte:
		return newPtr[byte]()

	case *sql.NullFloat64:
		return newPtr[float64]()

	case *sql.NullBool:
		return newPtr[bool]()

	case *sql.NullTime:
		return newPtr[time.Time]()
	}

	if r.Kind() == reflect.Struct && r.NumField() == 2 {
		fld0 := r.Field(0)
		fld1 := r.Field(1)

		if fld0.Name == "V" && fld1.Name == "Valid" {
			rv := reflect.ValueOf(orig).Elem()
			n := reflect.New(fld0.Type)
			rv.Field(0).Set(n.Elem())
			return newForSrcPtr(n, fld0.Type)
		}
	}

	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.New(__typ_int)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return reflect.New(__typ_uint)

	case reflect.Float32, reflect.Float64:
		return reflect.New(__typ_float)

	case reflect.String:
		return reflect.New(__typ_str)

	case reflect.Bool:
		return reflect.New(__typ_bool)

	case reflect.Slice:
		switch r.Elem().Kind() {
		case reflect.Uint8:
			var b = make([]byte, 0)
			return reflect.ValueOf(&b)

		case reflect.Int32:
			var b = make([]rune, 0)
			return reflect.ValueOf(&b)
		}

		return reflect.MakeSlice(r, 0, 0)
	}

	typ, ok := dbtype.For(r)
	if !ok {
		return reflect.New(r)
	}

	switch typ {
	case dbtype.Text, dbtype.String, dbtype.Char,
		dbtype.Decimal, dbtype.UUID, dbtype.ULID,
		dbtype.JSON:
		return newPtr[string]()

	case dbtype.Int:
		return newPtr[int64]()

	case dbtype.Uint:
		return newPtr[uint64]()

	case dbtype.Float:
		return newPtr[float64]()

	case dbtype.Bool:
		return newPtr[bool]()

	case dbtype.Bytes, dbtype.BLOB:
		var b = make([]byte, 0)
		return reflect.ValueOf(&b)

	case dbtype.Timestamp, dbtype.LocalTime, dbtype.DateTime:
		return reflect.ValueOf(&time.Time{})
	}

	return reflect.New(r)
}
