package engine

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/core/serializer/marshallers"
	"github.com/Nigel2392/go-structs"
)

type Serializer interface {
	Serialize(interface{}) ([]byte, error)
	Deserialize([]byte, interface{}) error
}

type SerializerEngine struct {
	Struct     *structs.Struct
	Serializer Serializer
	Validators map[string][]func(interface{}) error
}

type Field struct {
	AbsName  string
	EncName  string
	Required bool
	Type     reflect.Type
}

func NewEngine(tag string, f ...Field) *SerializerEngine {
	var s = &SerializerEngine{
		Struct:     structs.New(tag),
		Serializer: &marshallers.JSON{},
	}
	for _, field := range f {
		s.Struct.AddField(field.AbsName, field.EncName, field.Type, field.Required)
	}
	return s
}

func (s *SerializerEngine) RegisterModel(model interface{}, fields ...string) {
	var typeOf, valueOf = reflect.TypeOf(model), reflect.ValueOf(model)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
		valueOf = valueOf.Elem()
	}
	for _, field := range fields {
		var fieldTyp, found = typeOf.FieldByName(field)
		if !found {
			panic(fmt.Sprintf("Field %s not found in %s", field, typeOf.Name()))
		}
		s.Struct.AddStructField(fieldTyp)
	}
}

func (s *SerializerEngine) StringField(absolute_name, name string, required bool) {
	s.Struct.StringField(absolute_name, name, required)
}

func (s *SerializerEngine) IntField(absolute_name, name string, required bool) {
	s.Struct.IntField(absolute_name, name, required)
}

func (s *SerializerEngine) FloatField(absolute_name, name string, required bool) {
	s.Struct.FloatField(absolute_name, name, required)
}

func (s *SerializerEngine) BoolField(absolute_name, name string, required bool) {
	s.Struct.BoolField(absolute_name, name, required)
}

func (s *SerializerEngine) SliceField(absolute_name, name string, required bool, typeOf reflect.Type) {
	s.Struct.SliceField(absolute_name, name, typeOf, required)
}

func (s *SerializerEngine) MapField(absolute_name, name string, required bool, typeOfKey, typeOfValue reflect.Type) {
	s.Struct.MapField(absolute_name, name, typeOfKey, typeOfValue, required)
}

func (s *SerializerEngine) StructField(absolute_name, name string, required bool, other *structs.Struct) {
	s.Struct.StructField(absolute_name, name, other, required)
}

func (s *SerializerEngine) AddField(absolute_name, name string, required bool, typ reflect.Type) {
	s.Struct.AddField(absolute_name, name, typ, required)
}

func (s *SerializerEngine) Serialize(v ...interface{}) ([]byte, error) {
	var value any
	if len(v) == 0 {
		panic("No value to serialize")
	}
	// If there is more than one value, we consider it as a slice
	// and we serialize it as a slice
	// Else we serialize it as a single value
	if len(v) > 1 {
		value = v
	} else {
		value = v[0]
	}

	s.Struct.Make()
	var typeOf = reflect.TypeOf(value)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	var err error
	switch typeOf.Kind() {
	case reflect.Struct:
		err = s.fillFieldsWithStruct(typeOf, value)
	case reflect.Slice:
		err = s.fillFieldsWithSlice(typeOf, value)
	case reflect.Map:
		err = s.fillFieldsWithMap(typeOf, value)
	default:
		if s.Struct.NumField() != 1 {
			return nil, fmt.Errorf("Cannot serialize non-struct type %s with more than one field", typeOf.Kind().String())
		}
		s.Struct.SetFieldByIndex(0, value)
	}
	if err != nil {
		return nil, err
	}

	data, err := s.Serializer.Serialize(s.Struct.Interface())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *SerializerEngine) fillFieldsWithStruct(typeOf reflect.Type, v interface{}) error {
	var structValue = reflect.ValueOf(v)
	if structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}
	for i := 0; i < typeOf.NumField(); i++ {
		var field = typeOf.Field(i)
		var structField = structValue.FieldByName(field.Name)
		if !structField.IsValid() {
			continue
		}
		if structField.Kind() == reflect.Ptr {
			structField = structField.Elem()
		}
		var myField = s.Struct.FieldByName(field.Name)
		if !myField.IsValid() {
			continue
		}
		if myField.Kind() == reflect.Ptr {
			myField = myField.Elem()
		}
		if structField.Kind() != myField.Kind() {
			return fmt.Errorf("Field %s has different types in struct and serializer", field.Name)
		}
		myField.Set(structField)
	}
	return nil
}

