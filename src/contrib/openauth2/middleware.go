package openauth2

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func UnauthenticatedUser() *User {
	return &User{
		Base: users.Base{
			IsLoggedIn: false,
		},
	}
}

// Get the user from a request.
func UserFromRequest(r *http.Request) *User {
	var u = authentication.Retrieve(r)
	if u != nil {
		return u.(*User)
	}

	return loadUserFromRequest(r)
}

// Get the user from a request.
func loadUserFromRequest(r *http.Request) *User {
	var session = sessions.Retrieve(r)
	except.Assert(
		session != nil,
		http.StatusInternalServerError,
		"Session must exist in the request",
	)

	var userID = session.Get(USER_ID_SESSION_KEY)
	if userID == nil {
		return UnauthenticatedUser()
	}

	var userRow, err = queries.GetQuerySet(&User{}).
		Filter("ID", userID).
		Filter("IsActive", true).
		Get()
	if err != nil && errors.Is(err, errors.NoRows) {
		return UnauthenticatedUser()
	} else if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to retrieve user from database",
		)
		return UnauthenticatedUser()
	}

	userRow.Object.IsLoggedIn = true
	return userRow.Object
}

func userFromRequestPure(r *http.Request) authentication.User {
	return loadUserFromRequest(r)
}

// Add a user to a request, if one exists in the session.
func AddUserMiddleware() mux.Middleware {
	return authentication.AddUserMiddleware(userFromRequestPure)
}
