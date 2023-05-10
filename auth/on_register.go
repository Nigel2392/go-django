package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Nigel2392/forms"
	"github.com/Nigel2392/forms/validators"
)

func ValidEmail(u *User) error {
	return validators.Email(forms.NewValue(string(u.Email)))
}

func ValidUsername(u *User) error {
	var err error
	if err = validators.Length(3, 75)(forms.NewValue(u.Username)); err != nil {
		return errors.New("Username must be between 3 and 75 characters long")
	}
	if err = validators.Regex(validators.REGEX_ALPHANUMERIC, false)(forms.NewValue(u.Username)); err != nil {
		return errors.New("Username must be alphanumeric and can contain dashes and underscores")
	}
	return nil
}

func ValidFirstName(u *User) error {
	var err = validators.Length(1, 75)(forms.NewValue(u.FirstName))
	if err != nil {
		return errors.New("First name must be between 1 and 75 characters long")
	}
	return nil
}

func ValidLastName(u *User) error {
	var err = validators.Length(1, 75)(forms.NewValue(u.LastName))
	if err != nil {
		return errors.New("Last name must be between 1 and 75 characters long")
	}
	return nil
}

func ValidPassword(u *User) error {
	var err = validatePassword(string(u.Password))
	if err != nil {
		return errors.New("Error validating password, make sure it conforms to the password policy")
	}
	return nil
}

func UserDoesNotExist(u *User) error {
	var ctx = context.Background()
	var user *User
	var err error
	switch strings.ToLower(USER_MODEL_LOGIN_FIELD) {
	case "email":
		user, err = Auth.Queries.GetUserByEmail(ctx, string(u.Email))
	case "username":
		user, err = Auth.Queries.GetUserByUsername(ctx, u.Username)
	default:
		return errors.New("Could not validate login field: " + USER_MODEL_LOGIN_FIELD)
	}
	if err != nil {
		return nil
	}
	if user == nil {
		return nil
	}
	if user.ID == 0 {
		return nil
	}
	return errors.New("User already exists")
}

func SetUserActive(u *User) error {
	u.IsActive = true
	return nil
}
