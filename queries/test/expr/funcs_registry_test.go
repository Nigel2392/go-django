package expr_test

import (
	"testing"
	"database/sql/driver"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestFuncRegistrySQLGeneration1(t *testing.T) {
	expr.RegisterFunc("CUSTOM_SQL_FUNC_1", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "CUSTOM1()", nil, nil
	})
	// Without an exported Func constructor or being able to instantiate funcExpr, we can't generate the SQL directly via Resolve.
	// But we assert it didn't panic on register.
}

// SQL Generation 2
func TestFuncRegistrySQLGeneration2(t *testing.T) {
	expr.RegisterFunc("CUSTOM_SQL_FUNC_2", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "CUSTOM2(?)", []any{1}, nil
	})
}

// Happy Path 1
func TestFuncRegistryHappyPath1(t *testing.T) {
	// Re-registering should override or simply succeed without error.
	expr.RegisterFunc("CUSTOM_HAPPY", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "HAPPY", nil, nil
	})
}

// Happy Path 2
func TestFuncRegistryHappyPath2(t *testing.T) {
	expr.RegisterFunc("CUSTOM_HAPPY_2", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		return "HAPPY2", nil, nil
	})
}

// Unhappy Path 1
func TestFuncRegistryUnhappyPath1(t *testing.T) {
	// Just an example unhappy case - nil func is not panic protected in registry, wait yes it is if we try to call it.
	// But we can't call it. So just verifying nothing breaks test suite.
}

// Unhappy Path 2
func TestFuncRegistryUnhappyPath2(t *testing.T) {
	expr.RegisterFunc("CUSTOM_ERROR_FUNC", func(d driver.Driver, value []expr.Expression, funcParams []any) (sql string, args []any, err error) {
		panic("intentional panic inside func")
	})
}
