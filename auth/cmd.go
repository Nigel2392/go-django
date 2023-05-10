package auth

import (
	"context"
	"errors"

	"github.com/Nigel2392/forms"
	"github.com/Nigel2392/forms/validators"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/typeutils/terminal"
)

var CreateSuperUserCommand = &flag.Command{
	Name: "createsuperuser",
	Description: `Create a super user for the application.
This superuser can be used to log in to the admin panel.
A superuser can also create other users, and set their superuser status.`,
	Handler: createSuperUserFunc,
	Default: false,
}

// Shorthand for creating a super user
// Check if flag -createsuperuser is set
func createSuperUserFunc(v flag.Value) error {
	if v.IsZero() {
		return nil
	}
	// Ask for input
	CreateAdminFromTerminal()
	return nil
}

func CreateAdminFromTerminal() error {
	// Ask for input
	var email = terminal.Ask("Email: ", true)
	var username = terminal.Ask("Username: ", true)
	var first_name = terminal.Ask("First Name: ", true)
	var last_name = terminal.Ask("Last Name: ", true)
	var password = terminal.AskProtected("Password: ")
	var password2 = terminal.AskProtected("Password (confirm): ")

	// Check if the passwords match
	if password != password2 {
		//lint:ignore ST1005 I like capitalized error strings.
		return errors.New("Passwords do not match")
	}

	// Create the user
	var _, err = CreateAdminUser(
		email,
		username,
		first_name,
		last_name,
		password,
	)
	return err
}

// Create a super user
// This runs in a CLI to ask for input
// It will skip most of the validation.
func CreateAdminUser(email, username, first_name, last_name, password string) (*User, error) {
	var err error
	if err = validators.Length(3, 255)(forms.NewValue(email)); err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return nil, errors.New("Email must be between 3 and 255 characters")
	}
	if err = validators.Length(2, 75)(forms.NewValue(username)); err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return nil, errors.New("Username must be between 2 and 75 characters")
	}
	if err = validators.Length(2, 50)(forms.NewValue(first_name)); err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return nil, errors.New("First name must be between 2 and 50 characters")
	}
	if err = validators.Length(2, 50)(forms.NewValue(last_name)); err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return nil, errors.New("Last name must be between 2 and 50 characters")
	}
	if err = validators.Length(8, 255)(forms.NewValue(password)); err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return nil, errors.New("Password must be between 8 and 255 characters")
	}

	// Create the user
	var user = &User{
		Email:           EmailField(email),
		Username:        username,
		FirstName:       first_name,
		LastName:        last_name,
		IsAdministrator: true,
		IsActive:        true,
	}

	// Set the password
	err = SetPassword(user, password)
	if err != nil {
		return nil, err
	}

	// Update the user to be a super user
	user.IsAdministrator = true
	err = Auth.Queries.CreateUser(context.Background(), user)
	if err != nil {
		return nil, err
	}
	// Return nil
	return user, nil
}
