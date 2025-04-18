package queries

import "github.com/Nigel2392/go-django/src/core/errs"

const (
	ErrNoDatabase    errs.Error = "No database connection"
	ErrUnknownDriver errs.Error = "Unknown driver"
	ErrNoTableName   errs.Error = "No table name"
	ErrFieldNull     errs.Error = "Field cannot be null"
	ErrLastInsertId  errs.Error = "Last insert id is not valid"
)
