package expr

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

const SELF_TABLE = "SELF" // the name of the self table, used in expressions

var (
	exprFieldRegex  = regexp.MustCompile(`!\[([^\]]*)\]`)
	exprTablesRegex = regexp.MustCompile(`#\[([^\]]*)\]`)
	exprValueRegex  = regexp.MustCompile(`\?\[([^\]][0-9]*)\]`)
)

type expressionStatementInfo struct {
	Driver driver.Driver    // the driver used to parse the statement
	Used   bool             // whether the statement has been resolved
	SQL    string           // the parsed SQL, available after Resolve is called
	Fields []*ResolvedField // resolved field names (i.e. ![FieldName] -> table.field_name)
	Tables []string         // resolved table names (i.e. #[SELF] -> table_name or #[FieldName.Relation] -> table_name)
}

type ExpressionStatement struct {
	info      expressionStatementInfo // state information
	Statement string
	Fields    []string
	Tables    []string
	Values    []any
}

func (s *ExpressionStatement) Clone() *ExpressionStatement {
	return &ExpressionStatement{
		info: expressionStatementInfo{
			Driver: s.info.Driver,
			Used:   s.info.Used,
			SQL:    s.info.SQL,
			Fields: slices.Clone(s.info.Fields),
			Tables: slices.Clone(s.info.Tables),
		},
		Statement: s.Statement,
		Fields:    slices.Clone(s.Fields),
		Tables:    slices.Clone(s.Tables),
		Values:    slices.Clone(s.Values),
	}
}

func (s *ExpressionStatement) Resolve(inf *ExpressionInfo) *ExpressionStatement {
	if s.info.Used && len(s.Fields) == len(s.info.Fields) {
		return s
	}

	s.info.Used = true
	s.info.Fields = make([]*ResolvedField, len(s.Fields))
	s.info.Tables = make([]string, len(s.Tables))
	s.info.Driver = inf.Driver

	for i, field := range s.Fields {
		s.info.Fields[i] = inf.ResolveExpressionField(field)
	}

	var tables = make(map[reflect.Type]string)
	for i, table := range s.Tables {
		if table == SELF_TABLE {
			var rTyp = reflect.TypeOf(inf.Model)
			if t, ok := tables[rTyp]; ok {
				s.info.Tables[i] = t
			} else {
				var defs = inf.Model.FieldDefs()
				s.info.Tables[i] = defs.TableName()
				tables[rTyp] = defs.TableName()
			}
			continue
		}

		var current, _, _, _, aliases, _, err = internal.WalkFields(inf.Model, table, inf.AliasGen)
		if err != nil {
			panic(err)
		}

		if len(aliases) > 0 {
			s.info.Tables[i] = aliases[len(aliases)-1]
			continue
		}

		var defs = current.FieldDefs()
		var tableName = defs.TableName()
		s.info.Tables[i] = inf.QuoteIdentifier(tableName)
	}

	if len(s.info.Fields) != len(s.Fields) {
		panic(fmt.Errorf("statement %q has %d fields, but %d was provided", s.Statement, len(s.info.Fields), len(s.Fields)))
	}

	if len(s.info.Tables) != len(s.Tables) {
		panic(fmt.Errorf("statement %q has %d tables, but %d was provided", s.Statement, len(s.info.Tables), len(s.Tables)))
	}

	return s
}

func (s *ExpressionStatement) SQL() (string, []any) {
	if !s.info.Used {
		panic("statement not resolved, call Resolve first")
	}

	if len(s.info.Fields) == 0 && len(s.info.Tables) == 0 {
		return s.Statement, s.Values
	}

	var fieldNames = make([]any, len(s.info.Fields))
	var args = make([]any, 0, len(s.Values)+len(s.info.Fields))
	for i, field := range s.info.Fields {
		fieldNames[i] = field.SQLText
		args = append(args, field.SQLArgs...)
	}

	formatted := fmt.Sprintf(s.Statement, fieldNames...)
	formatted = fmt.Sprintf(formatted, attrs.InterfaceList(s.info.Tables)...)
	return formatted, append(args, s.Values...)
}

// The statement should contain placeholders for the fields and values, which will be replaced with the actual values.
//
// The placeholders for fields should be in the format ![FieldName], and the placeholders for values should be in the format ?[Index],
// or the values should use the regular SQL placeholder directly (database driver dependent).
//
// Example usage:
//
//	 # sets the field name to the first field found in the statement, I.E. ![Field1]:
//
//		stmt, fields, values := ParseExprStatement("![Field1] = ![Age] + ?[1] + ![Height] + ?[2] * ?[1]", 3, 4)
func ParseExprStatement(statement string, value []any) *ExpressionStatement {
	var fields = make([]string, 0)
	for _, m := range exprFieldRegex.FindAllStringSubmatch(statement, -1) {
		if len(m) > 1 {
			fields = append(fields, m[1]) // m[0] is full match, m[1] is capture group
		}
	}

	var tables = make([]string, 0)
	for _, m := range exprTablesRegex.FindAllStringSubmatch(statement, -1) {
		if len(m) > 1 {
			tables = append(tables, m[1]) // m[0] is full match, m[1] is capture group
		}
	}

	var valuesIndices = exprValueRegex.FindAllStringSubmatch(statement, -1)
	var values = make([]any, len(valuesIndices))
	for i, m := range valuesIndices {
		var idx, err = strconv.Atoi(m[1])
		if err != nil {
			panic(fmt.Errorf("invalid index %q in statement %q: %w", m[1], statement, err))
		}

		idx -= 1 // convert to 0-based index
		if idx < 0 || idx >= len(value) {
			panic(fmt.Errorf("index %d out of range in statement %q, index is 1-based and must be between 1 and %d", idx+1, statement, len(value)))
		}

		values[i] = value[idx]
	}

	if len(valuesIndices) == 0 && len(value) > 0 {
		values = make([]any, len(value))
		copy(values, value)
	}

	statement = strings.Replace(statement, "%", "%%%%", -1)
	statement = exprFieldRegex.ReplaceAllString(statement, "%s")
	statement = exprTablesRegex.ReplaceAllString(statement, "%%s")
	statement = exprValueRegex.ReplaceAllString(statement, "?")
	return &ExpressionStatement{
		Statement: statement,
		Fields:    fields,
		Values:    values,
		Tables:    tables,
	}
}
