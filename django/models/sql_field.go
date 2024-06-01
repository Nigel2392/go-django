package models

import (
	"database/sql"
	"database/sql/driver"
	"net/mail"
	"reflect"
)

func isZero(rt reflect.Type, rv reflect.Value) bool {
	if !rv.IsValid() || (rt.Kind() == reflect.Ptr && rv.IsNil() || rt.Kind() != reflect.Ptr && rv.IsZero()) {
		return true
	}
	return false
}

func parseFn[T1 any, T2 any](parse func(T1) (T2, error)) func(T1) (T2, error) {
	return func(v T1) (T2, error) {
		var rTyp = reflect.TypeOf(v)
		var rVal = reflect.ValueOf(v)
		if isZero(rTyp, rVal) {
			return *new(T2), nil
		}
		if rTyp.Kind() == reflect.Ptr {
			return *new(T2), nil
		}
		return parse(v)
	}
}

func convertToGO[DBType any, GoType any](parse func(DBType) (GoType, error)) func(DBType) (GoType, error) {
	return parseFn(parse)
}

func convertToDB[GoType any, DBType any](parse func(*GoType) (DBType, error)) func(*GoType) (DBType, error) {
	return parseFn(parse)
}

var (
	_ (sql.Scanner)   = (*BaseSQLField[any, any])(nil)
	_ (driver.Valuer) = (*BaseSQLField[any, any])(nil)
)

type BaseSQLField[DBType, GoType any] struct {
	value       *GoType
	ConvertToDB func(*GoType) (DBType, error)
	ConvertToGO func(DBType) (*GoType, error)
}

func EmailField() *BaseSQLField[string, mail.Address] {
	return NewBaseSQLField(
		convertToDB(func(v *mail.Address) (string, error) {
			return v.String(), nil
		}),
		convertToGO(func(v string) (*mail.Address, error) {
			var addr, err = mail.ParseAddress(v)
			if err != nil {
				return &mail.Address{}, err
			}
			return addr, nil
		}),
	)
}

func NewBaseSQLField[DBType, GoType any](convertToDB func(*GoType) (DBType, error), convertToGO func(DBType) (*GoType, error)) *BaseSQLField[DBType, GoType] {
	return &BaseSQLField[DBType, GoType]{
		ConvertToDB: convertToDB,
		ConvertToGO: convertToGO,
	}
}

func (f *BaseSQLField[DBType, GoType]) Value() (driver.Value, error) {
	var (
		rTyp = reflect.TypeOf(f.value)
		rVal = reflect.ValueOf(f.value)
	)
	if isZero(rTyp, rVal) {
		return nil, nil
	}
	return f.ConvertToDB(f.value)
}

func (f *BaseSQLField[DBType, GoType]) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var v = value.(DBType)
	var parsed, err = f.ConvertToGO(v)
	if err != nil {
		return err
	}
	*f = BaseSQLField[DBType, GoType]{
		ConvertToDB: f.ConvertToDB,
		ConvertToGO: f.ConvertToGO,
		value:       parsed,
	}
	return nil
}