func (s *SerializerEngine) fillFieldsWithSlice(typeOf reflect.Type, v interface{}) error {
	if typeOf.Kind() != reflect.Slice {
		return fmt.Errorf("Cannot serialize non-slice type %s", typeOf.Kind().String())
	}
	var sliceValue = reflect.ValueOf(v)
	if sliceValue.Kind() == reflect.Ptr {
		sliceValue = sliceValue.Elem()
	}
	if sliceValue.Len() == 0 {
		return nil
	}
	if sliceValue.Len() != s.Struct.NumField() {
		return fmt.Errorf("Slice length is different from serializer length")
	}
	s.Struct.Make()
	for i := 0; i < s.Struct.NumField(); i++ {
		var sliceField = sliceValue.Index(i)
		if !sliceField.IsValid() {
			continue
		}

		sliceField = underLyingSliceValueOf(sliceField)

		if sliceField.Kind() != s.Struct.Field(i).Type.Kind() {
			return fmt.Errorf("Field %s has different types in slice and serializer: %s != %s", s.Struct.Field(i).Name, sliceField.Kind().String(), s.Struct.Field(i).Type.Kind().String())
		}
		s.Struct.SetFieldByIndex(i, sliceField)
	}
	return nil
}

func (s *SerializerEngine) fillFieldsWithMap(typeOf reflect.Type, v interface{}) error {
	if typeOf.Kind() != reflect.Map {
		return fmt.Errorf("Cannot serialize non-map type %s", typeOf.Kind().String())
	}
	var mapValue = reflect.ValueOf(v)
	if mapValue.Kind() == reflect.Ptr {
		mapValue = mapValue.Elem()
	}
	if mapValue.Len() == 0 {
		return nil
	}
	s.Struct.Make()
	for i := 0; i < s.Struct.NumField(); i++ {
		var mapKey = reflect.ValueOf(s.Struct.Field(i).Name)
		var mapField = mapValue.MapIndex(mapKey)
		if !mapField.IsValid() {
			continue
		}

		mapField = underLyingSliceValueOf(mapField)

		if mapField.Kind() != s.Struct.Field(i).Type.Kind() {
			return fmt.Errorf("Field %s has different types in map and serializer", s.Struct.Field(i).Name)
		}
		s.Struct.SetFieldByIndex(i, mapField)
	}
	return nil
}

