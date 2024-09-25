package fields

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/mail"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func CharField(opts ...func(Field)) Field {
	return NewField(S("text"), opts...)
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
		BaseField: NewField(S("email"), opts...),
	}
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewEmailInput(nil)
	}
	return f
}

func BooleanField(opts ...func(Field)) Field {
	var f = NewField(S("boolean"), opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewBooleanInput(nil)
	}
	return f
}

func DateField(typ widgets.DateWidgetType, opts ...func(Field)) Field {
	var f = NewField(S("date"), opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewDateInput(nil, typ)
	}
	return f
}

func NumberField[T widgets.NumberType](opts ...func(Field)) Field {
	var f = NewField(S("number"), opts...)
	if f.FormWidget == nil {
		f.FormWidget = widgets.NewNumberInput[T](nil)
	}
	return f
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
		BaseField:  NewField(S("text"), opts...),
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
