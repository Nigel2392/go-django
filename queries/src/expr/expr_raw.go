package expr

import (
	"fmt"
	"strings"
)

// RawExpr is a function expression for SQL queries.
// It is used to represent a function call in SQL queries.
//
// It can be used like so:
//
//		RawExpr{
//			// Represent the SQL function call, with each %s being replaced by the corresponding field in fields.
//			Statement:    `SUBSTR(TRIM(%s, " "), 0, 2) = ?``,
//	     	// The fields to be used in the SQL function call. Each field will be replaced by the corresponding value in args.
//			Fields: []string{"myField"},
//			// The arguments to be used in the SQL function call. Each argument will be replaced by the corresponding value in args.
//			Params:   []any{"ab"},
//		}
type RawExpr = RawNamedExpression

func Raw(statement string, value ...any) Expression {
	var stmt = ParseExprStatement(statement, value)
	return &RawExpr{
		Statement: stmt,
	}
}

type RawNamedExpression struct {
	Statement *ExpressionStatement // The statement to be executed, containing placeholders for fields and values.
	Field     string
	not       bool
	used      bool
}

// F creates a new RawNamedExpression or chainExpr with the given statement and values.
// It parses the statement to extract the fields and values, and returns a pointer to the new RawNamedExpression.
//
// The first field in the statement is used as the field name for the expression, and the rest of the fields are used as placeholders for the values.
//
// The statement should contain placeholders for the fields and values, which will be replaced with the actual values.
//
// The placeholders for fields should be in the format ![FieldName], and the placeholders for values should be in the format ?[Index],
// or the values should use the regular SQL placeholder directly (database driver dependent).
//
// Example usage:
//
//	 # sets the field name to the first field found in the statement, I.E. ![Age]:
//
//		expr := F("![Age] + ?[1] + ![Height] + ?[2] * ?[1]", 3, 4)
//		fmt.Println(expr.SQL()) // prints: "table.age + ? + table.height + ? * ?"
//		fmt.Println(expr.Args()) // prints: [3, 4, 3]
//
//	 # sets the field name to the first field found in the statement, I.E. ![Height]:
//
//		expr := F("? + ? + ![Height] + ? * ?", 4, 5, 6, 7)
//		fmt.Println(expr.SQL()) // prints: "? + ? + table.height + ? * ?"
//		fmt.Println(expr.Args()) // prints: [4, 5, 6, 7]
func F(statement any, value ...any) NamedExpression {
	var stmt *ExpressionStatement
	var fieldName string

	if exprBuilder, ok := statement.(ExpressionBuilder); ok {
		statement = exprBuilder.BuildExpression()
	}

	switch v := statement.(type) {
	case string:
		stmt = ParseExprStatement(v, value)
	case NamedExpression:
		return &chainExpr{
			inner: append(
				[]Expression{v},
				expressionFromInterface[Expression](value, false)...,
			),
		}
	default:
		return &chainExpr{
			inner: append(
				expressionFromInterface[Expression](statement, false),
				expressionFromInterface[Expression](value, false)...,
			),
		}

	}

	var fields = stmt.Raw("field")
	if len(fields) > 0 {
		fieldName = fields[0]
	} else {
		panic(fmt.Sprintf(
			"no field found in statement: %q: %+v",
			stmt.Statement, stmt.info.resolver.nodeTexts,
		))
	}

	return &RawNamedExpression{
		Statement: stmt,
		Field:     fieldName,
	}
}

func (e *RawNamedExpression) FieldName() string {
	return e.Field
}

func (e *RawNamedExpression) SQL(sb *strings.Builder) []any {
	var sql, args = e.Statement.SQL()
	sb.WriteString(sql)
	return args
}

func (e *RawNamedExpression) Clone() Expression {
	return &RawNamedExpression{
		Statement: e.Statement.Clone(),
		not:       e.not,
		used:      e.used,
	}
}

func (e *RawNamedExpression) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || e.used {
		return e
	}

	var nE = e.Clone().(*RawNamedExpression)
	nE.used = true
	nE.Statement = nE.Statement.Resolve(inf)

	return nE
}
