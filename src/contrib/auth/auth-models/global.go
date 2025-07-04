package models

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	models "github.com/Nigel2392/go-django/src/models"
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
	WithTx(tx drivers.Transaction) Querier
	Close() error
}

type DBQuerier interface {
	Querier
	DB() drivers.Database
	Begin(ctx context.Context) (drivers.Transaction, error)
}

type dbQuerier struct {
	db drivers.Database
	Querier
}

func (q *dbQuerier) DB() drivers.Database {
	return q.db
}

func (q *dbQuerier) Begin(ctx context.Context) (drivers.Transaction, error) {
	return q.db.Begin(ctx)
}

func NewQueries(db drivers.Database) (DBQuerier, error) {
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
