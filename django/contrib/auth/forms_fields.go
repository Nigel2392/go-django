package auth

import (
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

var _ (fields.Field) = (*PasswordField)(nil)

type PasswordString string

type PasswordField struct {
	*fields.BaseField
	Validators []func(PasswordString) error
}

func PasswordValidators(fn ...func(PasswordString) error) func(*PasswordField) {
	return func(p *PasswordField) {
		if p.Validators == nil {
			p.Validators = make([]func(PasswordString) error, 0)
		}
		p.Validators = append(p.Validators, fn...)
	}
}

func NewPasswordField(flags PasswordCharacterFlag, isRegistering bool, opts ...func(fields.Field)) *PasswordField {
	var p = &PasswordField{
		BaseField: fields.NewField(
			fields.S("password"),
		),
	}

	opts = append(opts,
		fields.MinLength(8),
		fields.MaxLength(64),
		ValidateCharacters(
			isRegistering,
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
