package auth

import (
	"encoding/base64"
	"strconv"
	"strings"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"golang.org/x/crypto/bcrypt"
)

var HASHER = func(b string) (string, error) {
	var bytes, err = bcrypt.GenerateFromPassword([]byte(b), bcrypt.DefaultCost)
	return string(bytes), err
}

var CHECKER = func(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return ErrPwdHashMismatch
	}
	return nil
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

func SetPassword(u *models.User, password string) error {
	var pw, err = HASHER(password)
	if err != nil {
		return err
	}
	u.Password = models.Password(pw)
	return nil
}

func CheckPassword(u *models.User, password string) error {
	return CHECKER(string(u.Password), password)
}
