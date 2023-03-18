package auth

import (
	"errors"
	"flag"

	"github.com/Nigel2392/typeutils/terminal"
)

// Shorthand for creating a super user
// Check if flag -createsuperuser is set
func CreateSuperUserFlag() error {
	if auth_db == nil {
		return errors.New("auth.Manager.auth_db is nil")
	}
	var f = flag.Bool("createsuperuser", false, "Create a super user")
	flag.Parse()
	if !*f {
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
