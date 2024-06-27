package models

import (
	"context"
	"database/sql"
)

type BaseBackend[T any] struct {
	CreateTableQuery string
	NewQuerier       func(*sql.DB) (T, error)
	PreparedQuerier  func(ctx context.Context, d *sql.DB) (T, error)
}

func (b *BaseBackend[T]) CreateTable(db *sql.DB) error {
	_, err := db.Exec(b.CreateTableQuery)
	return err
}

func (b *BaseBackend[T]) NewQuerySet(db *sql.DB) (T, error) {
	return b.NewQuerier(db)
}

func (b *BaseBackend[T]) Prepare(ctx context.Context, d *sql.DB) (T, error) {
	return b.PreparedQuerier(ctx, d)
}
