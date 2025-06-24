package testdb

import "github.com/Nigel2392/go-django-queries/src/drivers"

const (
	retries  int    = 10
	username string = "root"
	password string = "my-secret-pw"

	// make sure they are "used"
	_ = retries
	_ = username
	_ = password
)

type cached struct {
	which string
	db    drivers.Database
}

var global *cached

func Open() (which string, db drivers.Database) {
	if global != nil {
		return global.which, global.db
	}

	which, db = open()
	global = &cached{
		which: which,
		db:    db,
	}

	return which, db
}
