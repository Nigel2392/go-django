package django_reflect

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"strconv"
	"time"
	"unsafe"

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
func parseInt[OUT constraints.Integer](in string, bitSize int) (OUT, error) {
	res, err := strconv.ParseInt(in, 10, bitSize)
	return OUT(res), err
}
func parseUint[OUT constraints.Unsigned](in string, size int) (OUT, error) {
	res, err := strconv.ParseUint(in, 10, size)
	return OUT(res), err
}
func setPtr[OUT any](ptr *OUT, val OUT, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	*ptr = val
	return true, nil
}
func setUnsafePtr[OUT any](ptr unsafe.Pointer, val OUT, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	*(*OUT)(ptr) = val
	return true, nil
}

func ScanTo[DST any](dstPtr *DST, src any, flags ScanFlag) (wasSet bool, err error) {

	var anyDest = any(dstPtr)
	if flags.Is(SF_SQL_SCANNER) {
		if scanner, ok := anyDest.(sql.Scanner); ok {
			err := scanner.Scan(ConvertToUniformType(src))
			return err == nil, err
		}
	}

	if dv, ok := src.(driver.Valuer); ok {
		src, err = dv.Value()
		if err != nil {
			return false, err
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
				var v int
				v, err = parseInt[int](val, 0)
				wasSet, err = setPtr(this, v, err)
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
				var v int8
				v, err = parseInt[int8](val, 8)
				wasSet, err = setPtr(this, v, err)
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
				var v int16
				v, err = parseInt[int16](val, 16)
				wasSet, err = setPtr(this, v, err)
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
				var v int32
				v, err = parseInt[int32](val, 32)
				wasSet, err = setPtr(this, v, err)
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
				var v int64
				v, err = parseInt[int64](val, 64)
				wasSet, err = setPtr(this, v, err)
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
				var v uint
				v, err = parseUint[uint](val, 0)
				wasSet, err = setPtr(this, v, err)
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
				var v uint8
				v, err = parseUint[uint8](val, 8)
				wasSet, err = setPtr(this, v, err)
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
				var v uint16
				v, err = parseUint[uint16](val, 16)
				wasSet, err = setPtr(this, v, err)
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
				var v uint32
				v, err = parseUint[uint32](val, 32)
				wasSet, err = setPtr(this, v, err)
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
				var v uint64
				v, err = parseUint[uint64](val, 64)
				wasSet, err = setPtr(this, v, err)
			}
		}

	case *uintptr:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*this = uintptr(val)
			wasSet = true
		case uint64:
			*this = uintptr(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uintptr
				v, err = parseUint[uintptr](val, 0)
				wasSet, err = setPtr(this, v, err)
			}
		}

	case *float32:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*this = float32(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v float64
				v, err = strconv.ParseFloat(val, 32)
				wasSet, err = setPtr(this, float32(v), err)
			}
		}

	case *float64:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*this = val
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v float64
				v, err = strconv.ParseFloat(val, 64)
				wasSet, err = setPtr(this, v, err)
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
		case []rune:
			*this = []byte(string(val))
			wasSet = true
		}
		if !wasSet {
			switch val := ConvertToUniformType(src).(type) {
			case []byte:
				*this = val
				wasSet = true
			case string:
				*this = []byte(val)
				wasSet = true
			case []rune:
				*this = []byte(string(val))
				wasSet = true
			}
		}

	case *[]rune:
		switch val := src.(type) {
		case []byte:
			*this = []rune(string(val))
			wasSet = true
		case string:
			*this = []rune(val)
			wasSet = true
		case []rune:
			*this = val
			wasSet = true
		}
		if !wasSet {
			switch val := ConvertToUniformType(src).(type) {
			case []byte:
				*this = []rune(string(val))
				wasSet = true
			case string:
				*this = []rune(val)
				wasSet = true
			case []rune:
				*this = val
				wasSet = true
			}
		}

	case *string:
		switch val := src.(type) {
		case []byte:
			*this = string(val)
			wasSet = true
		case string:
			*this = val
			wasSet = true
		case []rune:
			*this = string(val)
			wasSet = true
		}
		if !wasSet {
			switch val := ConvertToUniformType(src).(type) {
			case []byte:
				*this = string(val)
				wasSet = true
			case string:
				*this = val
				wasSet = true
			case []rune:
				*this = string(val)
				wasSet = true
			}
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

		if !wasSet {
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
		}

	case *any:
		*this = src
		wasSet = true
	}

	if wasSet {
		return wasSet, err
	}

	if err != nil {
		return false, err
	}

	return RScanTo(reflect.ValueOf(dstPtr), src, flags)
}

