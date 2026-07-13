package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

func TestFuncConcat(t *testing.T) {
	info := getTestInfo() // DriverSQLite
	f := expr.CONCAT(expr.Field("FirstName"), expr.Field("LastName"))
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()

	expectedSQL := fixSQL(info, "(`test_model`.`first_name` || `test_model`.`last_name`)")
	switch testdb.ENGINE {
	case "mysql", "mysql_local", "mariadb":
		expectedSQL = fixSQL(info, "CONCAT(`test_model`.`first_name`, `test_model`.`last_name`)")
	case "postgres":
		expectedSQL = fixSQL(info, "CONCAT(`test_model`.`first_name`, `test_model`.`last_name`)")
	}

	if sql != expectedSQL {
		t.Errorf("Expected CONCAT output, got: %s", sql)
	}
}

func TestFuncCoalesce(t *testing.T) {
	info := getTestInfo()
	f := expr.COALESCE(expr.Field("Nickname"), expr.Field("FirstName"), expr.Value("Unknown"))
	resolved := f.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()

	expectedSQL := fixSQL(info, "COALESCE(`test_model`.`nickname`, `test_model`.`first_name`, ?)")
	if testdb.ENGINE == "postgres" {
		expectedSQL = fixSQL(info, "COALESCE(`test_model`.`nickname`, `test_model`.`first_name`, ?::TEXT)")
	}
	if sql != expectedSQL {
		t.Errorf("Expected COALESCE output, got: %s", sql)
	}
	if len(args) != 1 || args[0] != "Unknown" {
		t.Errorf("Unexpected args: %v", args)
	}
}

func TestFuncSum(t *testing.T) {
	info := getTestInfo()
	f := expr.SUM(expr.Field("Score"))
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()

	if sql != fixSQL(info, "SUM(`test_model`.`score`)") {
		t.Errorf("Expected SUM(`test_model`.`score`), got: %s", sql)
	}
}

func TestFuncSubstr(t *testing.T) {
	info := getTestInfo()
	f := expr.SUBSTR(expr.Field("Name"), 1, 5)
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()

	expectedSQL := fixSQL(info, "SUBSTR(`test_model`.`name`, 1, 5)")
	switch testdb.ENGINE {
	case "mysql", "mysql_local", "mariadb":
		expectedSQL = fixSQL(info, "SUBSTRING(`test_model`.`name`, 1, 5)")
	case "postgres":
		expectedSQL = fixSQL(info, "SUBSTRING(`test_model`.`name` FROM 1 FOR 5)")
	}

	if sql != expectedSQL {
		t.Errorf("Expected SUBSTR output, got: %s", sql)
	}
}
