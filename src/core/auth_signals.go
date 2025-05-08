/*
Default signals for authentication events.

# These should be, and are available for custom authentication apps

It defines a generic interface for the user object, which is used in the signals.
You can use this interface to define your own user object, and use the signals to
handle events.
*/
package core

import (
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type User interface {
	attrs.Definer
	authentication.User
}

type UserWithRequest struct {
	User User
	Req  *http.Request
}

// import "github.com/Nigel2392/go-signals"
//
// /*
// Example usage:
//
//	django_signals.SIGNAL_BEFORE_USER_SAVE.Connect(signals.NewRecv(func(s signals.Signal, user ...any) error {
//		return nil
//	}))
//
// */
var (
	url_values_pool  = signals.NewPool[url.Values]()
	user_signal_pool = signals.NewPool[User]()
	user_req_pool    = signals.NewPool[UserWithRequest]()
	any_signals      = signals.NewPool[any]()

	SIGNAL_BEFORE_USER_CREATE = user_signal_pool.Get("user.before_create") // -> Send(auth.User) (Returned error unused!)
	SIGNAL_AFTER_USER_CREATE  = user_signal_pool.Get("user.after_create")  // -> Send(auth.User) (Returned error unused!)

	SIGNAL_BEFORE_USER_UPDATE = user_signal_pool.Get("user.before_update") // -> Send(auth.User) (Returned error unused!)
	SIGNAL_AFTER_USER_UPDATE  = user_signal_pool.Get("user.after_update")  // -> Send(auth.User) (Returned error unused!)

	SIGNAL_BEFORE_USER_DELETE = any_signals.Get("user.before_delete") // -> Send(int64) (Returned error unused!)
	SIGNAL_AFTER_USER_DELETE  = any_signals.Get("user.after_delete")  // -> Send(int64) (Returned error unused!)

	SIGNAL_USER_LOGGED_IN  = user_req_pool.Get("auth.logged_in")      // -> Send(auth.User)		  (Returned error unused!)
	SIGNAL_USER_LOGGED_OUT = user_req_pool.Get("auth.logged_out")     // -> Send(auth.User(nil))		  (Returned error unused!)
	SIGNAL_LOGIN_FAILED    = url_values_pool.Get("auth.login_failed") // -> Send(auth.User, error) (Returned error unused!)
)
