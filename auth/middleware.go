package auth

import (
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/request"
)

// Get the user from a request.
func UserFromRequest(r *request.Request) *User {
	if r.User != nil {
		return r.User.(*User)
	}
	var userID = r.Session.Get(SESSION_COOKIE_NAME)
	if userID == nil {
		return UnAuthenticatedUser()
	}
	var uidInt, ok = userID.(int64)
	if !ok {
		return UnAuthenticatedUser()
	}
	var user, err = Auth.Queries.GetUserByID(r.Request.Context(), uidInt)
	if err != nil || user == nil {
		return UnAuthenticatedUser()
	}
	user.IsLoggedIn = true
	return user
}

func UserFromRequestPure(r *request.Request) request.User {
	return UserFromRequest(r)
}

// Set the user inside of the request.
func UserToRequest(r *request.Request, user *User) {
	r.Session.Set(SESSION_COOKIE_NAME, user.ID)
}

// Add a user to a request, if one exists in the session.
func AddUserMiddleware() router.Middleware {
	return middleware.AddUserMiddleware(UserFromRequestPure)
}

// Middleware which checks if the user is authenticated.
func LoginRequiredURLMiddleware(redirectURL string) router.Middleware {
	return middleware.LoginRequiredMiddleware(func(r *request.Request) {
		r.Data.AddMessage("error", "You must be logged in to access that page")
		r.Redirect(redirectURL, 302, r.Request.URL.Path)
	})
}

// Middleware which checks if the user is not authenticated.
func LogoutRequiredURLMiddleware(redirectURL string) router.Middleware {
	return middleware.LogoutRequiredMiddleware(func(r *request.Request) {
		r.Data.AddMessage("error", "You must be logged out to access that page")
		r.Redirect(redirectURL, 302, r.Request.URL.Path)
	})
}
