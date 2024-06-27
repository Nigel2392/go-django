//go:build testing_auth
// +build testing_auth

package auth

import (
	"net/http"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
)

func Login(r *http.Request, u *models.User) *models.User {
	u.IsLoggedIn = true
	return u
}

func Logout(r *http.Request) error {
	return nil
}
