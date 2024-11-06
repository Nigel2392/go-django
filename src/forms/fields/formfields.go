package fields

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/mail"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func CharField(opts ...func(Field)) Field {
	return NewField(opts...)
}

type EmailFormField struct {
	*BaseField
}

func (e *EmailFormField) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return ""
	}
	switch val := value.(type) {
	case string:
		return val
	case *mail.Address:
		if val == nil {
			return ""
		}
		return val.Address
	case mail.Address:
		return val.Address
	default:
		return value
	}
}

func (e *EmailFormField) ValueToGo(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	var val, ok = value.(string)
	if !ok {
		return nil, errs.ErrInvalidType
	}
	if val == "" {
		return nil, nil
	}

	var addr, err = mail.ParseAddress(val)
	if err != nil {
		return nil, errors.Join(
			errs.ErrInvalidSyntax,
			err,
		)
	}
	return addr, nil
}

func EmailField(opts ...func(Field)) Field {
	var f = &EmailFormField{
		BaseField: NewField(opts...),
	}
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewEmailInput(nil)
	}
	return f
}

func BooleanField(opts ...func(Field)) Field {
	var f = NewField(opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewBooleanInput(nil)
	}
	return f
}

func DateField(typ widgets.DateWidgetType, opts ...func(Field)) Field {
	var f = NewField(opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewDateInput(nil, typ)
	}
	return f
}

func NumberField[T widgets.NumberType](opts ...func(Field)) Field {
	var f = NewField(opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewNumberInput[T](nil)
	}
	return f
}

func DecimalField(opts ...func(Field)) Field {
	var f = NewField(opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewDecimalInput(
			nil, 0,
		)
	}
	return f
}

type NullableSQLField[SQLType any] struct {
	*BaseField
}

func SQLNullField[SQLType any](opts ...func(Field)) *NullableSQLField[SQLType] {
	return &NullableSQLField[SQLType]{
		BaseField: NewField(opts...),
	}
}

func (n *NullableSQLField[SQLType]) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	if v, ok := value.(SQLType); ok {
		return v
	}

	switch any(*new(SQLType)).(type) {
	case bool:
		return value.(sql.NullBool).Bool
	case byte:
		return value.(sql.NullByte).Byte
	case int16:
		return value.(sql.NullInt16).Int16
	case int32:
		return value.(sql.NullInt32).Int32
	case int64:
		return value.(sql.NullInt64).Int64
	case float64:
		return value.(sql.NullFloat64).Float64
	case string:
		return value.(sql.NullString).String
	case time.Time:
		return value.(sql.NullTime).Time
	}

	return nil
}

func (n *NullableSQLField[SQLType]) ValueToGo(value interface{}) (interface{}, error) {
	switch value.(type) {
	case SQLType:
		switch any(*new(SQLType)).(type) {
		case sql.NullBool:
			return sql.NullBool{Valid: true, Bool: value.(bool)}, nil
		case sql.NullByte:
			return sql.NullByte{Valid: true, Byte: value.(byte)}, nil
		case sql.NullInt16:
			return sql.NullInt16{Valid: true, Int16: value.(int16)}, nil
		case sql.NullInt32:
			return sql.NullInt32{Valid: true, Int32: value.(int32)}, nil
		case sql.NullInt64:
			return sql.NullInt64{Valid: true, Int64: value.(int64)}, nil
		case sql.NullFloat64:
			return sql.NullFloat64{Valid: true, Float64: value.(float64)}, nil
		case sql.NullString:
			return sql.NullString{Valid: true, String: value.(string)}, nil
		case sql.NullTime:
			return sql.NullTime{Valid: true, Time: value.(time.Time)}, nil
		}
	case sql.NullBool, sql.NullByte, sql.NullInt16, sql.NullInt32, sql.NullInt64, sql.NullFloat64, sql.NullString, sql.NullTime:
		return value, nil
	case string:
		var val, err = n.Widget().ValueToGo(value)
		if err != nil {
			return nil, err
		}
		return n.ValueToGo(val)
	}

	if value != nil {
		return nil, errs.ErrInvalidType
	}

	switch any(*new(SQLType)).(type) {
	case bool:
		return sql.NullBool{}, nil
	case byte:
		return sql.NullByte{}, nil
	case int16:
		return sql.NullInt16{}, nil
	case int32:
		return sql.NullInt32{}, nil
	case int64:
		return sql.NullInt64{}, nil
	case float64:
		return sql.NullFloat64{}, nil
	case string:
		return sql.NullString{}, nil
	case time.Time:
		return sql.NullTime{}, nil
	}

	return nil, errs.ErrInvalidType
}

