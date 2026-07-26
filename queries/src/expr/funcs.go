package expr

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

func init() {
	RegisterFunc("SUM", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("SUM lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("SUM(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("COUNT", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("COUNT lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("COUNT(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("AVG", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("AVG lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("AVG(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("MAX", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("MAX lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("MAX(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("MIN", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("MIN lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("MIN(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("COALESCE", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) < 2 {
			return "", []any{}, fmt.Errorf("COALESCE lookup requires at least two values")
		}
		args = make([]any, 0, len(value))
		var coalesce = make([]string, 0, len(value))
		for _, v := range value {
			var sb builder.BaseBuilder
			v.SQL(&sb)
			args = append(args, sb.Vars...)
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
			var sb builder.BaseBuilder
			v.SQL(&sb)
			args = append(args, sb.Vars...)
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
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		args = sb.Vars

		var startParam, endParam string
		switch v := funcParams[0].(type) {
		case Expression:
			var startBuilder builder.BaseBuilder
			v.SQL(&startBuilder)
			args = append(args, startBuilder.Vars...)
			startParam = startBuilder.String()
		default:
			if v != nil {
				startParam = fmt.Sprintf("%v", v) // assume it's a constant value
			}
		}

		switch v := funcParams[1].(type) {
		case Expression:
			var endBuilder builder.BaseBuilder
			v.SQL(&endBuilder)
			args = append(args, endBuilder.Vars...)
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
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("TRIM(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("UPPER", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("UPPER lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("UPPER(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("LOWER", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("LOWER lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("LOWER(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("LENGTH", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("LENGTH lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("LENGTH(%s)", sb.String()), sb.Vars, nil
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
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		return fmt.Sprintf("DATE(%s)", sb.String()), sb.Vars, nil
	})
	RegisterFunc("EXISTS", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 {
			return "", []any{}, fmt.Errorf("EXISTS lookup requires exactly one value")
		}
		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		switch d.(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB:
			return fmt.Sprintf("EXISTS (%s)", sb.String()), sb.Vars, nil
		case *drivers.DriverPostgres:
			return fmt.Sprintf("EXISTS (%s)", sb.String()), sb.Vars, nil
		case *drivers.DriverSQLite:
			return fmt.Sprintf("EXISTS (%s)", sb.String()), sb.Vars, nil
		}
		return "", nil, fmt.Errorf("unsupported driver for EXISTS: %T", d)
	})
	RegisterFunc("DATE_FORMAT", func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error) {
		if len(value) != 1 && len(funcParams) != 1 {
			return "", []any{}, fmt.Errorf("DATE_FORMAT lookup requires exactly one value and one format parameter")
		}

		var sb builder.BaseBuilder
		value[0].SQL(&sb)
		args = sb.Vars
		var format string
		switch v := funcParams[0].(type) {
		case Expression:
			var formatBuilder builder.BaseBuilder
			v.SQL(&formatBuilder)
			args = append(args, formatBuilder.Vars...)
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
			return fmt.Sprintf("DATE_FORMAT(%s, '%s')", sb.String(), format), args, nil
		case *drivers.DriverPostgres:
			return fmt.Sprintf("TO_CHAR(%s, '%s')", sb.String(), format), args, nil
		case *drivers.DriverSQLite:
			return fmt.Sprintf("STRFTIME('%s', %s)", format, sb.String()), args, nil
		}

		return "", nil, fmt.Errorf("unsupported driver for DATE_FORMAT: %T", d)
	})
}
