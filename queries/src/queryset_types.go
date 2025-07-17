package queries

import (
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type (
	// JoinType represents the type of join to use in a query.
	//
	// It is used to specify how to join two tables in a query.
	JoinType string
)

const (
	TypeJoinLeft  JoinType = "LEFT JOIN"
	TypeJoinRight JoinType = "RIGHT JOIN"
	TypeJoinInner JoinType = "INNER JOIN"
	TypeJoinFull  JoinType = "FULL JOIN"
	TypeJoinCross JoinType = "CROSS JOIN"
)

// A table represents a database table.
type Table struct {
	Name  string
	Alias string
}

// OrderBy represents an order by clause in a query.
//
// It contains the table to order by, the field to order by, an optional alias for the field,
// and a boolean indicating whether to order in descending order.
//
// It is used to specify how to order the results of a query.
type OrderBy struct {
	Column expr.TableColumn // The field to order by
	Desc   bool
}

type JoinDefCondition struct {
	ConditionA expr.TableColumn  // The first condition to join on
	ConditionB expr.TableColumn  // The second condition to join on
	Operator   expr.LogicalOp    // The operator to use to join the two conditions
	Next       *JoinDefCondition // The next join condition, if any
}

func writeCol(sb *strings.Builder, col *expr.TableColumn) {
	var (
		length int = 0
		list       = make([]string, 0, 6)
	)

	if col.TableOrAlias != "" {
		length += len(col.TableOrAlias)
		list = append(list, col.TableOrAlias)
	}
	if col.FieldAlias != "" {
		length += len(col.FieldAlias)
		list = append(list, col.FieldAlias)
	}
	if col.RawSQL != "" {
		length += len(col.RawSQL)
		list = append(list, col.RawSQL)
	}
	if col.FieldColumn != nil {
		var n = col.FieldColumn.ColumnName()
		length += len(n)
		list = append(list, n)
	}
	if col.ForUpdate {
		var n = "FOR_UPDATE"
		length += len(n)
		list = append(list, n)
	}
	for _, v := range col.Values {
		var n = fmt.Sprint(v)
		length += len(n)
		list = append(list, n)
	}

	sb.Grow(length + (len(list) - 1))

	for i, str := range list {
		if i > 0 {
			sb.WriteString(":")
		}
		sb.WriteString(str)
	}
}

func (j *JoinDefCondition) String() string {
	var sb = &strings.Builder{}
	var curr = j
	for curr != nil {

		writeCol(sb, &curr.ConditionA)
		sb.WriteString(string(curr.Operator))
		writeCol(sb, &curr.ConditionB)
		curr = curr.Next

	}
	return sb.String()
}

// JoinDef represents a join definition in a query.
//
// It contains the table to join, the type of join, and the fields to join on.
//
// See [JoinType] for different types of joins.
// See [expr.LogicalOp] for different logical operators.
type JoinDef struct {
	Table            Table
	TypeJoin         JoinType
	JoinDefCondition *JoinDefCondition
}

// FieldInfo represents information about a field in a query.
//
// It is both used by the QuerySet and by the QueryCompiler.
type FieldInfo[FieldType attrs.FieldDefinition] struct {
	Annotating  bool // Whether the field is being annotated
	SourceField FieldType
	Model       attrs.Definer
	RelType     attrs.RelationType
	Table       Table
	Chain       []string
	Fields      []FieldType
	Through     *FieldInfo[FieldType]
}

func (f *FieldInfo[T]) WriteFields(sb *strings.Builder, inf *expr.ExpressionInfo) []any {
	var args = make([]any, 0, len(f.Fields))
	var written bool

	// If the field has a through relation, write the fields of the through relation first
	//
	// This logic matches in [getScannableFields]
	if f.Through != nil {
		for _, field := range f.Through.Fields {
			if written {
				sb.WriteString(", ")
			}

			var a, _, ok = f.Through.WriteField(sb, inf, field, false)
			written = ok || written
			if !ok {
				continue
			}

			args = append(args, a...)
		}
	}

	for _, field := range f.Fields {
		if written {
			sb.WriteString(", ")
		}

		var a, _, ok = f.WriteField(sb, inf, field, false)
		written = ok || written
		if !ok {
			continue
		}

		args = append(args, a...)
	}

	return args
}

func (f *FieldInfo[T]) WriteUpdateFields(sb *strings.Builder, inf *expr.ExpressionInfo) []any {
	var args = make([]any, 0, len(f.Fields))
	var written bool

	// If the field has a through relation, write the fields of the through relation first
	//
	// This logic matches in [getScannableFields]
	if f.Through != nil {
		for _, field := range f.Through.Fields {
			if written {
				sb.WriteString(", ")
			}

			var a, _, ok = f.Through.WriteField(sb, inf, field, true)
			written = ok || written
			if !ok {
				continue
			}

			args = append(args, a...)
		}
	}

	for _, field := range f.Fields {
		if written {
			sb.WriteString(", ")
		}

		var a, _, ok = f.WriteField(sb, inf, field, true)
		written = ok || written
		if !ok {
			continue
		}

		args = append(args, a...)
	}

	return args
}

func (f *FieldInfo[T]) WriteField(sb *strings.Builder, inf *expr.ExpressionInfo, field attrs.FieldDefinition, forUpdate bool) (args []any, isSQL, written bool) {
	var fieldAlias string
	if ve, ok := field.(AliasField); ok && !forUpdate {
		fieldAlias = ve.Alias()
	}

	var tableAlias string
	if f.Table.Alias == "" {
		tableAlias = f.Table.Name
	} else {
		tableAlias = f.Table.Alias
	}
	var col = &expr.TableColumn{}
	if ve, ok := field.(VirtualField); ok && inf.Model != nil {
		var rawSql, a = ve.SQL(inf)
		if rawSql == "" {
			return nil, true, false
		}

		col.RawSQL = rawSql

		if fieldAlias != "" && !forUpdate && inf.SupportsAsExpr {
			col.FieldAlias = inf.Resolver.Alias().GetFieldAlias(
				tableAlias, fieldAlias,
			)
		}

		var fmtSql, extra = inf.FormatField(col)
		sb.WriteString(fmtSql)
		args = append(args, a...)
		args = append(args, extra...)
		return args, true, true
	}

	if !forUpdate {
		col.TableOrAlias = tableAlias
	}

	col.FieldColumn = field
	col.ForUpdate = forUpdate

	var fmtSql, _ = inf.FormatField(col)
	sb.WriteString(fmtSql)

	return []any{}, false, true
}

func generateFieldInfoKey[FieldType attrs.FieldDefinition](field *FieldInfo[FieldType]) (string, error) {
	var key string
	switch {
	case field.Model == nil:
		key = "__annotations__"
	case len(field.Fields) == 1:
		var fld = field.Fields[0]
		var fieldName = fld.Name()

		if aliasField, ok := any(fld).(AliasField); ok {
			var alias = aliasField.Alias()
			if alias != "" {
				fieldName = alias
			}
		}

		key = fieldName
	case len(field.Fields) > 1:
		key = "*"
	default:
		return "", fmt.Errorf(
			"field must have at least one field, got %d",
			len(field.Fields),
		)
	}

	if len(field.Chain) > 0 {
		key = fmt.Sprintf(
			"%s.%s",
			strings.Join(field.Chain, "."),
			key,
		)
	}

	return key, nil
}
