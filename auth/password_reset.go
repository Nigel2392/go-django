package auth

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/secret"
)

//lint:file-ignore ST1005 Errors can be returned to the user.

// Generate a password reset token for a user.
// This token can be used to get the user with VerifyPasswordResetToken.
// The token will expire after 24 hours.
// The token will also expire if the user's password is changed.
func GeneratePasswordResetToken(user *User) (string, error) {
	if !user.IsAuthenticated() {
		panic("Unauthenticated user supplied to GeneratePasswordResetToken.")
	}

	// Create a token string.
	var id = strconv.FormatUint(uint64(user.ID.UUID().ID()), 10)
	fmt.Println(id)
	var nowTime = strconv.FormatInt(time.Now().Unix(), 10)
	var password = secret.FnvHash(user.Password).String()
	var tokenString = id + "___" + password + "___" + nowTime

	// Encrypt the token.
	var token, err = secret.KEY.HTMLSafe().Encrypt(tokenString)
	if err != nil {
		return "", errors.New("Could not generate password reset token.")
	}

	// Create a signature for the token.
	var signature = secret.KEY.Sign(tokenString)

	return token + "." + signature, nil
}

// Beware! This returns the authenticated user.
// You can use this to reset the user's password.
//
//	token = ...
//	user, err := Manager.VerifyPasswordResetToken(token)
//	if err == nil {
//		user.ChangePassword("new password")
//	}
func VerifyPasswordResetToken(tokenString string) (*User, error) {
	// Split the token into the encrypted part and the signature.
	tParts := strings.Split(tokenString, ".")
	if len(tParts) != 2 {
		return nil, errors.New(MessageTokenInvalid)
	}
	tString := tParts[0]
	signature := tParts[1]

	// Decrypt the token.
	token, err := secret.KEY.HTMLSafe().Decrypt(tString)
	if err != nil {
		return nil, errors.New(MessageTokenInvalid)
	}

	// Verify the signature.
	if !secret.KEY.Verify(token, signature) {
		return nil, errors.New(MessageTokenInvalid)
	}

	// Split the token into its parts.
	tokenParts := strings.Split(token, "___")
	if len(tokenParts) != 3 {
		return nil, errors.New(MessageTokenInvalid)
	}

	// Parse the token parts.
	id, err := strconv.ParseUint(tokenParts[0], 10, 64)
	if err != nil {
		return nil, errors.New(MessageTokenInvalid)
	}
	var password = tokenParts[1]
	nowTime, err := strconv.ParseInt(tokenParts[2], 10, 64)
	if err != nil {
		return nil, errors.New(MessageTokenInvalid)
	}

	// Check if the token has expired.
	var expirationTime = time.Unix(nowTime, 0).Add(TokenExpiration)
	if time.Now().After(expirationTime) {
		return nil, errors.New(MessageTokenExpired)
	}

	// Get the user.
	user, err := GetUserByID(uint(id))
	if err != nil {
		return nil, errors.New(MessageTokenInvalid)
	}

	// If the user's password has changed, the token is deemed invalid.
	if secret.FnvHash(user.Password).String() != password {
		return nil, errors.New(MessageTokenInvalid)
	}

	user.IsLoggedIn = true
	return user, nil
}

// Very similar to VerifyPasswordResetToken, but also changes the user's password.
// Only updates the password column in the database.
// This is a convenience method.
func TokenResetPassword(tokenString, newPassword string) (*User, error) {
	// Verify the token.
	user, err := VerifyPasswordResetToken(tokenString)
	if err != nil {
		return nil, err
	}

	err = user.ChangePassword(newPassword)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ResetPassword resets a password for a given user, if the old password is correct.
// Only updates the password column in the database.
// This is a convenience method.
func ResetPassword(user *User, oldPassword, newPassword string) error {
	// Check the old password.
	if user.CheckPassword(oldPassword) != nil {
		return errors.New("Old password is incorrect.")
	}

	// Update the password.
	return user.ChangePassword(newPassword)
}
