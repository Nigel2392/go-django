package django_reflect

import (
	"database/sql"
	"reflect"
	"strconv"
	"time"

	"github.com/Nigel2392/go-django/internal/bitch"
	"golang.org/x/exp/constraints"
)

type ScanFlag = bitch.Flag

const (
	SF_NONE        ScanFlag = 0
	SF_SQL_SCANNER ScanFlag = 1 << iota
	SF_STRCONV
	SF_REFLECTCONV

	SF_CONVS   = SF_STRCONV | SF_REFLECTCONV
	SF_DEFAULT = SF_SQL_SCANNER | SF_STRCONV | SF_REFLECTCONV
)

func setZero(dstPtr reflect.Value) {
	dstPtr.Elem().Set(reflect.Zero(dstPtr.Elem().Type()))
}

func genericStrconv[OUT, IN constraints.Integer](in string, fn func(string) (IN, error)) (result OUT, err error) {
	res, err := fn(in)
	return OUT(res), err
}

func parseUint[OUT constraints.Unsigned](in string) (OUT, error) {
	res, err := strconv.ParseUint(in, 10, 0)
	return OUT(res), err
}

func ScanTo[DST any](dstPtr *DST, src any, flags ScanFlag) (wasSet bool, err error) {

	var anyDest = any(dstPtr)
	if flags.Is(SF_SQL_SCANNER) {
		if scanner, ok := anyDest.(sql.Scanner); ok {
			err := scanner.Scan(ConvertToUniformType(src))
			return err == nil, err
		}
	}

	switch this := anyDest.(type) {
	case *int:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = int(val)
			wasSet = true
		case uint64:
			*this = int(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = genericStrconv[int](val, strconv.Atoi)
				wasSet = true
			}
		}

	case *int8:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = int8(val)
			wasSet = true
		case uint64:
			*this = int8(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = genericStrconv[int8](val, strconv.Atoi)
				wasSet = true
			}
		}

	case *int16:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = int16(val)
			wasSet = true
		case uint64:
			*this = int16(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = genericStrconv[int16](val, strconv.Atoi)
				wasSet = true
			}
		}

	case *int32:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = int32(val)
			wasSet = true
		case uint64:
			*this = int32(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = genericStrconv[int32](val, strconv.Atoi)
				wasSet = true
			}
		}

	case *int64:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = int64(val)
			wasSet = true
		case uint64:
			*this = int64(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = genericStrconv[int64](val, strconv.Atoi)
				wasSet = true
			}
		}

	case *uint:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = uint(val)
			wasSet = true
		case uint64:
			*this = uint(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = parseUint[uint](val)
				wasSet = true
			}
		}

	case *uint8:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = uint8(val)
			wasSet = true
		case uint64:
			*this = uint8(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = parseUint[uint8](val)
				wasSet = true
			}
		}

	case *uint16:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = uint16(val)
			wasSet = true
		case uint64:
			*this = uint16(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = parseUint[uint16](val)
				wasSet = true
			}
		}

	case *uint32:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = uint32(val)
			wasSet = true
		case uint64:
			*this = uint32(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = parseUint[uint32](val)
				wasSet = true
			}
		}

	case *uint64:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = uint64(val)
			wasSet = true
		case uint64:
			*this = uint64(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = parseUint[uint64](val)
				wasSet = true
			}
		}

	case *float32:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*this = float32(val)
			wasSet = true
		case string:
			var res float64
			res, err = strconv.ParseFloat(val, 64)
			if flags.Is(SF_STRCONV) {
				*this = float32(res)
				wasSet = true
			}
		}

	case *float64:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*this = val
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = strconv.ParseFloat(val, 64)
				wasSet = true
			}
		}

	case *[]byte:
		switch val := src.(type) {
		case []byte:
			*this = val
			wasSet = true
		case string:
			*this = []byte(val)
			wasSet = true
		}
		switch val := ConvertToUniformType(src).(type) {
		case []byte:
			*this = val
			wasSet = true
		case string:
			*this = []byte(val)
			wasSet = true
		}

	case *string:
		switch val := src.(type) {
		case []byte:
			*this = string(val)
			wasSet = true
		case string:
			*this = val
			wasSet = true
		}
		switch val := ConvertToUniformType(src).(type) {
		case []byte:
			*this = string(val)
			wasSet = true
		case string:
			*this = val
			wasSet = true
		}

	case *bool:
		switch val := src.(type) {
		case bool:
			*this = val
			wasSet = true
		case int64:
			*this = val != 0
			wasSet = true
		case uint64:
			*this = val != 0
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = strconv.ParseBool(val)
				wasSet = true
			}
		}

		switch val := ConvertToUniformType(src).(type) {
		case bool:
			*this = val
			wasSet = true
		case int64:
			*this = val != 0
			wasSet = true
		case uint64:
			*this = val != 0
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				*this, err = strconv.ParseBool(val)
				wasSet = true
			}
		}

	case *any:
		*this = src
	}

	if wasSet {
		return wasSet, err
	}

	return RScanTo(reflect.ValueOf(dstPtr), src, flags), err
}

