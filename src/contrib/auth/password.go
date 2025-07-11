package auth

import (
	"unicode"

	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/pkg/errors"
)

type PasswordCharacterFlag uint8

const (
	ChrFlagSpecial PasswordCharacterFlag = 1 << iota
	ChrFlagDigit
	ChrFlagLower
	ChrFlagUpper
	ChrFlagAll = ChrFlagSpecial | ChrFlagDigit | ChrFlagLower | ChrFlagUpper
)

var ChrFlagDEFAULT = ChrFlagAll

type PasswordCharValidator struct {
	GenericError error
	Flags        PasswordCharacterFlag
}

func (p *PasswordCharValidator) Validate(password string) error {
	if len(password) == 0 {
		// return errs.Error("password must not be empty")
		return errors.Wrap(
			errs.ErrFieldRequired,
			"password must not be empty",
		)
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
			err.Append(autherrors.ErrPwdCasingUpper)
		}
	}
	if low_ct == 0 || low_ct == len(password) {
		if p.Flags&ChrFlagLower != 0 {
			err.Append(autherrors.ErrPwdCasingLower)
		}
	}
	if dig_ct == 0 || dig_ct == len(password) {
		if p.Flags&ChrFlagDigit != 0 {
			err.Append(autherrors.ErrPwdDigits)
		}
	}

	if spa_ct > 0 {
		err.Append(autherrors.ErrPwdSpaces)
	}

	if p.Flags&ChrFlagSpecial != 0 {
		// Require at least one special character
		if len(password) == upp_ct+low_ct+dig_ct+spa_ct {
			err.Append(autherrors.ErrPwdSpecial)
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
	var validator = &PasswordCharValidator{
		Flags: flags,
	}

	if !isRegister {
		validator.GenericError = autherrors.ErrGenericAuthFail
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
				var password, ok = i.(*Password)
				if ok {
					pw = string(password.Raw)
					goto validate
				}
				return errs.ErrInvalidType
			}
		validate:
			return validator.Validate(pw)
		})
	}
}
