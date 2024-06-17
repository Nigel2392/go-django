package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
)

type Backend interface {
	CreateTable(*sql.DB) error
	NewQuerySet(*sql.DB) (Querier, error)
	Prepare(ctx context.Context, db *sql.DB) (Querier, error)
}

type backendRegistry struct {
	backends map[reflect.Type]Backend
}

func (b *backendRegistry) registerForDriver(driver any, backend Backend) {
	var rTyp = reflect.TypeOf(driver)
	if rTyp.Kind() == reflect.Pointer {
		rTyp = rTyp.Elem()
	}

	if b.backends == nil {
		b.backends = make(map[reflect.Type]Backend)
	}

	if _, ok := b.backends[rTyp]; ok {
		panic("backend already registered")
	}

	b.backends[rTyp] = backend
}

func (b *backendRegistry) backendForDB(d driver.Driver) (Backend, bool) {
	var rTyp = reflect.TypeOf(d)
	if rTyp.Kind() == reflect.Pointer {
		rTyp = rTyp.Elem()
	}

	var backend, ok = b.backends[rTyp]
	return backend, ok
}

var (
	registry   = &backendRegistry{}
	Register   = registry.registerForDriver
	GetBackend = registry.backendForDB
)
