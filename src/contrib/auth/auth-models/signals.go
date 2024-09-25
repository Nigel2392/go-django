package models

import "github.com/Nigel2392/go-signals"

var (
	user_signal_pool          = signals.NewPool[*User]()
	id_signal_pool            = signals.NewPool[uint64]()
	SIGNAL_BEFORE_USER_CREATE = user_signal_pool.Get("user.before_create") // -> Send(auth.User) (Returned error unused!)
	SIGNAL_AFTER_USER_CREATE  = user_signal_pool.Get("user.after_create")  // -> Send(auth.User) (Returned error unused!)

	SIGNAL_BEFORE_USER_UPDATE = user_signal_pool.Get("user.before_update") // -> Send(auth.User) (Returned error unused!)
	SIGNAL_AFTER_USER_UPDATE  = user_signal_pool.Get("user.after_update")  // -> Send(auth.User) (Returned error unused!)

	SIGNAL_BEFORE_USER_DELETE = id_signal_pool.Get("user.before_delete") // -> Send(int64) (Returned error unused!)
	SIGNAL_AFTER_USER_DELETE  = id_signal_pool.Get("user.after_delete")  // -> Send(int64) (Returned error unused!)

)
