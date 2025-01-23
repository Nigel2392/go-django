package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
)

var (
	ErrBackendNotFound = errors.New("backend not found")
)

type Backend[T any] interface {
	CreateTable(*sql.DB) error
	NewQuerySet(*sql.DB) (T, error)
	Prepare(ctx context.Context, db *sql.DB) (T, error)
}

type BaseQuerier[T any] interface {
	WithTx(tx *sql.Tx) T
	Close() error
}

type Registry[T any] interface {
	RegisterForDriver(driver any, backend Backend[T])
	BackendForDB(driver.Driver) (Backend[T], error)
}

type backendRegistry[T BaseQuerier[T]] struct {
	backends map[reflect.Type]Backend[T]
}

func NewBackendRegistry[T BaseQuerier[T]]() Registry[T] {
	return &backendRegistry[T]{}
}

func (b *backendRegistry[T]) RegisterForDriver(driver any, backend Backend[T]) {
	var rTyp = reflect.TypeOf(driver)
	if rTyp.Kind() == reflect.Pointer {
		rTyp = rTyp.Elem()
	}

	if b.backends == nil {
		b.backends = make(map[reflect.Type]Backend[T])
	}

	if _, ok := b.backends[rTyp]; ok {
		panic("backend already registered")
	}

	b.backends[rTyp] = backend
}

func (b *backendRegistry[T]) BackendForDB(d driver.Driver) (Backend[T], error) {
	var rTyp = reflect.TypeOf(d)
	if rTyp.Kind() == reflect.Pointer {
		rTyp = rTyp.Elem()
	}

	var backend, ok = b.backends[rTyp]
	if !ok {
		return nil, ErrBackendNotFound
	}
	return backend, nil
}
