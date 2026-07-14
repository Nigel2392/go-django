package expr_test

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestFuncRegistrySQLGeneration1(t *testing.T) {
	expr.RegisterFunc("CUSTOM_SQL_FUNC_1", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "CUSTOM1()", nil, nil
	})
	info := getTestInfo()
	sql, _, err := expr.LookupFunc(info, "CUSTOM_SQL_FUNC_1", nil, nil)
	if err != nil {
		t.Fatalf("LookupFunc error: %v", err)
	}
	if !strings.Contains(sql, "CUSTOM1()") {
		t.Errorf("Expected CUSTOM1(), got %s", sql)
	}
}

// SQL Generation 2
func TestFuncRegistrySQLGeneration2(t *testing.T) {
	expr.RegisterFunc("CUSTOM_SQL_FUNC_2", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "CUSTOM2(?, ?)", []any{1, 2}, nil
	})
	info := getTestInfo()
	sql, args, err := expr.LookupFunc(info, "CUSTOM_SQL_FUNC_2", nil, nil)
	if err != nil {
		t.Fatalf("LookupFunc error: %v", err)
	}
	if !strings.Contains(sql, "CUSTOM2(?, ?)") {
		t.Errorf("Expected CUSTOM2(?, ?), got %s", sql)
	}
	if len(args) != 2 || args[0] != 1 || args[1] != 2 {
		t.Errorf("Unexpected args: %v", args)
	}
}

// Happy Path 1
func TestFuncRegistryLookupThroughFunc(t *testing.T) {
	expr.RegisterFunc("test_lookup_func", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "TEST_LOOKUP(?)", []any{99}, nil
	})

	info := getTestInfo()
	sql, args, err := expr.LookupFunc(info, "test_lookup_func", []expr.Expression{expr.Field("Age")}, nil)
	if err != nil {
		t.Fatalf("error while looking up func: %v", err)
	}

	if !strings.Contains(sql, "TEST_LOOKUP(?)") {
		t.Errorf("Expected TEST_LOOKUP, got %s", sql)
	}
	if len(args) != 1 || args[0] != 99 {
		t.Errorf("Unexpected args: %v", args)
	}

	// Test missing function
	_, _, err = expr.LookupFunc(info, "non_existent_func", []expr.Expression{expr.Field("Age")}, nil)
	if err == nil {
		t.Errorf("Expected error when looking up missing func")
	}
}

// Happy Path 2
func TestFuncRegistryHappyPath2(t *testing.T) {
	expr.RegisterFunc("CUSTOM_HAPPY_2", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "HAPPY2", nil, nil
	})
	info := getTestInfo()
	sql, _, err := expr.LookupFunc(info, "CUSTOM_HAPPY_2", nil, nil)
	if err != nil {
		t.Fatalf("LookupFunc error: %v", err)
	}
	if sql != "HAPPY2" {
		t.Errorf("Expected HAPPY2, got %s", sql)
	}
}

// Unhappy Path 1
func TestFuncRegistryUnhappyPath1(t *testing.T) {
	expr.RegisterFunc("CUSTOM_ERR", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "", nil, fmt.Errorf("my error")
	})
	info := getTestInfo()
	_, _, err := expr.LookupFunc(info, "CUSTOM_ERR", nil, nil)
	if err == nil {
		t.Errorf("Expected error when resolving func that returns error")
	}
}

// Unhappy Path 2
func TestFuncRegistryUnhappyPath2(t *testing.T) {
	expr.RegisterFunc("CUSTOM_PANIC", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		panic("intentional panic inside func")
	})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving panic func")
		}
	}()
	info := getTestInfo()
	_, _, _ = expr.LookupFunc(info, "CUSTOM_PANIC", nil, nil)
}
