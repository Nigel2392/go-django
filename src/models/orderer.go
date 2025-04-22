package models

import (
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/src/core/errs"
)

var (
	ErrDefault = errs.Error(
		"default ordering is invalid",
	)
	ErrEmpty = errs.Error(
		"ordering field cannot be empty",
	)
	ErrFormat = errs.Error(
		"ordering field must be in the format <field> <ASC|DESC> or <field>",
	)
)

// A struct for creating a valid SQL ordering string which should be compatible with most databases.
//
// # The ordering field is a list of strings which can be prefixed with a minus sign (-) to indicate descending order.
//
// The ordering field is validated with the Validate function, which should return true if the field is valid for use in said database.
//
// The default field is used if no ordering fields are provided, and is validated with the Validate function as well,
// it can also be prefixed with a minus sign (-) to indicate descending order.
type Orderer struct {
	TableName string
	Quote     string
	Fields    []string
	Validate  func(string) bool
	Default   string
}

func (o *Orderer) stringify(ordering string) (sort string, field string) {
	var ord = "ASC"
	if strings.HasPrefix(ordering, "-") {
		ord = "DESC"
		ordering = strings.TrimPrefix(ordering, "-")
	}
	return ord, ordering
}

func (o *Orderer) validate(ordering string) error {
	if ordering == "" {
		return fmt.Errorf("ordering field %q cannot be empty", ordering)
	}
	if !o.Validate(ordering) {
		return fmt.Errorf("invalid ordering field %s, must be one of %v", ordering, o.Fields)
	}
	return nil
}

func (o *Orderer) Build() (string, error) {
	var b strings.Builder
	for i, ordering := range o.Fields {
		var ord, field = o.stringify(ordering)

		if err := o.validate(field); err != nil {
			return "", err
		}

		if o.TableName != "" {
			b.Grow(1 + (len(o.Quote) * 4) + len(o.TableName) + len(field) + len(ord))
			b.WriteString(o.Quote)
			b.WriteString(o.TableName)
			b.WriteString(o.Quote)
			b.WriteString(".")
			b.WriteString(o.Quote)
			b.WriteString(field)
			b.WriteString(o.Quote)
		} else {
			b.Grow(1 + (len(o.Quote) * 2) + len(field))
			b.WriteString(o.Quote)
			b.WriteString(field)
			b.WriteString(o.Quote)
		}

		b.WriteString(" ")
		b.WriteString(ord)

		if i < len(o.Fields)-1 {
			b.WriteString(", ")
		}
	}

	// there are ordering fields, return the ordering string
	if b.Len() > 0 {
		return b.String(), nil
	}

	// no ordering fields, check if a default ordering is set
	if o.Default == "" {
		return "", errs.WrapErrors(
			ErrDefault, ErrEmpty,
		)
	}

	// default ordering is set, check if it is valid
	var defField string
	var defOrd = "ASC"
	if strings.Contains(o.Default, " ") {

		var split = strings.SplitN(o.Default, " ", 2)

		if len(split) != 2 {
			return "", errs.WrapErrors(
				ErrDefault, ErrFormat,
			)
		}

		var field = split[0]
		if err := o.validate(field); err != nil {
			return "", errs.WrapErrors(
				err, ErrDefault, ErrFormat,
			)
		}

		if split[1] != "ASC" && split[1] != "DESC" {
			return "", errs.WrapErrors(
				ErrDefault, ErrFormat,
			)
		}

		defField = field
		defOrd = split[1]
	} else {
		defOrd, defField = o.stringify(o.Default)
		if err := o.validate(defField); err != nil {
			return "", errs.WrapErrors(
				err, ErrDefault, ErrFormat,
			)
		}
	}

	b.WriteString(defField)
	b.WriteString(" ")
	b.WriteString(defOrd)

	return b.String(), nil
}