func RScanTo(dstPtr reflect.Value, src any, flags ScanFlag) (wasSet bool) {

	if src == nil {
		setZero(dstPtr)
		return true
	}

	var (
		srcV       = reflect.ValueOf(src)
		srcTyp     = srcV.Type()
		dstElemVal = dstPtr.Elem()
		dstElemTyp = dstElemVal.Type()
	)

	if srcTyp == nil || srcV.Kind() == reflect.Invalid {
		setZero(dstPtr)
		return true
	}

	if srcTyp == dstElemTyp {
		dstElemVal.Set(srcV)
		return true
	}

	// Get the raw memory address of the destination
	ptr := dstPtr.UnsafePointer()

	switch dstElemVal.Kind() {
	case reflect.Int:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int)(ptr) = int(val)
			wasSet = true
		case uint64:
			*(*int)(ptr) = int(val)
			wasSet = true
		}
	case reflect.Int8:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int8)(ptr) = int8(val)
			wasSet = true
		case uint64:
			*(*int8)(ptr) = int8(val)
			wasSet = true
		}

	case reflect.Int16:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int16)(ptr) = int16(val)
			wasSet = true
		case uint64:
			*(*int16)(ptr) = int16(val)
			wasSet = true
		}

	case reflect.Int32:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int32)(ptr) = int32(val)
			wasSet = true
		case uint64:
			*(*int32)(ptr) = int32(val)
			wasSet = true
		}

	case reflect.Int64:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int64)(ptr) = int64(val)
			wasSet = true
		case uint64:
			*(*int64)(ptr) = int64(val)
			wasSet = true
		}
	case reflect.Uint:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint)(ptr) = uint(val)
			wasSet = true
		case uint64:
			*(*uint)(ptr) = uint(val)
			wasSet = true
		}
	case reflect.Uint8:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint8)(ptr) = uint8(val)
			wasSet = true
		case uint64:
			*(*uint8)(ptr) = uint8(val)
			wasSet = true
		}

	case reflect.Uint16:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint16)(ptr) = uint16(val)
			wasSet = true
		case uint64:
			*(*uint16)(ptr) = uint16(val)
			wasSet = true
		}

	case reflect.Uint32:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint32)(ptr) = uint32(val)
			wasSet = true
		case uint64:
			*(*uint32)(ptr) = uint32(val)
			wasSet = true
		}

	case reflect.Uint64:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint64)(ptr) = uint64(val)
			wasSet = true
		case uint64:
			*(*uint64)(ptr) = uint64(val)
			wasSet = true
		}

	case reflect.Float32:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*(*float32)(ptr) = float32(val)
			wasSet = true
		}

	case reflect.Float64:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*(*float64)(ptr) = val
			wasSet = true
		}

	case reflect.Slice:
		if dstElemTyp.Elem().Kind() == reflect.Uint8 {
			switch val := src.(type) {
			case []byte:
				// Avoid reflect.MakeSlice and reflect.Copy
				newBytes := make([]byte, len(val))
				copy(newBytes, val)
				*(*[]byte)(ptr) = newBytes
				wasSet = true

			case string:
				newBytes := make([]byte, len(val))
				copy(newBytes, val)
				*(*[]byte)(ptr) = newBytes
				wasSet = true
			}

			if !wasSet {
				switch srcTyp.Kind() {
				case reflect.Slice:
					if srcTyp.Elem().Kind() == reflect.Uint8 {
						srcBytes := srcV.Bytes()
						newBytes := make([]byte, len(srcBytes))
						copy(newBytes, srcBytes)
						*(*[]byte)(ptr) = newBytes
						wasSet = true
					}
				case reflect.String:
					srcStr := srcV.String()
					newBytes := make([]byte, len(srcStr))
					copy(newBytes, srcStr)
					*(*[]byte)(ptr) = newBytes
					wasSet = true
				}
			}
		}

	case reflect.String:
		switch val := ConvertToUniformType(src).(type) {
		case []byte:
			*(*string)(ptr) = string(val)
			wasSet = true

		case string:
			*(*string)(ptr) = val
			wasSet = true
		}

	case reflect.Bool:
		switch val := src.(type) {
		case bool:
			*(*bool)(ptr) = val
			wasSet = true
		case int64:
			*(*bool)(ptr) = val != 0
			wasSet = true
		case uint64:
			*(*bool)(ptr) = val != 0
			wasSet = true
		}

		switch val := ConvertToUniformType(src).(type) {
		case []byte:
			*(*string)(ptr) = string(val)
			wasSet = true

		case string:
			*(*string)(ptr) = val
			wasSet = true
		}
	}

	if wasSet {
		return true
	}

	if srcTyp.AssignableTo(dstElemTyp) {
		dstElemVal.Set(srcV)
		return true
	}

	if flags.Is(SF_REFLECTCONV) && srcTyp.ConvertibleTo(dstElemTyp) {
		srcV = srcV.Convert(dstElemTyp)
		dstElemVal.Set(srcV)
		return true
	}

	return false
}

func ConvertToUniformType(val any) any {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v

	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v

	case float32:
		return float64(v)
	case float64:
		return float64(v)

	case []byte:
		return []byte(v)

	case string:
		return string(v)

	case time.Time:
		return v

	case interface{ Time() time.Time }:
		// see queries/src/drivers/types.go time types
		return v.Time()
	}

	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return rv.Int()

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return rv.Uint()

	case reflect.Float32,
		reflect.Float64:
		return rv.Float()

	case reflect.String:
		return rv.String()

	case reflect.Slice:
		if rv.Elem().Kind() == reflect.Uint8 {
			return rv.Bytes()
		}
	}

	return val
}
