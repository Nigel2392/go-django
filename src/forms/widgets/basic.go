package widgets

import (
	"errors"
	"net/mail"

	"github.com/Nigel2392/go-django/src/core/errs"
)

func NewTextInput(attrs map[string]string) Widget {
	return NewBaseWidget("text", "forms/widgets/text.html", attrs)
}

func NewTextarea(attrs map[string]string) Widget {
	return NewBaseWidget("textarea", "forms/widgets/textarea.html", attrs)
}

func NewEmailInput(attrs map[string]string) Widget {
	return &EmailInput{
		BaseWidget: NewBaseWidget("email", "forms/widgets/email.html", attrs),
	}
}

func NewPasswordInput(attrs map[string]string) Widget {
	return NewBaseWidget("password", "forms/widgets/password.html", attrs)
}

func NewHiddenInput(attrs map[string]string) Widget {
	return NewBaseWidget("hidden", "forms/widgets/hidden.html", attrs)
}

type EmailInput struct {
	*BaseWidget
}

func (e *EmailInput) ValueToForm(value interface{}) interface{} {
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

func (e *EmailInput) ValueToGo(value interface{}) (interface{}, error) {
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
