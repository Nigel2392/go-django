package serializer

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-structs"
)

type field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

func getTypedJSON(s *structs.Struct) []field {
	var fields = make([]field, 0, s.NumField())
	for i := 0; i < s.NumField(); i++ {
		var f = s.Field(i)
		var name = jsonTag(f)
		var typ = getTyp(f.Type)
		var required = structs.IsRequired(f)
		fields = append(fields, field{
			Name:     name,
			Type:     typ,
			Required: required,
		})
	}
	return fields
}

func jsonTag(f reflect.StructField) string {
	var tag = f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	var parts = strings.Split(tag, ",")
	if len(parts) >= 1 {
		var part = parts[0]
		if part != "" {
			return part
		}
	}
	return f.Name
}

func getTyp(f reflect.Type) string {
	var typ strings.Builder
	switch f.Kind() {
	case reflect.String,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Bool:
		typ.WriteString(f.Kind().String())
	case reflect.Struct:
		typ.WriteString("struct")
		typ.WriteByte('[')
		typ.WriteString(f.Name())
		typ.WriteByte(']')
	case reflect.Ptr:
		typ.WriteString("ptr")
		typ.WriteByte('[')
		typ.WriteString(getTyp(f.Elem()))
		typ.WriteByte(']')
	case reflect.Slice:
		typ.WriteString("slice")
		typ.WriteByte('[')
		typ.WriteString(getTyp(f.Elem()))
		typ.WriteByte(']')
	case reflect.Array:
		typ.WriteString("array")
		typ.WriteByte('[')
		typ.WriteString(getTyp(f.Elem()))
		typ.WriteString("<len: ")
		typ.WriteString(strconv.Itoa(f.Len()))
		typ.WriteString(">")
		typ.WriteByte(']')
	case reflect.Map:
		typ.WriteString("map")
		typ.WriteByte('[')
		typ.WriteString(getTyp(f.Key()))
		typ.WriteByte(',')
		typ.WriteString(getTyp(f.Elem()))
		typ.WriteByte(']')
	default:
		typ.WriteString(f.Kind().String())
	}

	return typ.String()
}
