package drivers

import (
	"time"
)

type (
	Text   string
	String string
	Int    int64
	Bool   bool
	Bytes  []byte
	Float  float64
	Time   = time.Time
)

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
