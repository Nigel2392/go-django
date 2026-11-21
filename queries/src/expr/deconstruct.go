package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/deconstruct"
)

func init() {
	deconstruct.Constructors.RegisterDestructibleType[Expression]()
	deconstruct.Constructors.RegisterDestructibleType[NamedExpression]()
	deconstruct.Constructors.RegisterDestructibleType[AnnotationExpression]()
	deconstruct.Constructors.RegisterDestructibleType[LogicalExpression]()
	deconstruct.Constructors.RegisterDestructibleType[ClauseExpression]()
	deconstruct.Constructors.RegisterDestructibleType[LogicalNamedExpressionFunc]()
	deconstruct.Constructors.RegisterDestructibleType[ResolvedExpression]()
	deconstruct.Constructors.RegisterDestructibleType[WhenExpression]()
	deconstruct.Constructors.RegisterDestructibleType[*ExpressionStatement]()
	deconstruct.Constructors.RegisterDestructibleType[LogicalOp]()
	deconstruct.Constructors.RegisterDestructibleType[String]()
	deconstruct.Constructors.RegisterDestructibleType[*ExprNode]()
	deconstruct.Constructors.RegisterDestructibleType[*ExprGroup]()
	deconstruct.Constructors.RegisterDestructibleType[*CaseExpression]()
	deconstruct.Constructors.RegisterDestructibleType[*RawNamedExpression]()
	deconstruct.Constructors.RegisterDestructibleType[*Function]()
}

func unmarshalAnyArg(raw jsontext.Value, opts ...json.Options) (any, error) {
	switch raw.Kind() {
	case '0':
		s := strings.TrimSpace(string(raw))
		if !strings.ContainsAny(s, ".eE") {
			if i64, err := strconv.ParseInt(s, 10, 64); err == nil {
				if strconv.IntSize == 64 || (i64 >= math.MinInt && i64 <= math.MaxInt) {
					return int(i64), nil
				}
				return i64, nil
			}
		}
		var f float64
		if err := json.Unmarshal(raw, &f, opts...); err != nil {
			return nil, err
		}
		return f, nil
	case '[':
		var rawSlice []jsontext.Value
		if err := json.Unmarshal(raw, &rawSlice, opts...); err != nil {
			return nil, err
		}
		out := make([]any, len(rawSlice))
		for i, item := range rawSlice {
			v, err := unmarshalAnyArg(item, opts...)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case '{':
		var rawMap map[string]jsontext.Value
		if err := json.Unmarshal(raw, &rawMap, opts...); err != nil {
			return nil, err
		}
		if len(rawMap) == 3 {
			if rawPath, hasPath := rawMap["path"]; hasPath {
				if _, hasArgs := rawMap["args"]; hasArgs {
					if _, hasKwargs := rawMap["kwargs"]; hasKwargs {
						var path string
						if err := json.Unmarshal(rawPath, &path); err == nil && path != "" {
							var d deconstruct.Deconstructible
							if err := json.Unmarshal(raw, &d, opts...); err != nil {
								return nil, err
							}
							return d, nil
						}
					}
				}
			}
		}
		out := make(map[string]any, len(rawMap))
		for k, item := range rawMap {
			v, err := unmarshalAnyArg(item, opts...)
			if err != nil {
				return nil, err
			}
			out[k] = v
		}
		return out, nil
	default:
		var v any
		if err := json.Unmarshal(raw, &v, opts...); err != nil {
			return nil, err
		}
		return v, nil
	}
}

func unmarshalKwarg[T any](kwargs deconstruct.Kwargs, key string, opts ...json.Options) (T, bool, error) {
	rawVal, ok := kwargs[key]
	if !ok || rawVal == nil {
		var zero T
		return zero, false, nil
	}
	if typed, ok := rawVal.(T); ok {
		return typed, true, nil
	}
	rawBytes, err := json.Marshal(rawVal, opts...)
	if err != nil {
		var zero T
		return zero, true, err
	}
	var out T
	if err := json.Unmarshal(rawBytes, &out, opts...); err != nil {
		var zero T
		return zero, true, err
	}
	return out, true, nil
}
