package expr

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

// LogicalOp represents the logical operator to use in a query.
//
// It is used to compare two values in a logical expression.
// The logical operators are used in the WHERE clause of a SQL query,
// or inside of queryset join conditions.
type LogicalOp string

func (op LogicalOp) Resolve(*ExpressionInfo) Expression { return op }
func (op LogicalOp) Clone() Expression                  { return op }
func (op LogicalOp) SQL(sb *strings.Builder) []any {
	sb.WriteString(" ")
	sb.WriteString(string(op))
	sb.WriteString(" ")
	return []any{}
}

const (
	EQ  LogicalOp = "="
	NE  LogicalOp = "!="
	GT  LogicalOp = ">"
	LT  LogicalOp = "<"
	GTE LogicalOp = ">="
	LTE LogicalOp = "<="
	IN  LogicalOp = "IN"

	ADD LogicalOp = "+"
	SUB LogicalOp = "-"
	MUL LogicalOp = "*"
	DIV LogicalOp = "/"
	MOD LogicalOp = "%"

	BITAND LogicalOp = "&"
	BITOR  LogicalOp = "|"
	BITXOR LogicalOp = "^"
	BITLSH LogicalOp = "<<"
	BITRSH LogicalOp = ">>"
	BITNOT LogicalOp = "~"
)

// ExprOp represents the expression operator to use in a query.
//
// It is used to combine multiple expressions in a logical expression.
type ExprOp string

const (
	OpAnd ExprOp = "AND"
	OpOr  ExprOp = "OR"
)

type LookupExpression = func(sb *strings.Builder) []any

type LookupTransform interface {
	// returns the drivers that support this transform
	// if empty, the transform is supported by all drivers
	Drivers() []driver.Driver

	// name of the transform
	Name() string

	//	// Allowed transform types which can be applied aftr this transform.
	//	// If empty, any transform can be applied after this transform.
	//	// If nil, no transforms are allowed after this transform.
	//	AllowedTransforms() []string
	//
	//	// AllowedLookups returns the lookups that can be applied after this transform.
	//	// 'exact', 'not' and 'is_null' are always allowed, so they dont have to be specified.
	//	// If empty, any lookup can be applied after this transform.
	//	AllowedLookups() []string

	// Resolves the expression and generates a new expressionq
	Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error)
}

type Lookup interface {
	// returns the drivers that support this lookup
	// if empty, the lookup is supported by all drivers
	Drivers() []driver.Driver

	// name of the lookup
	Name() string

	// number of arguments the lookup expects, or -1 for variable arguments
	Arity() (min, max int)

	// normalize the arguments for the lookup
	NormalizeArgs(inf *ExpressionInfo, value []any) ([]any, error)

	// Resolve resolves the lookup for the given field and value
	// and generates an expression for the lookup.
	Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression, args []any) LookupExpression
}

type TableColumn struct {
	// The table or alias to use in the join condition
	// If this is set, the FieldColumn must be specified
	TableOrAlias string

	// The alias for the field in the join condition.
	FieldAlias string

	// RawSQL is the raw SQL to use in the join condition
	RawSQL string

	// The field or column to use in the join condition
	FieldColumn attrs.FieldDefinition

	// ForUpdate specifies if the field should be used in an UPDATE statement
	// This will automatically append "= ?" to the SQL statement
	ForUpdate bool

	// The value to use for the placeholder if the field column is not specified
	Values []any
}

