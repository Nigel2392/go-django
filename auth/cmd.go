package auth

import (
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/typeutils/terminal"
)

var CreateSuperUserCommand = flag.Command{
	Name:        "createsuperuser",
	Description: "Create a super user",
	Handler:     createSuperUserFunc,
	Default:     false,
}

// Shorthand for creating a super user
// Check if flag -createsuperuser is set
func createSuperUserFunc(v flag.Value) error {
	if v.IsZero() {
		return nil
	}
	// Ask for input
	var email = terminal.Ask("Email: ", true)
	var username = terminal.Ask("Username: ", true)
	var first_name = terminal.Ask("First Name: ", true)
	var last_name = terminal.Ask("Last Name: ", true)
	var password = terminal.AskProtected("Password: ")

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
func CreateAdminUser(email, username, first_name, last_name, password string) (*User, error) {
	// Create the user
	var user, err = Register(email, username, first_name, last_name, password)
	if err != nil {
		return nil, err
	}

	// Update the user to be a super user
	user.IsAdministrator = true

	err = user.Update()
	if err != nil {
		return nil, err
	}
	// Return nil
	return user, nil
}
