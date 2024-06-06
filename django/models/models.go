package models

import (
	"database/sql"
	"database/sql/driver"

	"github.com/Nigel2392/django/core/attrs"
)

type Saver interface {
	Save() error
}

type Updater interface {
	Update() error
}

type Deleter interface {
	Delete() error
}

type Reloader interface {
	Reload() error
}

type Model interface {
	attrs.Definer
	Saver
	Updater
	Deleter
}

type SQLField interface {
	driver.Valuer
	sql.Scanner
}
