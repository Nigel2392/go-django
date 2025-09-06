package expr

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type BaseLookup struct {
	AllowedDrivers []driver.Driver
	Identifier     LookupFilter
	ArgMin         int
	ArgMax         int
	Normalize      func(any) any
	ResolveFunc    func(inf *ExpressionInfo, lhsResolved ResolvedExpression, values []any) LookupExpression
}

func (l *BaseLookup) Drivers() []driver.Driver {
	return l.AllowedDrivers
}

func (l *BaseLookup) Name() string {
	return l.Identifier
}

func (l *BaseLookup) Arity() (min, max int) {
	return l.ArgMin, l.ArgMax
}

func (l *BaseLookup) NormalizeArgs(inf *ExpressionInfo, values []any) ([]any, error) {
	if l.Normalize == nil {
		return values, nil // no normalization function, return as is
	}

	var newValues = make([]any, len(values))
	for i, v := range values {
		newValues[i] = l.Normalize(v)
	}
	return newValues, nil
}

func (l *BaseLookup) Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression, values []any) func(sb *strings.Builder) []any {
	if l.ResolveFunc == nil {
		panic(fmt.Sprintf(
			"lookup %q does not have a resolve function defined, cannot be used",
			l.Identifier,
		))
	}
	return l.ResolveFunc(inf, lhsResolved, values)
}

type LogicalLookup struct {
	BaseLookup
	Operator LogicalOp
}

func (l *LogicalLookup) NormalizeArgs(inf *ExpressionInfo, values []any) ([]any, error) {
	if len(values) != 1 {
		return nil, fmt.Errorf("lookup %s requires exactly one value", l.Identifier)
	}

	var v = values[0]
	switch v := v.(type) {
	case Expression:
		return []any{v.Resolve(inf)}, nil
	case ExpressionBuilder:
		return []any{v.BuildExpression().Resolve(inf)}, nil
	}

	return l.BaseLookup.NormalizeArgs(inf, values)
}

func (l *LogicalLookup) Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression, values []any) func(sb *strings.Builder) []any {
	return func(sb *strings.Builder) []any {
		var lhsExpr strings.Builder
		var args = lhsResolved.SQL(&lhsExpr)
		sb.WriteString(inf.Lookups.FormatLookupCol(
			l.Identifier, lhsExpr.String(),
		))
		sb.WriteString(" ")

		switch arg := values[0].(type) {
		case Expression:
			var inner strings.Builder
			var innerArgs = arg.SQL(&inner)
			args = append(args, innerArgs...)

			var opRHS, _ = inf.Lookups.FormatLogicalOpRHS(
				l.Operator, inf.Lookups.FormatLookupCol(l.Identifier, inner.String()),
			)
			sb.WriteString(opRHS)
		default:
			var opRHS, exprArgs = inf.Lookups.FormatLogicalOpRHS(
				l.Operator, inf.Lookups.FormatLookupCol(l.Identifier, inf.Placeholder), arg,
			)
			sb.WriteString(opRHS)
			args = append(args, exprArgs...)
		}

		return args
	}
}

type PatternLookup struct {
	BaseLookup
	Pattern string
}

func (l *PatternLookup) Arity() (min, max int) {
	return 1, 1
}

func (l *PatternLookup) NormalizeArgs(inf *ExpressionInfo, value []any) ([]any, error) {
	var v = value[0]
	switch v := v.(type) {
	case Expression:
		return []any{v.Resolve(inf)}, nil
	}

	var rVal = reflect.ValueOf(v)
	if rVal.Kind() != reflect.String {
		return nil, fmt.Errorf("lookup %s requires a string value, got %T", l.Identifier, v)
	}

	var valStr = rVal.String()
	var normalizedValue = inf.Lookups.PrepForLikeQuery(
		valStr,
	)

	return []any{normalizedValue}, nil
}

func (l *PatternLookup) Resolve(inf *ExpressionInfo, resolvedExpression ResolvedExpression, values []any) func(sb *strings.Builder) []any {

	return func(sb *strings.Builder) []any {
		var lhsExpr strings.Builder
		var args = resolvedExpression.SQL(
			&lhsExpr,
		)

		sb.WriteString(inf.Lookups.FormatLookupCol(
			l.Identifier, lhsExpr.String(),
		))

		sb.WriteString(" ")

		switch arg := values[0].(type) {
		case Expression:
			var inner strings.Builder
			args = append(
				args, arg.SQL(&inner)...,
			)
			sb.WriteString(inf.Lookups.PatternOpRHS(
				l.Identifier, inner.String(),
			))
		default:
			sb.WriteString(inf.Lookups.FormatOpRHS(
				l.Identifier, inf.Lookups.FormatLookupCol(
					l.Identifier, inf.Placeholder,
				),
			))
			args = append(args, fmt.Sprintf(
				l.Pattern, arg.(string),
			))
		}

		return args
	}
}

type IsNullLookup struct {
	BaseLookup
}

func (l *IsNullLookup) Arity() (min, max int) {
	return 0, 1 // can be called with no arguments, defaults to true
}

func (l *IsNullLookup) NormalizeArgs(inf *ExpressionInfo, values []any) ([]any, error) {
	if len(values) == 0 {
		return []any{true}, nil // assume null if no values are provided
	}
	var isNull bool
	var rVal = reflect.ValueOf(values[0])
	switch rVal.Kind() {
	case reflect.Bool:
		isNull = rVal.Bool()
	default:
		isNull = rVal.IsZero()
	}
	return []any{isNull}, nil
}

