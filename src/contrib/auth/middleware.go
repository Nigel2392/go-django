package auth

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

const SESSION_COOKIE_NAME = "user_authentication"

func UnAuthenticatedUser() *User {
	return &User{
		IsLoggedIn: false,
	}
}

// Get the user from a request.
func UserFromRequest(r *http.Request) *User {

	var u = authentication.Retrieve(r)
	if u != nil {
		return u.(*User)
	}

	var session = sessions.Retrieve(r)
	except.Assert(
		session != nil,
		http.StatusInternalServerError,
		"Session must exist in the request",
	)

	var userID = session.Get(SESSION_COOKIE_NAME)
	if userID == nil {
		return UnAuthenticatedUser()
	}

	var uidInt, ok = userID.(uint64)
	if !ok {
		return UnAuthenticatedUser()
	}
	var userRow, err = queries.GetQuerySet(&User{}).Filter("ID", uidInt).Get()
	if err != nil {
		return UnAuthenticatedUser()
	}

	user := userRow.Object
	user.IsLoggedIn = true
	return user
}

func UserFromRequestPure(r *http.Request) authentication.User {
	return UserFromRequest(r)
}

// Add a user to a request, if one exists in the session.
func AddUserMiddleware() mux.Middleware {
	return authentication.AddUserMiddleware(UserFromRequestPure)
}
