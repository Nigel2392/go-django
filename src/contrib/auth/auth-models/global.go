package models

import (
	"context"
	"database/sql"

	models "github.com/Nigel2392/go-django/src/models"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

var (
	queries DBQuerier
)

type Querier interface {
	Count(ctx context.Context) (int64, error)
	CountMany(ctx context.Context, isActive bool, isAdministrator bool) (int64, error)
	CreateUser(ctx context.Context, email string, username string, password string, firstName string, lastName string, isAdministrator bool, isActive bool) (int64, error)
	DeleteUser(ctx context.Context, id uint64) error
	Retrieve(ctx context.Context, limit int32, offset int32) ([]*User, error)
	RetrieveByEmail(ctx context.Context, email string) (*User, error)
	RetrieveByID(ctx context.Context, id uint64) (*User, error)
	RetrieveByUsername(ctx context.Context, username string) (*User, error)
	RetrieveMany(ctx context.Context, isActive bool, isAdministrator bool, limit int32, offset int32) ([]*User, error)
	UpdateUser(ctx context.Context, email string, username string, password string, firstName string, lastName string, isAdministrator bool, isActive bool, iD uint64) error
	WithTx(tx *sql.Tx) Querier
	Close() error
}

type DBQuerier interface {
	Querier
	DB() *sql.DB
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type dbQuerier struct {
	db *sql.DB
	Querier
}

func (q *dbQuerier) DB() *sql.DB {
	return q.db
}

func (q *dbQuerier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, opts)
}

func NewQueries(db *sql.DB) (DBQuerier, error) {
	if queries != nil {
		return queries, nil
	}

	var backend, err = BackendForDB(db.Driver())
	if err != nil {
		return nil, err
	}

	qs, err := backend.NewQuerySet(db)
	if err != nil {
		return nil, err
	}

	queries = &dbQuerier{
		db:      db,
		Querier: qs,
	}

	return queries, nil
}

var backend = models.NewBackendRegistry[Querier]()
var Register = backend.RegisterForDriver
var BackendForDB = backend.BackendForDB
