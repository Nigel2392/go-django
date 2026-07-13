package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

func TestCastStringGeneration(t *testing.T) {
	info := getTestInfo()
	c := expr.CastString(expr.Field("Age"), 255)
	resolved := c.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)

	expectedType := "TEXT"
	switch testdb.ENGINE {
	case "mysql", "mysql_local", "mariadb":
		expectedType = "CHAR(255)"
	case "postgres":
		expectedType = "VARCHAR(255)"
	}

	expectedSQL := fixSQL(info, "CAST(`test_model`.`age` AS "+expectedType+")")
	if !strings.EqualFold(strings.TrimSpace(sb.String()), strings.TrimSpace(expectedSQL)) {
		t.Errorf("Expected %s output, got: %s", expectedType, sb.String())
	}
	if len(args) != 0 {
		t.Errorf("Expected 0 args for F cast, got %d", len(args))
	}
}

func TestCastFloatGeneration(t *testing.T) {
	info := getTestInfo()
	c := expr.CastFloat(expr.Field("Score"), 10, 2)
	resolved := c.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)

	expectedType := "REAL"
	switch testdb.ENGINE {
	case "mysql", "mysql_local", "mariadb":
		expectedType = "DECIMAL(10,2)"
	case "postgres":
		expectedType = "NUMERIC(10,2)" // or NUMERIC
	}

	expectedSQL := fixSQL(info, "CAST(`test_model`.`score` AS "+expectedType+")")
	if !strings.EqualFold(strings.TrimSpace(sb.String()), strings.TrimSpace(expectedSQL)) {
		t.Errorf("Expected %s output, got: %s", expectedType, sb.String())
	}
	if len(args) != 0 {
		t.Errorf("Expected 0 args for CastFloat on a field, got %v", args)
	}
}

func TestCastDateTimeGeneration(t *testing.T) {
	info := getTestInfo()
	c := expr.CastDate(expr.Field("CreatedAt"))
	resolved := c.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)

	expectedType := "TEXT"
	switch testdb.ENGINE {
	case "mysql", "mysql_local", "mariadb":
		expectedType = "DATE"
	case "postgres":
		expectedType = "DATE"
	}

	expectedSQL := fixSQL(info, "CAST(`test_model`.`created_at` AS "+expectedType+")")
	if !strings.EqualFold(strings.TrimSpace(sb.String()), strings.TrimSpace(expectedSQL)) {
		t.Errorf("Expected %s output, got: %s", expectedType, sb.String())
	}
}
