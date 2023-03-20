package auth

import (
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/forms/validators"

	"github.com/Nigel2392/router/v3/request"
	"gorm.io/gorm"
)

// Register the User model with the gob package.
func init() {
	gob.Register(User{})
}

var auth_db *gorm.DB
var SESSION_COOKIE_NAME string = "session_id"
var DB_KEY db.DATABASE_KEY = "auth"

var USER_ABSOLUTE_URL_FUNC func(*User) string

var (
	USER_MODEL_LOGIN_FIELD string = "Username"
	USER_MODEL_NAME        string = "Users"
	GROUP_MODEL_NAME       string = "Groups"
	PERMISSION_MODEL_NAME  string = "Permissions"
	AUTH_APP_NAME          string = "Authentication"

	DEFAULT_USER_GROUP_NAMES = []string{
		"user",
	}
)

var (
	LOGIN_URL  string
	LOGOUT_URL string
)

var (
	TokenExpiration     = 24 * time.Hour
	MessageTokenInvalid = "Invalid password reset token."
	MessageTokenExpired = "Password reset token has expired."
)

// Initialize the auth package.
func Init(pool db.Pool[*gorm.DB], flags *flag.Flags) {
	var database = db.GetDefaultDatabase(DB_KEY, pool)
	auth_db = database.DB()
	database.Register(
		&User{},
		&Group{},
		&Permission{},
	)

	database.AutoMigrate()

	// Register the createsuperuser command.
	flags.RegisterCommand(CreateSuperUserCommand)
}

// Create a new unauthenticated used, with the login field set to the login parameter.
func NewUser(login string) *User {
	var u = &User{}
	u.SetLoginField(login)
	return u
}

// Return an unauthenticated user.
func UnAuthenticatedUser() *User {
	return &User{}
}

// GetUser returns the user from the database.
// If the user is not found, return an error.
func GetUser(query any, args ...interface{}) (*User, error) {
	var user User
	var err error
	switch query.(type) {
	case string:
		err = auth_db.Where(fmt.Sprintf("%s = ?", query), args...).First(&user).Error
	default:
		err = auth_db.Where(query, args...).First(&user).Error
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Register a group.
func RegisterGroups(groups ...*Group) error {
	var err error
	for _, group := range groups {
		err = auth_db.FirstOrCreate(group, "LOWER(name) = ?", strings.ToLower(group.Name)).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// Log the user in and set the user inside of the request.
//
// If the authentication was not successful, return the unauthenticated user, and an error.
//
// Authentication for the login_column_name is case insensitive!
func Login(r *request.Request, login, password string) (user *User, err error) {
	var u User

	// Do some quick validation before we try to hit the database.
	switch strings.ToLower(USER_MODEL_LOGIN_FIELD) {
	case "email":
		if err := validators.Regex(validators.REGEX_EMAIL)(login); err != nil {
			SIGNAL_LOGIN_FAILED.Send(&u, err)
			return UnAuthenticatedUser(), err
		}
	case "username":
		if err := validators.Length(3, 75)(login); err != nil {
			SIGNAL_LOGIN_FAILED.Send(&u, err)
			return UnAuthenticatedUser(), err
		}
	default:
		r.Logger.Warning("Could not validate login field: " + USER_MODEL_LOGIN_FIELD)
	}

	// Commented out due to the fact that createsuperuser does not perform this check.
	//	// Check the password strength.
	//	// If the password is not strong enough, return an error.
	//	if err := validators.PasswordStrength(password); err != nil {
	//		SIGNAL_LOGIN_FAILED.Send(&u, err)
	//		return UnAuthenticatedUser(), err
	//	}

	// Check the database for the user.
	err = auth_db.Where("LOWER("+USER_MODEL_LOGIN_FIELD+") = ?", strings.ToLower(login)).First(&u).Error
	if err != nil {
		SIGNAL_LOGIN_FAILED.Send(&u, err)
		return UnAuthenticatedUser(), err
	}

	// Validate the password.
	if err := u.CheckPassword(password); err != nil {
		//lint:ignore ST1005 potential error message to user.
		var err = errors.New("Invalid password")
		SIGNAL_LOGIN_FAILED.Send(&u, err)
		return UnAuthenticatedUser(), err
	}

	// Check if the user is active.
	if !u.IsActive {
		var err = errors.New("User is not active")
		SIGNAL_LOGIN_FAILED.Send(&u, err)
		return UnAuthenticatedUser(), err
	}

	// User is authenticated.
	LoginUnsafe(r, &u)
	return &u, nil
}

// Log the user in, without doing any of the validation.
//
// This is useful for when you have already done the validation (After registering for example).
func LoginUnsafe(r *request.Request, user *User) {
	r.Session.RenewToken()
	user.IsLoggedIn = true
	UserToRequest(r, user)
	SIGNAL_USER_LOGGED_IN.Send(user)
}

// Log the user out and set the user inside of the request to the unauthenticated user.
func Logout(r *request.Request) error {
	SIGNAL_USER_LOGGED_OUT.Send(r.User.(*User))
	UserToRequest(r, UnAuthenticatedUser())
	return r.Session.Destroy()
}

// Register a user.
func Register(email, username, first_name, last_name, password string) (*User, error) {
	var u = &User{
		Email:     email,
		Username:  username,
		FirstName: first_name,
		LastName:  last_name,
		Password:  password,
	}

	// Validate the fields
	// (Email, username, first_name, last_name, password)
	if err := u.validateFields(); err != nil {
		return nil, err
	}

	// Set the password
	err := u.SetPassword(password)
	if err != nil {
		return nil, err
	}

	// Validate the user
	if err := u.validate(); err != nil {
		return nil, err
	}

	// Create the user
	err = auth_db.Create(u).Error
	if err != nil {
		fmt.Println("Failed to create user: " + err.Error())
		//lint:ignore ST1005 potential error message to user.
		return nil, errors.New("Failed to create user")
	}
	return u, nil
}

// Verify that a user has the given permissions.
func HasPerms(user *User, permissions ...*Permission) bool {
	if user.IsAdministrator {
		return true
	}

	// If the user already has the groups loaded, continue...
	// Otherwise, load in the groups and permissions for the user
	// Permissions get loaded in on each request for the user.
	if len(user.Groups) == 0 {
		// Get all of the groups that the user is in
		// Load in the permissions for each group, into each group
		auth_db.Model(user).Association("Groups.Permissions").Find(&user.Groups)
	}

	// Return true if the user has all of the permissions
	return user.HasPerms(permissions...)
}

// Verify that a user has the given group
func HasGroup(user *User, groupName string) bool {
	return user.HasGroup(groupName)
}
