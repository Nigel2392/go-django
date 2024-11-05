package models

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Saver interface {
	Save(c context.Context) error
}

type Updater interface {
	Update(c context.Context) error
}

type Deleter interface {
	Delete(c context.Context) error
}

type Reloader interface {
	Reload(c context.Context) error
}

//	type DBSaver interface {
//		Save(db *sql.DB) error
//	}
//
//	type DBUpdater interface {
//		Update(db *sql.DB) error
//	}
//
//	type DBDeleter interface {
//		Delete(db *sql.DB) error
//	}
//
//	type DBReloader interface {
//		Reload(db *sql.DB) error
//	}

type Model interface {
	attrs.Definer
	Saver
	Deleter
}

type SQLField interface {
	driver.Valuer
	sql.Scanner
}
