package openauth2

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func UnAuthenticatedUser() *User {
	return &User{
		IsLoggedIn: false,
	}
}

// Get the user from a request.
func UserFromRequest(r *http.Request) *User {

	var session = sessions.Retrieve(r)
	except.Assert(
		session != nil,
		http.StatusInternalServerError,
		"Session must exist in the request",
	)

	var userID = session.Get(USER_ID_SESSION_KEY)
	if userID == nil {
		return UnAuthenticatedUser()
	}

	var userRow, err = queries.GetQuerySet(&User{}).
		Filter("ID", userID).
		Filter("IsActive", true).
		Get()
	if err != nil && errors.Is(err, errors.NoRows) {
		return UnAuthenticatedUser()
	} else if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to retrieve user from database",
		)
		return UnAuthenticatedUser()
	}

	userRow.Object.IsLoggedIn = true
	return userRow.Object
}

func UserFromRequestPure(r *http.Request) authentication.User {
	return UserFromRequest(r)
}

// Add a user to a request, if one exists in the session.
func AddUserMiddleware() mux.Middleware {
	return authentication.AddUserMiddleware(UserFromRequestPure)
}
