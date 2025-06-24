package expr

import (
	"fmt"
	"reflect"
)

func expressionFromInterface[T Expression](exprValue interface{}, asValue bool) []T {
	var exprs = make([]T, 0)
	switch v := exprValue.(type) {
	case Expression:
		exprs = append(exprs, v.(T))
	case []Expression:
		for _, expr := range v {
			exprs = append(exprs, expr.(T))
		}
	case []T:
		exprs = append(exprs, v...)
	case []any:
		for _, expr := range v {
			exprs = append(exprs, expressionFromInterface[T](expr, asValue)...)
		}
	case string:
		if asValue {
			exprs = append(exprs, Value(v).(T))
		} else {
			exprs = append(exprs, Field(v).(T))
		}
	default:
		var rTyp = reflect.TypeOf(exprValue)
		var rVal = reflect.ValueOf(exprValue)
		switch rTyp.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < rVal.Len(); i++ {
				var elem = rVal.Index(i).Interface()
				exprs = append(exprs, expressionFromInterface[T](elem, asValue)...)
			}
		default:
			exprs = append(exprs, Value(v).(T))
		}
	}

	return exprs
}

func Express(key interface{}, vals ...interface{}) []ClauseExpression {
	switch v := key.(type) {
	case string:
		if len(vals) == 0 {
			panic(fmt.Errorf("no values provided for key %q", v))
		}
		return []ClauseExpression{Q(v, vals...)}
	case Expression:
		var expr = &ExprGroup{children: make([]Expression, 0, len(vals)+1), op: OpAnd}
		expr.children = append(expr.children, v)
		for _, val := range vals {
			var v, ok = val.(Expression)
			if !ok {
				panic(fmt.Errorf("value %v is not an Expression", val))
			}
			expr.children = append(expr.children, v)
		}
		return []ClauseExpression{expr}
	case []Expression:
		var expr = &ExprGroup{children: make([]Expression, 0, len(vals)+1), op: OpAnd}
		expr.children = append(expr.children, v...)
		for _, val := range vals {
			var v, ok = val.(Expression)
			if !ok {
				panic(fmt.Errorf("value %v is not an Expression", val))
			}
			expr.children = append(expr.children, v)
		}
		return []ClauseExpression{expr}
	case []ClauseExpression:
		var expr = &ExprGroup{children: make([]Expression, 0, len(vals)+len(v)), op: OpAnd}
		for _, e := range v {
			expr.children = append(expr.children, e)
		}
		for _, val := range vals {
			var v, ok = val.(Expression)
			if !ok {
				panic(fmt.Errorf("value %v is not an Expression", val))
			}
			expr.children = append(expr.children, v)
		}
		return []ClauseExpression{expr}
	case map[string]interface{}:
		var expr = make([]ClauseExpression, 0, len(v))
		for k, val := range v {
			expr = append(expr, Q(k, val))
		}
		return expr
	default:
		panic(fmt.Errorf("unsupported type %T", key))
	}
}
