package models

import (
	"context"
	"database/sql"
)

type BaseBackend[T any] struct {
	CreateTableQuery string
	CreateTableFn    func(*sql.DB) error
	NewQuerier       func(*sql.DB) (T, error)
	PreparedQuerier  func(ctx context.Context, d *sql.DB) (T, error)
}

func (b *BaseBackend[T]) CreateTable(db *sql.DB) error {
	if b.CreateTableQuery == "" {
		return nil
	}
	if b.CreateTableFn != nil {
		return b.CreateTableFn(db)
	}
	_, err := db.Exec(b.CreateTableQuery)
	return err
}

func (b *BaseBackend[T]) NewQuerySet(db *sql.DB) (T, error) {
	return b.NewQuerier(db)
}

func (b *BaseBackend[T]) Prepare(ctx context.Context, d *sql.DB) (T, error) {
	if b.PreparedQuerier == nil {
		return b.NewQuerySet(d)
	}
	return b.PreparedQuerier(ctx, d)
}
