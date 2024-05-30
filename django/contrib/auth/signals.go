package auth

//
//import "github.com/Nigel2392/go-signals"
//
///*
//Example usage:
//
//	auth.SIGNAL_BEFORE_USER_SAVE.Connect(signals.NewRecv(func(s signals.Signal, user ...any) error {
//		return nil
//	}))
//*/
//
//var (
//	user_signal_pool        = signals.NewPool[*User]()
//	group_signal_pool       = signals.NewPool[*Group]()
//	permissions_signal_pool = signals.NewPool[*Permission]()
//	id_signal_pool          = signals.NewPool[int64]()
//
//	// Signals for users.
//	SIGNAL_BEFORE_USER_CREATE = user_signal_pool.Get("user.before_create") // -> Send(auth.User) (Returned error unused!)
//	SIGNAL_BEFORE_USER_UPDATE = user_signal_pool.Get("user.before_update") // -> Send(auth.User) (Returned error unused!)
//	SIGNAL_AFTER_USER_CREATE  = user_signal_pool.Get("user.after_create")  // -> Send(auth.User) (Returned error unused!)
//	SIGNAL_AFTER_USER_UPDATE  = user_signal_pool.Get("user.after_update")  // -> Send(auth.User) (Returned error unused!)
//
//	// Signals for groups.
//	SIGNAL_BEFORE_GROUP_CREATE = group_signal_pool.Get("group.before_create") // -> Send(auth.Group) (Returned error unused!)
//	SIGNAL_BEFORE_GROUP_UPDATE = group_signal_pool.Get("group.before_update") // -> Send(auth.Group) (Returned error unused!)
//	SIGNAL_AFTER_GROUP_CREATE  = group_signal_pool.Get("group.after_create")  // -> Send(auth.Group) (Returned error unused!)
//	SIGNAL_AFTER_GROUP_UPDATE  = group_signal_pool.Get("group.after_update")  // -> Send(auth.Group) (Returned error unused!)
//
//	// Signals for permissions.
//	SIGNAL_BEFORE_PERMISSION_CREATE = permissions_signal_pool.Get("permission.before_create") // -> Send(auth.Permission) (Returned error unused!)
//	SIGNAL_BEFORE_PERMISSION_UPDATE = permissions_signal_pool.Get("permission.before_update") // -> Send(auth.Permission) (Returned error unused!)
//	SIGNAL_AFTER_PERMISSION_CREATE  = permissions_signal_pool.Get("permission.after_create")  // -> Send(auth.Permission) (Returned error unused!)
//	SIGNAL_AFTER_PERMISSION_UPDATE  = permissions_signal_pool.Get("permission.after_update")  // -> Send(auth.Permission) (Returned error unused!)
//
//	// Deletions only require the ID of the object.
//	SIGNAL_BEFORE_USER_DELETE       = id_signal_pool.Get("user.before_delete")       // -> Send(int64) (Returned error unused!)
//	SIGNAL_BEFORE_GROUP_DELETE      = id_signal_pool.Get("group.before_delete")      // -> Send(int64) (Returned error unused!)
//	SIGNAL_BEFORE_PERMISSION_DELETE = id_signal_pool.Get("permission.before_delete") // -> Send(int64) (Returned error unused!)
//	SIGNAL_AFTER_USER_DELETE        = id_signal_pool.Get("user.after_delete")        // -> Send(int64) (Returned error unused!)
//	SIGNAL_AFTER_GROUP_DELETE       = id_signal_pool.Get("group.after_delete")       // -> Send(int64) (Returned error unused!)
//	SIGNAL_AFTER_PERMISSION_DELETE  = id_signal_pool.Get("permission.after_delete")  // -> Send(int64) (Returned error unused!)
//
//	//	// Signals which notify of login/logout events.
//	//	SIGNAL_USER_LOGGED_IN  = user_signal_pool.Get("auth.logged_in")  // -> Send(auth.User)		  (Returned error unused!)
//	//	SIGNAL_USER_LOGGED_OUT = user_signal_pool.Get("auth.logged_out") // -> Send(auth.User)		  (Returned error unused!)
//	//	SIGNAL_LOGIN_FAILED    = signal_pool.Get("auth.login_failed")    // -> Send(auth.User, error) (Returned error unused!)
//)
//
