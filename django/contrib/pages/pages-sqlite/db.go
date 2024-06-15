// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package models_sqlite

import (
	"database/sql"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type Queries struct {
	db models.DBTX
}

func New(db models.DBTX) models.DBQuerier {
	return &Queries{db: db}
}

func (q *Queries) WithTx(tx *sql.Tx) models.Querier {
	return &Queries{
		db: tx,
	}
}
