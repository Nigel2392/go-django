package openauth2

import (
	"net/http"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func UnAuthenticatedUser() *openauth2models.User {
	return &openauth2models.User{
		IsLoggedIn: false,
	}
}

// Get the user from a request.
func UserFromRequest(r *http.Request) *openauth2models.User {

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

	var uidInt, ok = userID.(uint64)
	if !ok {
		return UnAuthenticatedUser()
	}
	var user, err = App.queryset.RetrieveUserByID(r.Context(), uidInt)
	if err != nil {
		return UnAuthenticatedUser()
	}

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
