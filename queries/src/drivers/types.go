package drivers

import (
	"database/sql/driver"
	"time"
)

type (
	Text   string
	String string
	Char   string
	Int    int64
	Bool   bool
	Bytes  []byte
	Float  float64

	timeType  time.Time
	Timestamp time.Time
	LocalTime time.Time
	DateTime  time.Time
)

func (t *timeType) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		*t = timeType(v)
		return nil
	case string:
		var _t, err = time.Parse(time.RFC3339, v)
		*t = timeType(_t)
		return err
	case []byte:
		var _t, err = time.Parse(time.RFC3339, string(v))
		*t = timeType(_t)
		return err
	case int64:
		*t = timeType(time.Unix(v, 0))
	case uint64:
		*t = timeType(time.Unix(int64(v), 0))
	default:
		return nil
	}
	return nil
}

func (t Timestamp) Time() time.Time              { return time.Time(t) }
func (t Timestamp) Value() (driver.Value, error) { return t.Time(), nil }
func (t Timestamp) Scan(value any) error         { return (*timeType)(&t).Scan(value) }

func (t LocalTime) Time() time.Time              { return time.Time(t) }
func (t LocalTime) Value() (driver.Value, error) { return t.Time(), nil }
func (t LocalTime) Scan(value any) error         { return (*timeType)(&t).Scan(value) }

func (t DateTime) Time() time.Time              { return time.Time(t) }
func (t DateTime) Value() (driver.Value, error) { return t.Time(), nil }
func (t *DateTime) Scan(value any) error        { return (*timeType)(t).Scan(value) }

//
//	func (t *Text) Scan(value any) error {
//		switch v := value.(type) {
//		case string:
//			*t = Text(v)
//		case []byte:
//			*t = Text(v)
//		}
//		return query_errors.ErrTypeMismatch
//	}
//
//	func (s *String) Scan(value any) error {
//		switch v := value.(type) {
//		case string:
//			*s = String(v)
//		case []byte:
//			*s = String(v)
//		}
//		return query_errors.ErrTypeMismatch
//	}
//
//	func (i *Int) Scan(value any) error {
//		var rvSelf = reflect.ValueOf(i).Elem()
//		var rvValue = reflect.ValueOf(value)
//		if rvValue.Type() == rvSelf.Type() || rvValue.Type().ConvertibleTo(rvSelf.Type()) {
//			rvSelf.Set(rvValue.Convert(rvSelf.Type()))
//			return nil
//		}
//		return query_errors.ErrTypeMismatch
//	}
//
//	func (b *Bool) Scan(value any) error {
//		var rvSelf = reflect.ValueOf(b).Elem()
//		var rvValue = reflect.ValueOf(value)
//		switch rvValue.Kind() {
//		case reflect.Bool:
//			rvSelf.SetBool(rvValue.Bool())
//			return nil
//		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
//			if rvValue.Int() == 0 {
//				rvSelf.SetBool(false)
//			} else {
//				rvSelf.SetBool(true)
//			}
//			return nil
//		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
//			if rvValue.Uint() == 0 {
//				rvSelf.SetBool(false)
//			} else {
//				rvSelf.SetBool(true)
//			}
//			return nil
//		case reflect.String:
//			if rvValue.String() == "true" || rvValue.String() == "1" {
//				rvSelf.SetBool(true)
//			}
//			if rvValue.String() == "false" || rvValue.String() == "0" {
//				rvSelf.SetBool(false)
//			}
//			return nil
//		}
//		return query_errors.ErrTypeMismatch
//	}
//
//	func (b *Bytes) Scan(value any) error {
//		switch v := value.(type) {
//		case []byte:
//			*b = Bytes(v)
//		case string:
//			*b = Bytes(v)
//		default:
//			return query_errors.ErrTypeMismatch
//		}
//		return nil
//	}
//
//	func (f *Float) Scan(value any) error {
//		var rvSelf = reflect.ValueOf(f).Elem()
//		var rvValue = reflect.ValueOf(value)
//		if rvValue.Type() == rvSelf.Type() || rvValue.Type().ConvertibleTo(rvSelf.Type()) {
//			rvSelf.Set(rvValue.Convert(rvSelf.Type()))
//			return nil
//		}
//		return query_errors.ErrTypeMismatch
//	}
//
