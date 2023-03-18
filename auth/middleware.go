package auth

import (
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/request"
)

//	func IsAuthenticated(redirectURL string) func(next router.Handler) router.Handler {
//		return func(next router.Handler) router.Handler {
//			return router.HandleFuncWrapper{F: func(v router.Vars, w http.ResponseWriter, r *http.Request) {
//				rq := request.NewRequest(w, r, v)
//				var user = UserFromRequest(App, rq)
//				if !user.IsAuthenticated() {
//					rq.TemplateData.AddMessage("error", "You must be logged in to access that page")
//					rq.Redirect(redirectURL, 302)
//				} else {
//					next.ServeHTTP(v, w, r)
//				}
//			}}
//		}
//	}
//

// Get the user from a request.
func UserFromRequest(r *request.Request) *User {
	var userID = r.Session.Get(SESSION_COOKIE_NAME)
	if userID == nil {
		return UnAuthenticatedUser()
	}
	var user, err = GetUserByID(userID)
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

// Middleware to check if a user has a certain permission.
func PermsMiddleware(redirectURL string, permissions ...*Permission) router.Middleware {
	return func(h router.Handler) router.Handler {
		return router.HandleFunc(func(r *request.Request) {

			// Get the user from the request.
			var user = UserFromRequest(r)

			// Check if the user is authenticated.
			if !user.IsAuthenticated() {
				r.Data.AddMessage("error", "You must be logged in to access that page")
				r.Redirect(redirectURL, 302, r.Request.URL.Path)
				return
			}

			// Check if the user has the permission.
			if HasPerms(user, permissions...) {
				// Call the next handler.
				h.ServeHTTP(r)
				return
			}

			// If the user does not have the permission, redirect them.
			r.Data.AddMessage("error", "You do not have permission to access that page")
			r.Redirect("/", 302)
		})
	}
}

var LoginRequiredMiddleware = LoginRequiredURLMiddleware(LOGIN_URL)
var LogoutRequiredMiddleware = LogoutRequiredURLMiddleware(LOGIN_URL)

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
