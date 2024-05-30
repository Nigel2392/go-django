package fields

import "reflect"

func IsZero(value interface{}) bool {
	var rv = reflect.ValueOf(value)
	if !rv.IsValid() {
		return true
	}
	if rv.Kind() == reflect.Ptr {
		return rv.IsNil()
	}
	if rv.Kind() == reflect.String {
		return rv.String() == ""
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array || rv.Kind() == reflect.Map {
		return rv.Len() == 0
	}
	return false
}
