package auth

import (
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
)

var _ (fields.Field) = (*PasswordField)(nil)

type PasswordField struct {
	*fields.BaseField
	Validators []func(string) error
}

func PasswordValidators(fn ...func(string) error) func(*PasswordField) {
	return func(p *PasswordField) {
		if p.Validators == nil {
			p.Validators = make([]func(string) error, 0)
		}
		p.Validators = append(p.Validators, fn...)
	}
}

func NewPasswordField(opts ...func(*PasswordField)) *PasswordField {
	var p = &PasswordField{
		BaseField: fields.NewField(fields.S("password")),
	}

	for _, opt := range opts {
		opt(p)
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

	for _, v := range p.Validators {
		if err := v(val); err != nil {
			return nil, err
		}
	}

	return val, nil
}
