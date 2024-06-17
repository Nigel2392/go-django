package models

import (
	"context"
	"database/sql"
)

type BaseBackend struct {
	CreateTableQuery string
	NewQuerier       func(*sql.DB) (Querier, error)
	PreparedQuerier  func(ctx context.Context, d *sql.DB) (Querier, error)
}

func (b *BaseBackend) CreateTable(db *sql.DB) error {
	_, err := db.Exec(b.CreateTableQuery)
	return err
}

func (b *BaseBackend) NewQuerySet(db *sql.DB) (Querier, error) {
	return b.NewQuerier(db)
}

func (b *BaseBackend) Prepare(ctx context.Context, d *sql.DB) (Querier, error) {
	return b.PreparedQuerier(ctx, d)
}
