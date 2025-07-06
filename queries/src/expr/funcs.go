package expr

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

func init() {
	RegisterFunc("SUM", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("SUM lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("SUM(%s)", sb.String()), args, nil
	})
	RegisterFunc("COUNT", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("COUNT lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("COUNT(%s)", sb.String()), args, nil
	})
	RegisterFunc("AVG", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("AVG lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("AVG(%s)", sb.String()), args, nil
	})
	RegisterFunc("MAX", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("MAX lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("MAX(%s)", sb.String()), args, nil
	})
	RegisterFunc("MIN", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("MIN lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("MIN(%s)", sb.String()), args, nil
	})
	RegisterFunc("COALESCE", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) < 2 {
			return "", []any{}, fmt.Errorf("COALESCE lookup requires at least two values")
		}
		args = make([]any, 0, len(value))
		var coalesce = make([]string, 0, len(value))
		for _, v := range value {
			var sb strings.Builder
			args = append(args, v.SQL(&sb)...)
			coalesce = append(coalesce, sb.String())
		}
		return fmt.Sprintf("COALESCE(%s)", strings.Join(coalesce, ", ")), args, nil
	})
	RegisterFunc("CONCAT", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) < 2 {
			return "", []any{}, fmt.Errorf("CONCAT lookup requires at least two values")
		}
		args = make([]any, 0, len(value))
		var concat = make([]string, 0, len(value))
		for _, v := range value {
			var sb strings.Builder
			args = append(args, v.SQL(&sb)...)
			concat = append(concat, sb.String())
		}
		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return fmt.Sprintf("CONCAT(%s)", strings.Join(concat, ", ")), args, nil
		case *drivers.DriverPostgres:
			return fmt.Sprintf("CONCAT(%s)", strings.Join(concat, ", ")), args, nil
		case *drivers.DriverSQLite:
			return fmt.Sprintf("(%s)", strings.Join(concat, " || ")), args, nil
		}
		return "", nil, fmt.Errorf("unsupported driver for CONCAT: %T", d)
	})
	RegisterFunc("SUBSTR", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("SUBSTR lookup requires exactly one value")
		}
		if len(funcParams) != 2 {
			return "", []any{}, fmt.Errorf("SUBSTR lookup requires exactly two function parameters (start and length)")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)

		var startParam, endParam string
		switch v := funcParams[0].(type) {
		case Expression:
			var startBuilder strings.Builder
			args = append(args, v.SQL(&startBuilder)...)
			startParam = startBuilder.String()
		default:
			if v != nil {
				startParam = fmt.Sprintf("%v", v) // assume it's a constant value
			}
		}

		switch v := funcParams[1].(type) {
		case Expression:
			var endBuilder strings.Builder
			args = append(args, v.SQL(&endBuilder)...)
			endParam = endBuilder.String()
		default:
			if v != nil {
				endParam = fmt.Sprintf("%v", v) // assume it's a constant value
			}
		}

		if startParam == "" {
			return "", nil, fmt.Errorf("SUBSTR lookup requires a valid start parameter")
		}

		if endParam == "" {
			switch d.(type) {
			case *drivers.DriverMySQL, *drivers.DriverMariaDB:
				return fmt.Sprintf("SUBSTRING(%s, %s)", sb.String(), startParam), args, nil
			case *drivers.DriverPostgres:
				return fmt.Sprintf("SUBSTRING(%s FROM %s)", sb.String(), startParam), args, nil
			case *drivers.DriverSQLite:
				return fmt.Sprintf("SUBSTR(%s, %s)", sb.String(), startParam), args, nil
			}
			return "", nil, fmt.Errorf("unsupported driver for SUBSTR: %T", d)
		}

		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return fmt.Sprintf("SUBSTRING(%s, %s, %s)", sb.String(), startParam, endParam), args, nil
		case *drivers.DriverPostgres:
			return fmt.Sprintf("SUBSTRING(%s FROM %s FOR %s)", sb.String(), startParam, endParam), args, nil
		case *drivers.DriverSQLite:
			return fmt.Sprintf("SUBSTR(%s, %s, %s)", sb.String(), startParam, endParam), args, nil
		}

		return "", nil, fmt.Errorf("unsupported driver for SUBSTR: %T", d)
	})
	RegisterFunc("TRIM", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("TRIM lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("TRIM(%s)", sb.String()), args, nil
	})
	RegisterFunc("UPPER", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("UPPER lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("UPPER(%s)", sb.String()), args, nil
	})
	RegisterFunc("LOWER", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("LOWER lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("LOWER(%s)", sb.String()), args, nil
	})
	RegisterFunc("LENGTH", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("LENGTH lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("LENGTH(%s)", sb.String()), args, nil
	})
	RegisterFunc("NOW", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return "NOW()", nil, nil
		case *drivers.DriverPostgres:
			return "CURRENT_TIMESTAMP", nil, nil
		case *drivers.DriverSQLite:
			return "CURRENT_TIMESTAMP", nil, nil
		}
		return "", nil, fmt.Errorf("unsupported driver for NOW: %T", d)
	})
	RegisterFunc("UTCNOW", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return "UTC_TIMESTAMP()", nil, nil
		case *drivers.DriverPostgres:
			return "CURRENT_TIMESTAMP AT TIME ZONE 'UTC'", nil, nil
		case *drivers.DriverSQLite:
			return "julianday('now')", nil, nil
		}
		return "", nil, fmt.Errorf("unsupported driver for UTCNOW: %T", d)
	})
	RegisterFunc("LOCALTIMESTAMP", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return "LOCALTIMESTAMP()", nil, nil
		case *drivers.DriverPostgres:
			return "LOCALTIMESTAMP", nil, nil
		case *drivers.DriverSQLite:
			return "CURRENT_TIMESTAMP", nil, nil
		}
		return "", nil, fmt.Errorf("unsupported driver for LOCALTIMESTAMP: %T", d)
	})
	RegisterFunc("DATE", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("DATE lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		return fmt.Sprintf("DATE(%s)", sb.String()), args, nil
	})
	RegisterFunc("EXISTS", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("EXISTS lookup requires exactly one value")
		}
		var sb strings.Builder
		args = value[0].SQL(&sb)
		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return fmt.Sprintf("EXISTS (%s)", sb.String()), args, nil
		case *drivers.DriverPostgres:
			return fmt.Sprintf("EXISTS (%s)", sb.String()), args, nil
		case *drivers.DriverSQLite:
			return fmt.Sprintf("EXISTS (%s)", sb.String()), args, nil
		}
		return "", nil, fmt.Errorf("unsupported driver for EXISTS: %T", d)
	})
	RegisterFunc("DATE_FORMAT", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 && len(funcParams) != 1 {
			return "", []any{}, fmt.Errorf("DATE_FORMAT lookup requires exactly one value and one format parameter")
		}

		var sb strings.Builder
		args = value[0].SQL(&sb)
		var format string
		switch v := funcParams[0].(type) {
		case Expression:
			var formatBuilder strings.Builder
			args = append(args, v.SQL(&formatBuilder)...)
			format = formatBuilder.String()
		default:
			if v != nil {
				format = fmt.Sprintf("%v", v) // assume it's a constant value
			}
		}

		if format == "" {
			return "", nil, fmt.Errorf("DATE_FORMAT lookup requires a valid format parameter")
		}

		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return fmt.Sprintf("DATE_FORMAT(%s, %s)", sb.String(), format), args, nil
		case *drivers.DriverPostgres:
			return fmt.Sprintf("TO_CHAR(%s, %s)", sb.String(), format), args, nil
		case *drivers.DriverSQLite:
			return fmt.Sprintf("STRFTIME(%s, %s)", format, sb.String()), args, nil
		}

		return "", nil, fmt.Errorf("unsupported driver for DATE_FORMAT: %T", d)
	})
}
