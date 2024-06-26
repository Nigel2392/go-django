package auth

import (
	"reflect"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/goldcrest"
)

var _ (fields.Field) = (*PasswordField)(nil)

type PasswordString string

type PasswordField struct {
	*fields.BaseField
	Validators []func(PasswordString) error
}

func init() {
	var passwordType = reflect.TypeOf(PasswordString(""))

	goldcrest.Register(
		attrs.HookFormFieldForType, 0,
		attrs.FormFieldGetter(func(f attrs.Field, t reflect.Type, v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
			if t == passwordType {
				return NewPasswordField(ChrFlagDigit|ChrFlagLower|ChrFlagUpper|ChrFlagSpecial, opts...), true
			}
			return nil, false
		}),
	)
}

func PasswordValidators(fn ...func(PasswordString) error) func(*PasswordField) {
	return func(p *PasswordField) {
		if p.Validators == nil {
			p.Validators = make([]func(PasswordString) error, 0)
		}
		p.Validators = append(p.Validators, fn...)
	}
}

func NewPasswordField(flags PasswordCharacterFlag, opts ...func(fields.Field)) *PasswordField {
	var p = &PasswordField{
		BaseField: fields.NewField(
			fields.S("password"),
		),
	}

	opts = append(opts,
		fields.MinLength(8),
		fields.MaxLength(64),
		ValidateCharacters(
			false,
			flags,
		),
	)

	for _, opt := range opts {
		opt(p)
	}

	if p.FormWidget == nil {
		p.FormWidget = widgets.NewPasswordInput(nil)
	}

	return p
}

func (p *PasswordField) Clean(value interface{}) (interface{}, error) {
	var val, ok = value.(string)
	if !ok {
		return nil, errs.ErrInvalidType
	}

	if val == "" && p.Required() {
		return nil, errs.ErrFieldRequired
	} else if val == "" {
		return nil, nil
	}

	var pw = PasswordString(val)

	for _, v := range p.Validators {
		if err := v(pw); err != nil {
			return nil, err
		}
	}

	return pw, nil
}