func (c *TableColumn) Validate() error {
	if c.TableOrAlias != "" && (c.ForUpdate || c.RawSQL != "") {
		return fmt.Errorf("cannot format column with (Expression, ForUpdate or RawSQL) and TableOrAlias: %v", c)
	}

	if c.RawSQL == "" && len(c.Values) == 0 && c.FieldColumn == nil && c.FieldAlias == "" {
		return fmt.Errorf("cannot format column with no value, raw SQL, field alias, expression or field column: %v", c)
	}

	if c.ForUpdate && len(c.Values) > 0 {
		return fmt.Errorf("columns do not handle update values, ForUpdate and Value cannot be used together: %v", c)
	}

	if c.ForUpdate && c.RawSQL != "" {
		return fmt.Errorf("columns do support RawSQL and ForUpdate together: %v", c)
	}

	if c.FieldColumn != nil && c.RawSQL != "" {
		return fmt.Errorf("cannot format column with both FieldColumn and RawSQL: %v", c)
	}

	if c.FieldAlias != "" && c.ForUpdate {
		return fmt.Errorf("cannot format column with ForUpdate and FieldAlias: %v", c)
	}

	if c.FieldAlias != "" && len(c.Values) > 0 {
		return fmt.Errorf("cannot format column with FieldAlias and Value: %v", c)
	}

	return nil
}

type ExpressionBuilder interface {
	BuildExpression() Expression
}

type ResolvedExpression interface {
	SQL(sb *strings.Builder) []any
}

type Expression interface {
	ResolvedExpression
	Clone() Expression
	Resolve(inf *ExpressionInfo) Expression
}

type LogicalExpression interface {
	Expression
	Scope(LogicalOp, Expression) LogicalExpression
	EQ(key interface{}, vals ...interface{}) LogicalExpression
	NE(key interface{}, vals ...interface{}) LogicalExpression
	GT(key interface{}, vals ...interface{}) LogicalExpression
	LT(key interface{}, vals ...interface{}) LogicalExpression
	GTE(key interface{}, vals ...interface{}) LogicalExpression
	LTE(key interface{}, vals ...interface{}) LogicalExpression
	ADD(key interface{}, vals ...interface{}) LogicalExpression
	SUB(key interface{}, vals ...interface{}) LogicalExpression
	MUL(key interface{}, vals ...interface{}) LogicalExpression
	DIV(key interface{}, vals ...interface{}) LogicalExpression
	MOD(key interface{}, vals ...interface{}) LogicalExpression
	BITAND(key interface{}, vals ...interface{}) LogicalExpression
	BITOR(key interface{}, vals ...interface{}) LogicalExpression
	BITXOR(key interface{}, vals ...interface{}) LogicalExpression
	BITLSH(key interface{}, vals ...interface{}) LogicalExpression
	BITRSH(key interface{}, vals ...interface{}) LogicalExpression
	BITNOT(key interface{}, vals ...interface{}) LogicalExpression
}

type ClauseExpression interface {
	Expression
	IsNot() bool
	Not(b bool) ClauseExpression
	And(...Expression) ClauseExpression
	Or(...Expression) ClauseExpression
}

type NamedExpression interface {
	Expression
	FieldName() string
}

var logicalOps = map[string]LogicalOp{
	// Equality comparison operators
	"=":  EQ,
	"!=": NE,
	">":  GT,
	"<":  LT,
	">=": GTE,
	"<=": LTE,

	// Arithmetic operators
	"+": ADD,
	"-": SUB,
	"*": MUL,
	"/": DIV,
	"%": MOD,

	// Bitwise operators
	"&":  BITAND,
	"|":  BITOR,
	"^":  BITXOR,
	"<<": BITLSH,
	">>": BITRSH,
	"~":  BITNOT,
}

func Op(op any) (LogicalOp, bool) {
	var rV = reflect.ValueOf(op)
	if rV.Kind() == reflect.String {
		var strOp = rV.String()
		op, ok := logicalOps[strOp]
		return op, ok
	}
	return "", false
}

/*
	The following interfaces must be kept in sync with the interfaces in the 'src/queries.go' file.
*/

// A field can adhere to this interface to indicate that the field should be
// aliased when generating the SQL for the field.
//
// For example: this is used in annotations to alias the field name.
type AliasField interface {
	attrs.Field
	Alias() string
}

// A field can adhere to this interface to indicate that the field should be
// rendered as SQL.
//
// For example: this is used in fields.ExpressionField to render the expression as SQL.
type VirtualField interface {
	attrs.FieldDefinition
	SQL(inf *ExpressionInfo) (string, []any)
}
