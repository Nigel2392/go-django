package models

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

type BaseBackend[T any] struct {
	CreateTableQuery string
	CreateTableFn    func(drivers.Database) error
	NewQuerier       func(drivers.Database) (T, error)
	PreparedQuerier  func(ctx context.Context, d drivers.Database) (T, error)
}

func (b *BaseBackend[T]) CreateTable(db drivers.Database) error {
	if b.CreateTableFn != nil {
		return b.CreateTableFn(db)
	}
	if b.CreateTableQuery == "" {
		return fmt.Errorf("no CreateTableQuery or CreateTableFn provided")
	}
	_, err := db.ExecContext(context.Background(), b.CreateTableQuery)
	return err
}

func (b *BaseBackend[T]) NewQuerySet(db drivers.Database) (T, error) {
	return b.NewQuerier(db)
}

func (b *BaseBackend[T]) Prepare(ctx context.Context, d drivers.Database) (T, error) {
	if b.PreparedQuerier == nil {
		return b.NewQuerySet(d)
	}
	return b.PreparedQuerier(ctx, d)
}
