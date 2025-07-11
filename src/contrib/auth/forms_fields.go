package auth

import (
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var _ (fields.Field) = (*PasswordField)(nil)

type PasswordField struct {
	*fields.BaseField
	Validators []func(*Password) error
}

func PasswordValidators(fn ...func(*Password) error) func(*PasswordField) {
	return func(p *PasswordField) {
		if p.Validators == nil {
			p.Validators = make([]func(*Password) error, 0)
		}
		p.Validators = append(p.Validators, fn...)
	}
}

type PasswordFieldOptions struct {
	Flags             PasswordCharacterFlag
	IsRegistering     bool
	UseDefaultOptions bool
	Options           []func(fields.Field)
}

func NewPasswordField(config PasswordFieldOptions, opts ...func(fields.Field)) *PasswordField {
	var p = &PasswordField{
		BaseField: fields.NewField(),
	}

	var (
		flags         = config.Flags
		isRegistering = config.IsRegistering
	)

	if config.UseDefaultOptions {
		opts = append(opts,
			fields.MinLength(8),
			fields.MaxLength(64),
		)
	}

	if config.Flags != 0 {
		opts = append(opts, ValidateCharacters(
			isRegistering,
			flags,
		))
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.FormWidget == nil {
		p.FormWidget = widgets.NewPasswordInput(nil)
	}

	return p
}

func (p *PasswordField) ValueToForm(value interface{}) interface{} {
	var val, ok = value.(*Password)
	if !ok {
		return nil
	}

	if val.IsZero() {
		return ""
	}

	if val.Hash != "" {
		return val.Hash
	}

	return val.Raw
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

	var pw = NewPassword(val)

	for _, v := range p.Validators {
		if err := v(pw); err != nil {
			return nil, err
		}
	}

	return pw, nil
}
