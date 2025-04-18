package queries

import (
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/pkg/errors"
)

func SaveObject[T attrs.Definer](obj T) error {
	var fieldDefs = obj.FieldDefs()
	var primaryField = fieldDefs.Primary()
	var primaryValue = primaryField.GetValue()
	if fields.IsZero(primaryValue) {
		return CreateObject(obj)
	}
	return UpdateObject(obj)
}

func GetObject[T attrs.Definer](obj T) error {
	var queryInfo, err = getQueryInfo(obj)
	if err != nil {
		return err
	}

	var (
		primaryField = queryInfo.definitions.Primary()
		primaryValue = primaryField.GetValue()
		query        strings.Builder
		args         []any
	)

	if fields.IsZero(primaryValue) {
		return errors.Wrapf(
			ErrFieldNull,
			"Primary field %q cannot be null",
			primaryField.Name(),
		)
	}

	query.WriteString("SELECT * FROM ")
	query.WriteString(queryInfo.tableName)
	query.WriteString(" WHERE ")
	query.WriteString(primaryField.ColumnName())
	query.WriteString(" = ?")
	args = append(args, primaryValue)

	var dbSpecific = queryInfo.dbx.Rebind(query.String())
	logger.Debugf("GetObject (%T, %v): %s", obj, primaryValue, dbSpecific)
	return queryInfo.dbx.Get(obj, dbSpecific, args...)
}

func ListObjects[T attrs.Definer](obj T, offset, limit uint64, ordering ...string) ([]T, error) {
	var queryInfo, err = getQueryInfo(obj)
	if err != nil {
		return nil, err
	}

	var (
		primaryField = queryInfo.definitions.Primary()
		primaryName  = primaryField.ColumnName()
		fieldNames   = make([]string, 0, len(queryInfo.fields))
	)
	for _, field := range queryInfo.fields {
		fieldNames = append(fieldNames, field.ColumnName())
	}

	var orderer = models.Orderer{
		Fields: ordering,
		Validate: func(field string) bool {
			return slices.Contains(fieldNames, field)
		},
		Default: "-" + primaryName,
	}

	orderStr, err := orderer.Build()
	if err != nil {
		return nil, err
	}

	var query = strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(fieldNames, ", "))
	query.WriteString(" FROM ")
	query.WriteString(queryInfo.tableName)
	query.WriteString(" ORDER BY ")
	query.WriteString(orderStr)
	query.WriteString(" LIMIT ? OFFSET ?")

	var args = make([]any, 2)
	args[0] = limit
	args[1] = offset

	var dbSpecific = queryInfo.dbx.Rebind(query.String())
	logger.Debugf("ListObjects (%T): %s", obj, dbSpecific)

	var newList = make([]T, 0, limit)
	err = queryInfo.dbx.Select(&newList, dbSpecific, args...)
	return newList, err
}

func CreateObject[T attrs.Definer](obj T) error {
	var queryInfo, err = getQueryInfo(obj)
	if err != nil {
		return err
	}

	var (
		written      bool
		primaryField = queryInfo.definitions.Primary()
		query        strings.Builder
		args         []any
	)

	query.WriteString("INSERT INTO ")
	query.WriteString(queryInfo.tableName)
	query.WriteString(" (")

	for _, field := range queryInfo.fields {
		if field.IsPrimary() || !field.AllowEdit() {
			continue
		}

		var value = field.GetValue()
		if value == nil && !field.AllowNull() {
			return errors.Wrapf(
				ErrFieldNull,
				"Field %q cannot be null",
				field.Name(),
			)
		}

		if written {
			query.WriteString(", ")
		}

		query.WriteString(field.ColumnName())
		args = append(args, value)
		written = true
	}

	query.WriteString(") VALUES (")
	for i := 0; i < len(args); i++ {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("?")
	}
	query.WriteString(")")

	var dbSpecific = queryInfo.dbx.Rebind(query.String())
	logger.Debugf("UpdateObject (%T): %s", obj, dbSpecific)
	result, err := queryInfo.dbx.Exec(dbSpecific, args...)
	if err != nil {
		return err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return err
	}

	return primaryField.SetValue(lastId, true)
}

func UpdateObject[T attrs.Definer](obj T) error {
	var queryInfo, err = getQueryInfo(obj)
	if err != nil {
		return err
	}

	var (
		written      bool
		primaryField = queryInfo.definitions.Primary()
		primaryValue = primaryField.GetValue()
		query        strings.Builder
		args         []any
	)

	if fields.IsZero(primaryValue) {
		return errors.Wrapf(
			ErrFieldNull,
			"Primary field %q cannot be null",
			primaryField.Name(),
		)
	}

	query.WriteString("UPDATE ")
	query.WriteString(queryInfo.tableName)
	query.WriteString(" SET ")

	for _, field := range queryInfo.fields {
		if field.IsPrimary() || !field.AllowEdit() {
			continue
		}

		var value = field.GetValue()
		if value == nil && !field.AllowNull() {
			return errors.Wrapf(
				ErrFieldNull,
				"Field %q cannot be null",
				field.Name(),
			)
		}

		if written {
			query.WriteString(", ")
		}

		query.WriteString(field.ColumnName())
		query.WriteString(" = ?")
		args = append(args, value)
		written = true
	}

	query.WriteString(" WHERE ")
	query.WriteString(primaryField.ColumnName())
	query.WriteString(" = ?")
	args = append(args, primaryValue)

	var dbSpecific = queryInfo.dbx.Rebind(query.String())
	logger.Debugf("UpdateObject (%T, %v): %s", obj, primaryValue, dbSpecific)
	_, err = queryInfo.dbx.Exec(dbSpecific, args...)
	return err
}

func DeleteObject[T attrs.Definer](obj T) error {
	var queryInfo, err = getQueryInfo(obj)
	if err != nil {
		return err
	}

	var (
		primaryField = queryInfo.definitions.Primary()
		primaryValue = primaryField.GetValue()
		query        strings.Builder
		args         []any
	)

	if fields.IsZero(primaryValue) {
		return errors.Wrapf(
			ErrFieldNull,
			"Primary field %q cannot be null",
			primaryField.Name(),
		)
	}

	query.WriteString("DELETE FROM ")
	query.WriteString(queryInfo.tableName)
	query.WriteString(" WHERE ")
	query.WriteString(primaryField.ColumnName())
	query.WriteString(" = ?")
	args = append(args, primaryValue)

	var dbSpecific = queryInfo.dbx.Rebind(query.String())
	logger.Debugf("DeleteObject (%T, %v): %s", obj, primaryValue, dbSpecific)
	_, err = queryInfo.dbx.Exec(dbSpecific, args...)
	return err
}
