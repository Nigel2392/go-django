package auth

import (
	"net/url"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/go-signals"
)

// import "github.com/Nigel2392/go-signals"
//
// /*
// Example usage:
//
//	auth.SIGNAL_BEFORE_USER_SAVE.Connect(signals.NewRecv(func(s signals.Signal, user ...any) error {
//		return nil
//	}))
//
// */
var (
	signal_pool      = signals.NewPool[url.Values]()
	user_signal_pool = signals.NewPool[*models.User]()
	id_signal_pool   = signals.NewPool[uint64]()

	SIGNAL_BEFORE_USER_CREATE = user_signal_pool.Get("user.before_create") // -> Send(auth.User) (Returned error unused!)
	SIGNAL_AFTER_USER_CREATE  = user_signal_pool.Get("user.after_create")  // -> Send(auth.User) (Returned error unused!)

	SIGNAL_BEFORE_USER_UPDATE = user_signal_pool.Get("user.before_update") // -> Send(auth.User) (Returned error unused!)
	SIGNAL_AFTER_USER_UPDATE  = user_signal_pool.Get("user.after_update")  // -> Send(auth.User) (Returned error unused!)

	SIGNAL_BEFORE_USER_DELETE = id_signal_pool.Get("user.before_delete") // -> Send(int64) (Returned error unused!)
	SIGNAL_AFTER_USER_DELETE  = id_signal_pool.Get("user.after_delete")  // -> Send(int64) (Returned error unused!)

	SIGNAL_USER_LOGGED_IN  = user_signal_pool.Get("auth.logged_in")  // -> Send(auth.User)		  (Returned error unused!)
	SIGNAL_USER_LOGGED_OUT = user_signal_pool.Get("auth.logged_out") // -> Send(auth.User(nil))		  (Returned error unused!)
	SIGNAL_LOGIN_FAILED    = signal_pool.Get("auth.login_failed")    // -> Send(auth.User, error) (Returned error unused!)
)
