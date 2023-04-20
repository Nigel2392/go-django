package auth

import (
	"errors"

	"github.com/Nigel2392/go-django/forms"
	"github.com/Nigel2392/go-django/forms/validators"

	"golang.org/x/crypto/bcrypt"
)

// Validates email, username, first name, and last name.
func defaultValidation(u *User) error {
	if err := validators.Regex(validators.REGEX_EMAIL)(forms.NewValue(u.Email)); err != nil {
		//lint:ignore ST1005 Email is a field.
		return errors.New("Email is not valid")
	}
	if err := validators.Length(3, 255)(forms.NewValue(u.Email)); err != nil {
		//lint:ignore ST1005 Email is a field.
		return errors.New("Email must be between 3 and 255 characters")
	}
	if err := validators.Regex(validators.REGEX_ALPHANUMERIC)(forms.NewValue(u.Username)); err != nil {
		//lint:ignore ST1005 Username is a field.
		return errors.New("Username can only contain letters and numbers")
	}
	if err := validators.Length(3, 75)(forms.NewValue(u.Username)); err != nil {
		//lint:ignore ST1005 Username is a field.
		return errors.New("Username must be between 3 and 75 characters")
	}
	if err := validators.Length(0, 50)(forms.NewValue(u.FirstName)); err != nil {
		//lint:ignore ST1005 FirstName is a field.
		return errors.New("First name must be between 0 and 50 characters")
	}
	if err := validators.Length(0, 50)(forms.NewValue(u.LastName)); err != nil {
		//lint:ignore ST1005 LastName is a field.
		return errors.New("Last name must be between 0 and 50 characters")
	}
	return nil
}

func removeSliceItem[T any](slice []T, index int) []T {
	if len(slice) == 0 {
		return nil
	} else if len(slice) == 1 {
		return []T{}
	} else if index == 0 {
		return slice[1:]
	} else if index == len(slice)-1 {
		return slice[:len(slice)-1]
	} else {
		return append(slice[:index], slice[index+1:]...)
	}
}

// BcryptHash returns a bcrypt hash of the given string
func BcryptHash(s string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// BcryptCompare compares a bcrypt hash with a string
func BcryptCompare(hash, s string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(s))
}
