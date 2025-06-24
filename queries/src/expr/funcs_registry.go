package expr

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
)

type funcLookupsRegistry struct {
	m   map[string]func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error)
	d_m map[reflect.Type]map[string]func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error)
}

func (l *funcLookupsRegistry) lookupFunc(driver driver.Driver, lookup string) (func(inf *ExpressionInfo, value []Expression, funcParams []any) (sql string, args []any, err error), bool) {
	var m, ok = l.d_m[reflect.TypeOf(driver)]
	if !ok {
		m = l.m
	}

	fn, ok := m[lookup]
	if !ok {
		fn, ok = l.m[lookup]
	}
	if !ok {
		return nil, false
	}

	// Expressions need to be resolved before they can be used in the function
	// Wrap the function and look over the func params to resolve any Expressions
	// that are passed in.
	return func(inf *ExpressionInfo, value []Expression, funcParams []any) (sql string, args []any, err error) {
		var params = make([]any, 0, len(funcParams))
		for _, p := range funcParams {
			switch v := p.(type) {
			case Expression:
				params = append(params, v.Resolve(inf))
			default:
				params = append(params, v)
			}
		}

		return fn(inf.Driver, value, params)
	}, true
}

func (l *funcLookupsRegistry) Lookup(inf *ExpressionInfo, lookup string, value []Expression, funcParams []any) (sql string, args []any, err error) {
	var fn, ok = l.lookupFunc(inf.Driver, lookup)
	if !ok {
		return "", nil, fmt.Errorf(
			"function %q not found for driver %T: %w",
			lookup, inf.Driver, query_errors.ErrUnsupportedLookup)
	}

	sql, args, err = fn(inf, value, funcParams)
	if err != nil {
		return "", nil, fmt.Errorf(
			"error executing function %q: %w",
			lookup, err,
		)
	}

	return sql, args, nil
}

func (l *funcLookupsRegistry) register(lookup string, fn func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error), drivers ...driver.Driver) {
	if len(drivers) == 0 {
		l.m[lookup] = fn
		return
	}

	for _, drv := range drivers {
		var t = reflect.TypeOf(drv)
		if _, ok := l.d_m[t]; !ok {
			l.d_m[t] = make(map[string]func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error))
		}
		l.d_m[t][lookup] = fn
	}
}

var funcLookups = &funcLookupsRegistry{
	m:   make(map[string]func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error)),
	d_m: make(map[reflect.Type]map[string]func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error)),
}

func RegisterFunc(funcName string, fn func(d driver.Driver, value []Expression, funcParams []any) (sql string, args []any, err error), drivers ...driver.Driver) {
	funcLookups.register(funcName, fn, drivers...)
}