func (l *IsNullLookup) Resolve(inf *ExpressionInfo, resolvedExpression ResolvedExpression, values []any) func(sb *strings.Builder) []any {
	return func(sb *strings.Builder) []any {
		var lhsExpr strings.Builder
		var args = resolvedExpression.SQL(&lhsExpr)
		sb.WriteString(inf.Lookups.FormatLookupCol(
			l.Identifier, lhsExpr.String(),
		))
		if values[0].(bool) {
			sb.WriteString(" IS NULL")
		} else {
			sb.WriteString(" IS NOT NULL")
		}
		return args
	}
}

type InLookup struct {
	BaseLookup
}

func (l *InLookup) Arity() (min, max int) {
	return 1, -1 // variable number of arguments
}

func (l *InLookup) NormalizeArgs(inf *ExpressionInfo, values []any) ([]any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("lookup %s requires at least one value", l.Identifier)
	}

	switch v := values[0].(type) {
	case Expression:
		if len(values) > 1 {
			return nil, fmt.Errorf("lookup %s cannot be used with an expression and additional values", l.Identifier)
		}
		return []any{v.Resolve(inf)}, nil
	case ExpressionBuilder:
		if len(values) > 1 {
			return nil, fmt.Errorf("lookup %s cannot be used with an expression builder and additional values", l.Identifier)
		}
		return []any{v.BuildExpression().Resolve(inf)}, nil
	}

	return flattenList(values), nil
}

func (l *InLookup) Resolve(inf *ExpressionInfo, resolvedExpression ResolvedExpression, values []any) func(sb *strings.Builder) []any {
	return func(sb *strings.Builder) []any {
		var lhsExpr strings.Builder
		var args = resolvedExpression.SQL(&lhsExpr)
		sb.WriteString(inf.Lookups.FormatLookupCol(
			l.Identifier, lhsExpr.String(),
		))
		sb.WriteString(" IN (")

		switch v := values[0].(type) {
		case Expression: // handle expression case (maybe subquery)
			args = append(
				args, v.SQL(sb)...,
			)
		default:
			var placeholders = make([]string, len(values))
			for i := range values {
				placeholders[i] = inf.Lookups.FormatLookupCol(
					l.Identifier, inf.Placeholder,
				)
			}
			sb.WriteString(strings.Join(placeholders, ", "))
			args = append(args, values...)
		}

		sb.WriteString(")")
		return args
	}
}

//	RegisterLookup("range", func(d driver.Driver, field string, value []any) (string, []any, error) {
//		if len(value) != 2 {
//			return "", value, fmt.Errorf("RANGE lookup requires exactly two values")
//		}
//		return fmt.Sprintf("%s BETWEEN ? AND ?", field), value, nil
//	})

type RangeLookup struct {
	BaseLookup
}

func (l *RangeLookup) Arity() (min, max int) {
	return 2, 2 // exactly two arguments
}

func (l *RangeLookup) Resolve(inf *ExpressionInfo, resolvedExpression ResolvedExpression, values []any) func(sb *strings.Builder) []any {
	return func(sb *strings.Builder) []any {
		if len(values) != 2 {
			panic(fmt.Sprintf("lookup %s requires exactly two values, got %d", l.Identifier, len(values)))
		}

		var lhsExpr strings.Builder
		var args = resolvedExpression.SQL(&lhsExpr)
		sb.WriteString(inf.Lookups.FormatLookupCol(
			l.Identifier, lhsExpr.String(),
		))

		sb.WriteString(" BETWEEN ")
		sb.WriteString(inf.Placeholder)
		sb.WriteString(" AND ")
		sb.WriteString(inf.Placeholder)

		args = append(args, values[0], values[1])
		return args
	}
}

func flattenList(values []any) []any {
	var inList = make([]any, 0, len(values))
	for _, v := range values {
		var rV = reflect.ValueOf(v)
		if !rV.IsValid() {
			inList = append(inList, nil)
			continue
		}

		if rV.Kind() == reflect.Slice || rV.Kind() == reflect.Array {
			var inner = make([]any, 0, rV.Len())
			for i := 0; i < rV.Len(); i++ {
				var elem = rV.Index(i).Interface()
				inner = append(inner, elem)
			}
			inList = append(inList, flattenList(inner)...)
			continue
		}

		inList = append(inList, normalizeDefinerArg(v))
	}
	return inList
}

func logicalLookup(lookupName LookupFilter, op LogicalOp, normalize func(any) any, allowedDrivers ...driver.Driver) Lookup {
	return &LogicalLookup{
		BaseLookup: BaseLookup{
			ArgMin:         1,
			ArgMax:         1,
			AllowedDrivers: allowedDrivers,
			Identifier:     lookupName,
			Normalize:      normalize,
		},
		Operator: op,
	}
}

func patternLookup(lookupName string, pattern string, allowedDrivers ...driver.Driver) Lookup {
	return &PatternLookup{
		BaseLookup: BaseLookup{
			Identifier:     lookupName,
			AllowedDrivers: allowedDrivers,
		},
		Pattern: pattern,
	}
}

func normalizeDefinerArg(v any) any {
	if definer, ok := v.(attrs.Definer); ok {
		var fieldDefs = definer.FieldDefs()
		var pk = fieldDefs.Primary()
		return pk.GetValue()
	}
	return v
}