func (n *NullableSQLField[SQLType]) Widget() widgets.Widget {
	if n.FormWidget != nil {
		return n.FormWidget
	}
	switch any(*new(SQLType)).(type) {
	case int:
		return widgets.NewNumberInput[int](nil)
	case int8:
		return widgets.NewNumberInput[int8](nil)
	case int16:
		return widgets.NewNumberInput[int16](nil)
	case int32:
		return widgets.NewNumberInput[int32](nil)
	case int64:
		return widgets.NewNumberInput[int64](nil)
	case uint:
		return widgets.NewNumberInput[uint](nil)
	case uint8:
		return widgets.NewNumberInput[uint8](nil)
	case uint16:
		return widgets.NewNumberInput[uint16](nil)
	case uint32:
		return widgets.NewNumberInput[uint32](nil)
	case uint64:
		return widgets.NewNumberInput[uint64](nil)
	case float32:
		return widgets.NewNumberInput[float32](nil)
	case float64:
		return widgets.NewNumberInput[float64](nil)
	case string:
		return widgets.NewTextInput(nil)
	case bool:
		return widgets.NewBooleanInput(nil)
	case time.Time:
		return widgets.NewDateInput(nil, widgets.DateWidgetTypeDateTime)
	default:
		return widgets.NewTextInput(nil)
	}
}

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

type MarshallerFormField[T any] struct {
	*BaseField
	NewEncoder func(b io.Writer) Encoder
	NewDecoder func(b io.Reader) Decoder
}

func MarshallerField[T any](encoder func(io.Writer) Encoder, decoder func(io.Reader) Decoder, opts ...func(Field)) *MarshallerFormField[T] {
	assert.True(encoder != nil, "encoder is required")
	assert.True(decoder != nil, "decoder is required")

	return &MarshallerFormField[T]{
		BaseField:  NewField(opts...),
		NewEncoder: encoder,
		NewDecoder: decoder,
	}
}

func (m *MarshallerFormField[T]) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return ""
	}

	var b = new(bytes.Buffer)
	if err := m.NewEncoder(b).Encode(value); err != nil {
		return ""
	}

	return b.String()
}

func (m *MarshallerFormField[T]) ValueToGo(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	var (
		b = bytes.NewBufferString(value.(string))
		v T
	)

	if err := m.NewDecoder(b).Decode(&v); err != nil {
		return nil, err
	}

	return v, nil
}

// A wrapper around MarshallerFormField to better handle returned errors by ValueToGo
type JSONFormField[T any] struct {
	*MarshallerFormField[T]
}

func EncFunc[T Encoder](f func(io.Writer) T) func(io.Writer) Encoder {
	return func(b io.Writer) Encoder {
		return f(b)
	}
}

func DecFunc[T Decoder](f func(io.Reader) T) func(io.Reader) Decoder {
	return func(b io.Reader) Decoder {
		return f(b)
	}
}

func JSONField[T any](opts ...func(Field)) *JSONFormField[T] {
	var f = &JSONFormField[T]{
		MarshallerFormField: MarshallerField[T](
			EncFunc(json.NewEncoder),
			DecFunc(json.NewDecoder),
			opts...,
		),
	}

	if f.FormWidget == nil {
		f.FormWidget = widgets.NewTextarea(nil)
	}

	return f
}

func (j *JSONFormField[T]) ValueToGo(value interface{}) (interface{}, error) {
	if v, err := j.MarshallerFormField.ValueToGo(value); err != nil {
		switch err.(type) {
		case *json.SyntaxError:
			return nil, errs.Error("Invalid JSON syntax")
		case *json.UnmarshalTypeError:
			return nil, errs.Error("Invalid JSON type")
		case *json.InvalidUnmarshalError:
			return nil, errs.Error("Invalid JSON value")
		case *json.UnsupportedTypeError:
			return nil, errs.Error("Unsupported JSON type")
		case *json.UnsupportedValueError:
			return nil, errs.Error("Unsupported JSON value")
		default:
			return nil, errs.Error("Unexpected JSON error")
		}
	} else {
		return v, nil
	}
}
