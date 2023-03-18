package auth

import "github.com/Nigel2392/go-signals"

/*
Example usage:

	auth.SIGNAL_BEFORE_USER_SAVE.Connect(signals.NewRecv(func(s signals.Signal, user ...any) error {
		return nil
	}))
*/

var (
	user_signal_pool = signals.NewPool[*User]()
	signal_pool      = signals.NewPool[any]()

	// Signals before anything happens to the user.
	SIGNAL_BEFORE_USER_SAVE   = user_signal_pool.Get("user.before_save")   // -> Send(auth.User) 	   (Returned error unused!)
	SIGNAL_BEFORE_USER_CREATE = user_signal_pool.Get("user.before_create") // -> Send(auth.User) 	   (Returned error unused!)
	SIGNAL_BEFORE_USER_UPDATE = user_signal_pool.Get("user.before_update") // -> Send(auth.User) 	   (Returned error unused!)
	SIGNAL_BEFORE_USER_DELETE = user_signal_pool.Get("user.before_delete") // -> Send(auth.User) 	   (Returned error unused!)

	// Signals after anything happens to the user.
	SIGNAL_AFTER_USER_SAVE   = user_signal_pool.Get("user.after_save")   // -> Send(auth.User) 	   (Returned error unused!)
	SIGNAL_AFTER_USER_CREATE = user_signal_pool.Get("user.after_create") // -> Send(auth.User) 	   (Returned error unused!)
	SIGNAL_AFTER_USER_UPDATE = user_signal_pool.Get("user.after_update") // -> Send(auth.User) 	   (Returned error unused!)
	SIGNAL_AFTER_USER_DELETE = user_signal_pool.Get("user.after_delete") // -> Send(auth.User) 	   (Returned error unused!)

	// Signals which notify of login/logout events.
	SIGNAL_USER_LOGGED_IN  = user_signal_pool.Get("auth.logged_in")  // -> Send(auth.User)		   (Returned error unused!)
	SIGNAL_USER_LOGGED_OUT = user_signal_pool.Get("auth.logged_out") // -> Send(auth.User)		   (Returned error unused!)
	SIGNAL_LOGIN_FAILED    = signal_pool.Get("auth.login_failed")    // -> Send(auth.User, error) (Returned error unused!)
)
