package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

type funcTestCase struct {
	Name         string
	Fn           expr.Expression
	GenericSQL   string
	SqliteSQL    string
	MysqlSQL     string
	PostgresSQL  string
	ExpectedArgs []any
}

func (tc *funcTestCase) getExpected(info *expr.ExpressionInfo) string {
	var expected = tc.GenericSQL
	switch testdb.ENGINE {
	case "sqlite", "sqlite3":
		if tc.SqliteSQL != "" {
			expected = tc.SqliteSQL
		}
	case "mysql", "mysql_local", "mariadb":
		if tc.MysqlSQL != "" {
			expected = tc.MysqlSQL
		}
	case "postgres":
		if tc.PostgresSQL != "" {
			expected = tc.PostgresSQL
		}
	}
	return fixSQL(info, expected)
}

func TestFuncsImplTableDriven(t *testing.T) {
	info := getTestInfo()

	tests := []funcTestCase{
		{
			Name:       "SUM",
			Fn:         expr.SUM("Score"),
			GenericSQL: "SUM(`test_model`.`score`)",
		},
		{
			Name:       "COUNT",
			Fn:         expr.COUNT("Score"),
			GenericSQL: "COUNT(`test_model`.`score`)",
		},
		{
			Name:       "AVG",
			Fn:         expr.AVG("Score"),
			GenericSQL: "AVG(`test_model`.`score`)",
		},
		{
			Name:       "MAX",
			Fn:         expr.MAX("Score"),
			GenericSQL: "MAX(`test_model`.`score`)",
		},
		{
			Name:       "MIN",
			Fn:         expr.MIN("Score"),
			GenericSQL: "MIN(`test_model`.`score`)",
		},
		{
			Name:         "COALESCE",
			Fn:           expr.COALESCE("Score", expr.V(0)),
			GenericSQL:   "COALESCE(`test_model`.`score`, ?)",
			PostgresSQL:  "COALESCE(`test_model`.`score`, ?::INT)",
			ExpectedArgs: []any{0},
		},
		{
			Name:       "CONCAT",
			Fn:         expr.CONCAT("FirstName", "LastName"),
			GenericSQL: "CONCAT(`test_model`.`first_name`, `test_model`.`last_name`)",
			SqliteSQL:  "(`test_model`.`first_name` || `test_model`.`last_name`)",
		},
		{
			Name:        "SUBSTR",
			Fn:          expr.SUBSTR("Name", 1, 5),
			GenericSQL:  "SUBSTR(`test_model`.`name`, 1, 5)",
			MysqlSQL:    "SUBSTRING(`test_model`.`name`, 1, 5)",
			PostgresSQL: "SUBSTRING(`test_model`.`name` FROM 1 FOR 5)",
		},
		{
			Name:         "EXISTS",
			Fn:           expr.EXISTS(expr.V(1)),
			GenericSQL:   "EXISTS (?)",
			PostgresSQL:  "EXISTS (?::INT)",
			ExpectedArgs: []any{1},
		},
		{
			Name:       "UPPER",
			Fn:         expr.UPPER("Name"),
			GenericSQL: "UPPER(`test_model`.`name`)",
		},
		{
			Name:       "LOWER",
			Fn:         expr.LOWER("Name"),
			GenericSQL: "LOWER(`test_model`.`name`)",
		},
		{
			Name:       "LENGTH",
			Fn:         expr.LENGTH("Name"),
			GenericSQL: "LENGTH(`test_model`.`name`)",
		},
		{
			Name:        "NOW",
			Fn:          expr.NOW(),
			GenericSQL:  "NOW()",
			SqliteSQL:   "CURRENT_TIMESTAMP",
			PostgresSQL: "CURRENT_TIMESTAMP",
		},
		{
			Name:        "UTCNOW",
			Fn:          expr.UTCNOW(),
			GenericSQL:  "UTC_TIMESTAMP()",
			SqliteSQL:   "julianday('now')",
			PostgresSQL: "CURRENT_TIMESTAMP AT TIME ZONE 'UTC'",
		},
		{
			Name:        "LOCALTIMESTAMP",
			Fn:          expr.LOCALTIMESTAMP(),
			GenericSQL:  "LOCALTIMESTAMP()",
			SqliteSQL:   "CURRENT_TIMESTAMP",
			PostgresSQL: "LOCALTIMESTAMP",
		},
		{
			Name:       "DATE",
			Fn:         expr.DATE("Name"),
			GenericSQL: "DATE(`test_model`.`name`)",
		},
		{
			Name:        "DATE_FORMAT",
			Fn:          expr.DATE_FORMAT("Name", "%Y"),
			GenericSQL:  "DATE_FORMAT(`test_model`.`name`, '%Y')",
			SqliteSQL:   "STRFTIME('%Y', `test_model`.`name`)",
			PostgresSQL: "TO_CHAR(`test_model`.`name`, '%Y')",
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			resolved := tc.Fn.Resolve(info)
			var sb strings.Builder
			args := resolved.SQL(&sb)
			sql := sb.String()

			expectedSQL := tc.getExpected(info)
			if sql != expectedSQL {
				t.Errorf("[%s] Expected %s, got: %s", testdb.ENGINE, expectedSQL, sql)
			}
			if len(args) != len(tc.ExpectedArgs) {
				t.Errorf("[%s] Expected %d args, got %d", testdb.ENGINE, len(tc.ExpectedArgs), len(args))
			} else {
				for i := range args {
					if args[i] != tc.ExpectedArgs[i] {
						t.Errorf("Arg %d mismatch: expected %v, got %v", i, tc.ExpectedArgs[i], args[i])
					}
				}
			}
		})
	}
}