func RScanTo(dstPtr reflect.Value, src any, flags ScanFlag) (wasSet bool, err error) {

	if src == nil {
		setZero(dstPtr)
		return true, nil
	}

	var (
		srcV       = reflect.ValueOf(src)
		srcTyp     = srcV.Type()
		dstElemVal = dstPtr.Elem()
		dstElemTyp = dstElemVal.Type()
	)

	if srcTyp == nil || srcV.Kind() == reflect.Invalid {
		setZero(dstPtr)
		return true, nil
	}

	if srcTyp == dstElemTyp {
		dstElemVal.Set(srcV)
		return true, nil
	}

	if srcTyp.AssignableTo(dstElemTyp) {
		dstElemVal.Set(srcV)
		return true, nil
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
		case string:
			if flags.Is(SF_STRCONV) {
				var v int
				v, err = parseInt[int](val, 0)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}
	case reflect.Int8:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int8)(ptr) = int8(val)
			wasSet = true
		case uint64:
			*(*int8)(ptr) = int8(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v int8
				v, err = parseInt[int8](val, 8)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Int16:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int16)(ptr) = int16(val)
			wasSet = true
		case uint64:
			*(*int16)(ptr) = int16(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v int16
				v, err = parseInt[int16](val, 16)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Int32:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int32)(ptr) = int32(val)
			wasSet = true
		case uint64:
			*(*int32)(ptr) = int32(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v int32
				v, err = parseInt[int32](val, 32)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Int64:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*int64)(ptr) = int64(val)
			wasSet = true
		case uint64:
			*(*int64)(ptr) = int64(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v int64
				v, err = parseInt[int64](val, 64)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Uint:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint)(ptr) = uint(val)
			wasSet = true
		case uint64:
			*(*uint)(ptr) = uint(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uint
				v, err = parseUint[uint](val, 0)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Uint8:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint8)(ptr) = uint8(val)
			wasSet = true
		case uint64:
			*(*uint8)(ptr) = uint8(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uint8
				v, err = parseUint[uint8](val, 8)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Uint16:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint16)(ptr) = uint16(val)
			wasSet = true
		case uint64:
			*(*uint16)(ptr) = uint16(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uint16
				v, err = parseUint[uint16](val, 16)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Uint32:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint32)(ptr) = uint32(val)
			wasSet = true
		case uint64:
			*(*uint32)(ptr) = uint32(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uint32
				v, err = parseUint[uint32](val, 32)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Uint64:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uint64)(ptr) = uint64(val)
			wasSet = true
		case uint64:
			*(*uint64)(ptr) = uint64(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uint64
				v, err = parseUint[uint64](val, 64)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Uintptr:
		switch val := ConvertToUniformType(src).(type) {
		case int64:
			*(*uintptr)(ptr) = uintptr(val)
			wasSet = true
		case uint64:
			*(*uintptr)(ptr) = uintptr(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v uintptr
				v, err = parseUint[uintptr](val, 0)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Float32:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*(*float32)(ptr) = float32(val)
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v float64
				v, err = strconv.ParseFloat(val, 32)
				wasSet, err = setUnsafePtr(ptr, float32(v), err)
			}
		}

	case reflect.Float64:
		switch val := ConvertToUniformType(src).(type) {
		case float64:
			*(*float64)(ptr) = val
			wasSet = true
		case string:
			if flags.Is(SF_STRCONV) {
				var v float64
				v, err = strconv.ParseFloat(val, 64)
				wasSet, err = setUnsafePtr(ptr, v, err)
			}
		}

	case reflect.Slice:
		if dstElemTyp.Elem().Kind() == reflect.Uint8 {
			switch val := src.(type) {
			case []byte:
				*(*[]byte)(ptr) = val
				wasSet = true
			case []rune:
				*(*[]byte)(ptr) = []byte(string(val))
				wasSet = true
			case string:
				*(*[]byte)(ptr) = []byte(val)
				wasSet = true
			}

			if !wasSet {
				switch val := ConvertToUniformType(src).(type) {
				case []byte:
					*(*[]byte)(ptr) = val
					wasSet = true
				case []rune:
					*(*[]byte)(ptr) = []byte(string(val))
					wasSet = true
				case string:
					*(*[]byte)(ptr) = []byte(val)
					wasSet = true
				}
			}
		}

		if dstElemTyp.Elem().Kind() == reflect.Int32 {
			switch val := src.(type) {
			case []byte:
				*(*[]rune)(ptr) = []rune(string(val))
				wasSet = true
			case []rune:
				*(*[]rune)(ptr) = val
				wasSet = true
			case string:
				*(*[]rune)(ptr) = []rune(val)
				wasSet = true
			}

			if !wasSet {
				switch val := ConvertToUniformType(src).(type) {
				case []byte:
					*(*[]rune)(ptr) = []rune(string(val))
					wasSet = true
				case []rune:
					*(*[]rune)(ptr) = val
					wasSet = true
				case string:
					*(*[]rune)(ptr) = []rune(val)
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
		switch val := ConvertToUniformType(src).(type) {
		case bool:
			*(*bool)(ptr) = val
			wasSet = true

		case int64:
			*(*bool)(ptr) = val != 0
			wasSet = true

		case uint64:
			*(*bool)(ptr) = val != 0
			wasSet = true

		case string:
			if flags.Is(SF_STRCONV) {
				var v bool
				v, err = strconv.ParseBool(val)
				wasSet, err = setUnsafePtr(ptr, v, err)

			}
		}
	}

	if err != nil {
		return false, err
	}

	if wasSet {
		return true, err
	}

	if srcTyp.AssignableTo(dstElemTyp) {
		dstElemVal.Set(srcV)
		return true, err
	}

	if flags.Is(SF_REFLECTCONV) && srcTyp.ConvertibleTo(dstElemTyp) {
		srcV = srcV.Convert(dstElemTyp)
		dstElemVal.Set(srcV)
		return true, err
	}

	return false, err
}

// Tries to convert val to an expected type.
//
// For example, all ints will be converted to int64
// The same logic goes for uint, float and complex respectively.
func ConvertToUniformType(val any) any {
	switch v := val.(type) {

	case int64,
		uint64,
		float64,
		complex128,
		[]byte,
		[]rune,
		string,
		bool,
		time.Time:
		return v

	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uintptr:
		return uint64(v)
	case float32:
		return float64(v)
	case complex64:
		return complex128(v)

	case interface{ Time() time.Time }:
		// see queries/src/drivers/types.go time types
		return v.Time()
	}

	rv := reflect.ValueOf(val)
	rt := rv.Type()
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
		reflect.Uint64,
		reflect.Uintptr:
		return rv.Uint()

	case reflect.Float32,
		reflect.Float64:
		return rv.Float()

	case reflect.String:
		return rv.String()

	case reflect.Bool:
		return rv.Bool()

	case reflect.Slice:
		elem := rt.Elem()
		if elem.Kind() == reflect.Uint8 {
			return rv.Bytes()
		}

		if elem.Kind() == reflect.Int32 {
			return fastRunes(rv)
		}
	}

	return val
}

func fastRunes(rv reflect.Value) []rune {
	return unsafe.Slice((*rune)(rv.UnsafePointer()), rv.Len())
}
