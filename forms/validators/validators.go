package validators

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode"
)

type FormValue interface {
	IsFile() bool
	Value() string
	File() (string, io.ReadCloser)
}

type Validator func(FormValue) error

func New(validators ...Validator) []Validator {
	return validators
}

// MaxLength returns a validator that checks if the length of the string is at most max.
func MaxLength(max int) Validator {
	return func(value FormValue) error {
		if len(value.Value()) > max {
			return fmt.Errorf("value is too long")
		}
		return nil
	}
}

// MinLength returns a validator that checks if the length of the string is at least min.
func MinLength(min int) Validator {
	return func(value FormValue) error {
		if len(value.Value()) < min {
			return fmt.Errorf("value is too short")
		}
		return nil
	}
}

// Check if the string is at least min and at most max.
func Length(min, max int) Validator {
	return func(value FormValue) error {
		if len(value.Value()) < min {
			return fmt.Errorf("value is too short")
		}
		if len(value.Value()) > max {
			return fmt.Errorf("value is too long")
		}
		return nil
	}
}

// Check if the string

// Checks if:
// - password is at least 8 characters long
// - password is at most 64 characters long
// - password contains at least one uppercase letter
// - password contains at least one lowercase letter
// - password contains at least one digit
// - password contains at least one non-digit
// - password does not contain any whitespace
func PasswordStrength(password FormValue) error {
	if len(password.Value()) < 8 {
		return fmt.Errorf("password is too short")
	} else if len(password.Value()) > 32 {
		return fmt.Errorf("password is too long")
	}
	var upp_ct int = 0
	var low_ct int = 0
	var dig_ct int = 0
	var spa_ct int = 0
	for _, c := range password.Value() {
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

	if upp_ct == 0 || upp_ct == len(password.Value()) {
		return fmt.Errorf("password must contain at least one uppercase letter, and at least one lowercase letter")
	}
	if low_ct == 0 || low_ct == len(password.Value()) {
		return fmt.Errorf("password must contain at least one lowercase letter, and at least one uppercase letter")
	}
	if dig_ct == 0 || dig_ct == len(password.Value()) {
		return fmt.Errorf("password must contain at least one digit, and at least one non-digit")
	}
	if spa_ct > 0 {
		return fmt.Errorf("password must not contain spaces")
	}
	// Require at least one special character
	if len(password.Value()) == upp_ct+low_ct+dig_ct {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}

// Matches regex,
// Also matches custom strings,
// Example: Regex("<<email>>")("email") -> errors.New("not a match")
// Example: Regex("<<float>>")("0.01") -> nil
func Regex(regex string) func(value FormValue) error {
	return func(value FormValue) error {
		var reg = regexp.MustCompile(toRegex(regex))
		var match = reg.MatchString(value.Value())
		if !match {
			return errors.New("not a match")
		}
		return nil
	}
}
