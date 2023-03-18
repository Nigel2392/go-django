package forms

import (
	"fmt"
	"reflect"
	"strconv"
)

// Set a value to a struct field
// If the struct field is a pointer, but the value is not, then the value will be set to the pointer
func correctPtrSet(val1 reflect.Value, val2 any) {
	var v2 = reflect.ValueOf(val2)
	if !v2.IsValid() {
		return
	}
	var elem1 reflect.Value
	var elem2 reflect.Value
	if val1.Kind() == reflect.Ptr && v2.Kind() != reflect.Ptr {
		val1.Set(reflect.New(val1.Type().Elem()))
		elem1 = val1.Elem()
		elem2 = v2
	} else if val1.Kind() != reflect.Ptr && v2.Kind() == reflect.Ptr {
		elem1 = val1
		elem2 = v2.Elem()
	} else {
		elem1 = val1
		elem2 = v2
	}
	switch elem1.Kind() {
	case reflect.String:
		elem1.SetString(convertToString(elem2))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		elem1.SetInt(int64(convertToInt(elem2)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		elem1.SetUint(uint64(convertToUint(elem2)))
	case reflect.Float32, reflect.Float64:
		elem1.SetFloat(convertToFloat(elem2))
	case reflect.Bool:
		elem1.SetBool(convertToBool(elem2))
	default:
		elem1.Set(reflect.ValueOf(convertReflected(elem1, elem2)))
	}
}

func convertReflected(v, v2 reflect.Value) any {
	switch v.Kind() {
	case reflect.String:
		return convertToString(v2)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return convertToInt(v2)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return convertToUint(v2)
	case reflect.Float32, reflect.Float64:
		return convertToFloat(v2)
	case reflect.Bool:
		return convertToBool(v2)
	default:
		return v2.Interface()
	}
}

func convertToString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", v.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func convertToInt(v reflect.Value) int {
	switch v.Kind() {
	case reflect.String:
		var i, err = strconv.Atoi(v.String())
		if err != nil {
			return 0
		}
		return i
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int(v.Float())
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func convertToUint(v reflect.Value) uint {
	switch v.Kind() {
	case reflect.String:
		var i, err = strconv.Atoi(v.String())
		if err != nil {
			return 0
		}
		return uint(i)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(v.Uint())
	case reflect.Float32, reflect.Float64:
		return uint(v.Float())
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func convertToFloat(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.String:
		var i, err = strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return 0
		}
		return i
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func convertToBool(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		var i, err = strconv.ParseBool(v.String())
		if err != nil {
			return false
		}
		return i
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.Bool:
		return v.Bool()
	default:
		return false
	}
}