func (s *SerializerEngine) Deserialize(data []byte, value ...interface{}) error {
	var v any
	if len(value) == 0 {
		panic("No value to serialize")
	}
	// If there is more than one value, we consider it as a slice
	// and we serialize it as a slice
	// Else we serialize it as a single value
	if len(value) > 1 {
		v = value
	} else {
		v = value[0]
	}

	s.Struct.Make()
	var err = s.Serializer.Deserialize(data, s.Struct.PtrTo())
	if err != nil {
		return err
	}
	var other = reflect.ValueOf(v)
	if other.Kind() == reflect.Ptr {
		other = other.Elem()
	}

	switch other.Kind() {
	case reflect.Struct:
		err = s.fillStructWithFields(other)
	case reflect.Slice:
		err = s.fillSliceWithFields(other)
	case reflect.Map:
		err = s.fillMapWithFields(other)
	default:
		if s.Struct.NumField() != 1 {
			return fmt.Errorf("Cannot deserialize non-struct type %s with more than one field", other.Kind().String())
		}
		var field = s.Struct.Field(0)
		var structField = s.Struct.FieldByName(field.Name)
		if !structField.IsValid() {
			return nil
		}
		if structField.Kind() == reflect.Ptr {
			structField = structField.Elem()
		}
		if other.Kind() != structField.Kind() {
			return fmt.Errorf("Field %s has different types in struct and serializer", field.Name)
		}

		if err = s.runValidators(field.Name, structField); err != nil {
			return err
		}

		other.Set(structField)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *SerializerEngine) fillStructWithFields(other reflect.Value) (err error) {
	for i := 0; i < s.Struct.NumField(); i++ {
		var field = s.Struct.Field(i)
		var structField = s.Struct.FieldByName(field.Name)
		if !structField.IsValid() {
			continue
		}
		if structField.Kind() == reflect.Ptr {
			structField = structField.Elem()
		}
		var otherField = other.FieldByName(field.Name)
		if !otherField.IsValid() {
			continue
		}
		if otherField.Kind() == reflect.Ptr {
			otherField = otherField.Elem()
		}
		if structField.Kind() != otherField.Kind() {
			return fmt.Errorf("Field %s has different types in struct and serializer", field.Name)
		}

		if structs.IsRequired(field) && !structField.IsValid() || structField.IsZero() {
			return fmt.Errorf("Field %s is required", field.Name)
		}

		err = s.runValidators(field.Name, structField)
		if err != nil {
			return err
		}

		otherField.Set(structField)
	}
	return nil
}

func (s *SerializerEngine) fillSliceWithFields(other reflect.Value) (err error) {
	if other.Len() != s.Struct.NumField() {
		return fmt.Errorf("Slice length is different from serializer length")
	}

	// If the slice is empty, we don't need to fill it
	for i := 0; i < s.Struct.NumField(); i++ {
		var field = s.Struct.Field(i)
		var structField = s.Struct.FieldByName(field.Name)
		if !structField.IsValid() {
			continue
		}
		if structField.Kind() == reflect.Ptr {
			structField = structField.Elem()
		}
		var otherField = other.Index(i)
		if !otherField.IsValid() {
			continue
		}

		otherField = underLyingSliceValueOf(otherField)

		if structField.Kind() != otherField.Kind() {
			return fmt.Errorf("Field %s has different types in struct and serializer", field.Name)
		}

		err = s.runValidators(field.Name, structField)
		if err != nil {
			return err
		}

		otherField.Set(structField)
	}
	return nil
}

func (s *SerializerEngine) fillMapWithFields(other reflect.Value) (err error) {
	for i := 0; i < s.Struct.NumField(); i++ {
		var field = s.Struct.Field(i)
		var structField = s.Struct.FieldByName(field.Name)
		if !structField.IsValid() {
			continue
		}
		if structField.Kind() == reflect.Ptr {
			structField = structField.Elem()
		}
		var otherField = other.MapIndex(reflect.ValueOf(field.Name))
		if !otherField.IsValid() {
			otherField = reflect.New(field.Type).Elem()
		}
		if otherField.Kind() == reflect.Ptr {
			otherField = otherField.Elem()
		}
		if structField.Kind() != otherField.Kind() {
			return fmt.Errorf("Field %s has different types in struct and serializer", field.Name)
		}

		if structs.IsRequired(field) && !structField.IsValid() || structField.IsZero() {
			return fmt.Errorf("Field %s is required", field.Name)
		}

		err = s.runValidators(field.Name, structField)
		if err != nil {
			return err
		}

		otherField.Set(structField)
		other.SetMapIndex(reflect.ValueOf(field.Name), otherField)
	}
	return nil
}

func underLyingSliceValueOf(v reflect.Value) reflect.Value {
	var kind = v.Kind()
	if kind == reflect.Ptr || kind == reflect.Interface || kind == reflect.Slice || kind == reflect.Array {
		v = v.Elem()
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func (s *SerializerEngine) runValidators(name string, structField reflect.Value) error {
	var validator, ok = s.Validators[name]
	if ok {
		var err error
		for _, v := range validator {
			err = v(structField)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
