package models

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/mail"
)

var _ (sql.Scanner) = (*Email)(nil)
var _ (driver.Valuer) = (*Email)(nil)

type Email mail.Address

func (e Email) String() string {
	return e.Address
}

func (e *Email) ScanAttribute(src interface{}) error {
	return e.Scan(src)
}

func (e *Email) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		a, err := mail.ParseAddress(v)
		if err != nil {
			return err
		}
		*e = Email(*a)
		return nil
	case []byte:
		a, err := mail.ParseAddress(string(v))
		if err != nil {
			return err
		}
		*e = Email(*a)
		return nil
	default:
		return errors.New("invalid email type")
	}
}

func (e Email) Value() (driver.Value, error) {
	var addr = e.Address
	return addr, nil
}

// // IsZero returns true if:
// // - the value is not valid
// // - the value is a pointer and is nil
// // - the value is not a pointer and is equal to the zero value of that type
// func isZero(rt reflect.Type, rv reflect.Value) bool {
// 	if !rv.IsValid() || (rt.Kind() == reflect.Ptr && rv.IsNil() || rt.Kind() != reflect.Ptr && rv.IsZero()) {
// 		return true
// 	}
// 	return false
// }

//
//func parseFn[T1 any, T2 any](parse func(T1) (T2, error)) func(T1) (T2, error) {
//	return func(v T1) (T2, error) {
//		var rTyp = reflect.TypeOf(v)
//		var rVal = reflect.ValueOf(v)
//		if isZero(rTyp, rVal) {
//			return *new(T2), nil
//		}
//		if rTyp.Kind() == reflect.Ptr {
//			return *new(T2), nil
//		}
//		return parse(v)
//	}
//}
//
//func convertToGO[DBType any, GoType any](parse func(DBType) (GoType, error)) func(DBType) (GoType, error) {
//	return parseFn(parse)
//}
//
//func convertToDB[GoType any, DBType any](parse func(*GoType) (DBType, error)) func(*GoType) (DBType, error) {
//	return parseFn(parse)
//}

// var (
// 	_ (sql.Scanner)   = (*BaseSQLField[any, any])(nil)
// 	_ (driver.Valuer) = (*BaseSQLField[any, any])(nil)
// )

// type BaseSQLField[DBType, GoType any] struct {
// 	structField *GoType
// 	ConvertToDB func(*GoType) (DBType, error)
// 	ConvertToGO func(DBType) (*GoType, error)
// }

// func EmailField() *BaseSQLField[string, mail.Address] {
// 	return NewBaseSQLField(
// 		//convertToDB(func(v *mail.Address) (string, error) {
// 		//	return v.String(), nil
// 		//}),
// 		//mail.ParseAddress,
// 		func(v *mail.Address) (string, error) {
// 			return v.String(), nil
// 		},
// 		mail.ParseAddress,
// 	)
// }

// func NewBaseSQLField[DBType, GoType any](convertToDB func(*GoType) (DBType, error), convertToGO func(DBType) (*GoType, error)) *BaseSQLField[DBType, GoType] {
// 	return &BaseSQLField[DBType, GoType]{
// 		ConvertToDB: convertToDB,
// 		ConvertToGO: convertToGO,
// 	}
// }

// func (f *BaseSQLField[DBType, GoType]) Set(v *GoType) {
// 	f.structField = v
// }

// func (f *BaseSQLField[DBType, GoType]) Get() *GoType {
// 	return f.structField
// }

// func (f *BaseSQLField[DBType, GoType]) Value() (driver.Value, error) {
// 	var (
// 		rTyp = reflect.TypeOf(f.structField)
// 		rVal = reflect.ValueOf(f.structField)
// 	)
// 	if isZero(rTyp, rVal) {
// 		return nil, nil
// 	}
// 	return f.ConvertToDB(f.structField)
// }

// func (f *BaseSQLField[DBType, GoType]) Scan(value interface{}) error {
// 	if value == nil {
// 		return nil
// 	}
// 	var v = value.(DBType)
// 	var parsed, err = f.ConvertToGO(v)
// 	if err != nil {
// 		return err
// 	}
// 	f.structField = parsed
// 	return nil
// }
