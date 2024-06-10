package auth

import (
	"unicode"

	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
)

type PasswordCharacterFlag uint8

const (
	ChrFlagSpecial PasswordCharacterFlag = 1 << iota
	ChrFlagDigit
	ChrFlagLower
	ChrFlagUpper
)

type PasswordCharValidator struct {
	GenericError error
	Flags        PasswordCharacterFlag
}

func (p *PasswordCharValidator) Validate(password string) error {
	if len(password) == 0 {
		return errs.Error("password must not be empty")
	}

	var (
		upp_ct int = 0
		low_ct int = 0
		dig_ct int = 0
		spa_ct int = 0
	)

	for _, c := range password {
		if unicode.IsUpper(c) {
			upp_ct++
		}
		if unicode.IsLower(c) {
			low_ct++
		}
		if unicode.IsDigit(c) {
			dig_ct++
		}
		if unicode.IsSpace(c) {
			spa_ct++
		}
	}

	var err = errs.NewMultiError()

	if upp_ct == 0 || upp_ct == len(password) {
		if p.Flags&ChrFlagUpper != 0 {
			e := errs.Error("password must contain at least one uppercase letter, and at least one lowercase letter")
			err.Append(e)
		}
	}
	if low_ct == 0 || low_ct == len(password) {
		if p.Flags&ChrFlagLower != 0 {
			e := errs.Error("password must contain at least one lowercase letter, and at least one uppercase letter")
			err.Append(e)
		}
	}
	if dig_ct == 0 || dig_ct == len(password) {
		if p.Flags&ChrFlagDigit != 0 {
			e := errs.Error("password must contain at least one digit, and at least one non-digit")
			err.Append(e)
		}
	}
	if spa_ct > 0 {
		e := errs.Error("password must not contain spaces")
		err.Append(e)
	}
	if p.Flags&ChrFlagSpecial != 0 {
		// Require at least one special character
		if len(password) == upp_ct+low_ct+dig_ct {
			e := errs.Error("password must contain at least one special character")
			err.Append(e)
		}
	}

	if err.Len() > 0 {
		if p.GenericError != nil {
			return p.GenericError
		}
		return err
	}

	return nil
}

// Checks if:
// - password is at least minlen characters long
// - password is at most maxlen characters long
// - password contains at least one special character if specified
// - password contains at least one uppercase letter
// - password contains at least one lowercase letter
// - password contains at least one digit
// - password contains at least one non-digit
// - password does not contain any whitespace
func ValidateCharacters(isRegister bool, flags PasswordCharacterFlag) func(fields.Field) {
	var validator = &PasswordCharValidator{}

	if isRegister {
		validator.GenericError = errs.Error("Invalid password")
	}

	return func(fv fields.Field) {

		fv.SetValidators(func(i interface{}) error {
			if i == nil {
				return nil
			}
			if i == "" {
				return nil
			}
			var pw, ok = i.(string)
			if !ok {
				var password, ok = i.(PasswordString)
				if ok {
					pw = string(password)
					goto validate
				}
				return errs.ErrInvalidType
			}
		validate:
			return validator.Validate(pw)
		})
	}
}
