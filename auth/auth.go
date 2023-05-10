package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/Nigel2392/forms"
	"github.com/Nigel2392/forms/validators"
	"github.com/Nigel2392/netcache/src/client"
	"github.com/Nigel2392/router/v3/request"
	"golang.org/x/crypto/bcrypt"

	_ "embed"
)

var (
	AUTH_APP_NAME = "auth"
	LOGIN_URL     string
	LOGOUT_URL    string
)

//go:embed auth_schema.sql
var createTableQuery string

func UnAuthenticatedUser() *User {
	return &User{
		IsLoggedIn: false,
	}
}

// Return a new unauthenticated user with the login field set.
func NewUser(login string) *User {
	var u = UnAuthenticatedUser()
	u.SetLoginField(login)
	return u
}

type AuthApp struct {
	Queries AuthQuerier
	Cache   *client.Cache
	Logger  request.Logger
}

var (
	SESSION_COOKIE_NAME         = "session_id"
	DEFAULT_PASSWORD_VALIDATORS = validators.New(
		validators.PasswordStrength(8, 32, false),
	)
)

var Auth *AuthApp

func Initialize(db DBTX) *AuthApp {
	if Auth == nil {
		Auth = &AuthApp{}
	}
	Auth.Queries = NewQueries(db)
	var tableQueries = strings.Split(createTableQuery, ";")
	for _, query := range tableQueries {
		if query = strings.TrimSpace(query); query == "" {
			continue
		}
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			panic(err)
		}
	}

	return Auth
}

// Log the user in and set the user inside of the request.
//
// If the authentication was not successful, return the unauthenticated user, and an error.
//
// Authentication for the login_column_name is case insensitive!
func Login(r *request.Request, login, password string) (user *User, err error) {
	var u *User
	// Check the database for the user.
	switch strings.ToLower(USER_MODEL_LOGIN_FIELD) {
	case "email":
		if err := validators.Email(forms.NewValue(login)); err != nil {
			return UnAuthenticatedUser(), err
		}
		u, err = Auth.Queries.GetUserByEmail(r.Request.Context(), login)
	case "username":
		if err := validators.Length(3, 75)(forms.NewValue(login)); err != nil {
			return UnAuthenticatedUser(), err
		}
		u, err = Auth.Queries.GetUserByUsername(r.Request.Context(), login)
	default:
		return nil, errors.New("Could not validate login field, please contact a site administrator: " + USER_MODEL_LOGIN_FIELD)
	}
	if err != nil {
		return UnAuthenticatedUser(), errors.New("User does not exist: " + err.Error())
	}

	if err = validatePassword(password); err != nil {
		return UnAuthenticatedUser(), errors.New("Invalid password")
	}

	// Validate the password.
	if err = CheckPassword(u, password); err != nil {
		return UnAuthenticatedUser(), errors.New("Password is incorrect")
	}

	// Check if the user is active.
	if !u.IsActive {
		return UnAuthenticatedUser(), errors.New("Non-active user attempted to login")
	}

	// User is authenticated.
	LoginUnsafe(r, u)
	return u, nil
}

var onRegister = map[string]func(*User) error{
	"ValidEmail":       ValidEmail,
	"ValidUsername":    ValidUsername,
	"ValidFirstName":   ValidFirstName,
	"ValidLastName":    ValidLastName,
	"ValidPassword":    ValidPassword,
	"UserDoesNotExist": UserDoesNotExist,
	"SetUserActive":    SetUserActive,
}

// Register a function to run on register.
//
// Default functions which are registered:
//   - "ValidEmail"
//   - "ValidUsername"
//   - "ValidFirstName"
//   - "ValidLastName"
//   - "ValidPassword"
//   - "UserDoesNotExist" (Checks for duplicates in the database by username or email.)
//   - "SetUserActive" (Sets IsActive to be true)
func OnRegister(name string, fn func(*User) error) {
	onRegister[name] = fn
}

// Un-register a function to run on register.
func UnRegister(name string) {
	delete(onRegister, name)
}

// Get a username from an email address.
func UsernameFromEmail(email string) string {
	var parts = strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

// Register a user.
func Register(email, username, first_name, last_name, password string) (*User, error) {
	var u = &User{
		Email:     EmailField(email),
		Username:  username,
		FirstName: first_name,
		LastName:  last_name,
		Password:  PasswordField(password),
	}

	// Run the on register functions.
	var err error
	for _, fn := range onRegister {
		if err = fn(u); err != nil {
			return nil, errors.New("Could not register: " + err.Error())
		}
	}

	// Set the password
	err = SetPassword(u, password)
	if err != nil {
		return nil, errors.New("Could not set password")
	}

	var ctx = context.Background()
	err = Auth.Queries.CreateUser(ctx, u)
	return u, errors.New("Could not create user: " + err.Error())
}

// Log the user in, without doing any of the validation.
//
// This is useful for when you have already done the validation (After registering for example).
func LoginUnsafe(r *request.Request, user *User) {
	r.Session.RenewToken()
	user.IsLoggedIn = true
	UserToRequest(r, user)
	// SIGNAL_USER_LOGGED_IN.Send(user)
}

// Log the user out and set the user inside of the request to the unauthenticated user.
func Logout(r *request.Request) error {
	// SIGNAL_USER_LOGGED_OUT.Send(r.User.(*User))
	UserToRequest(r, UnAuthenticatedUser())
	return r.Session.Destroy()
}

var HASHER = func(b string) (string, error) {
	var bytes, err = bcrypt.GenerateFromPassword([]byte(b), bcrypt.DefaultCost)
	return string(bytes), err
}

var CHECKER = func(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// This function likely cannot 100% guarantee that the password is hashed.
// It might be susceptible to false positives or meticulously crafted user input.
var IS_HASHED = func(hashedPassword string) bool {
	return isBcryptHash(string(hashedPassword))
}

var (
	bcryptEncodingStr = "./ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	bcryptEncoding    = base64.NewEncoding(bcryptEncodingStr).WithPadding(base64.NoPadding)
)

// Checks if the input is a hashed password.
func isBcryptHash(s string) bool {
	var parts = strings.Split(s, "$")
	if len(parts) != 4 {
		return false
	}
	if parts[0] != "" {
		return false
	}
	if parts[1] != "2b" &&
		parts[1] != "2y" &&
		parts[1] != "2a" &&
		parts[1] != "2x" {
		return false
	}
	// check that the cost is a valid number
	if _, err := strconv.Atoi(parts[2]); err != nil {
		return false
	}

	if len(parts[3]) != 53 {
		return false
	}

	var _, err = bcrypt.Cost([]byte(s))
	if err != nil {
		return false
	}

	var salt = parts[3][0:22]
	_, err = bcryptEncoding.DecodeString(salt)
	if err != nil {
		return false
	}

	var hash = parts[3][22:]
	_, err = bcryptEncoding.DecodeString(hash)
	return err == nil
}

func SetPassword(u *User, password string) error {
	var pw, err = HASHER(password)
	if err != nil {
		return err
	}
	u.Password = PasswordField(pw)
	return nil
}

func CheckPassword(u *User, password string) error {
	return CHECKER(string(u.Password), password)
}

func validatePassword(password string) error {
	if DEFAULT_PASSWORD_VALIDATORS != nil {
		for _, v := range DEFAULT_PASSWORD_VALIDATORS {
			if err := v(forms.NewValue(password)); err != nil {
				return err
			}
		}
	} else {
		panic("DEFAULT_PASSWORD_VALIDATORS is nil")
	}
	return nil
}
