package expr

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/errs"
)

func init() {
	RegisterLookup(logicalLookup(LOOKUP_EXACT, EQ, normalizeDefinerArg))
	RegisterLookup(logicalLookup(LOOKUP_NOT, NE, normalizeDefinerArg))

	RegisterLookup(logicalLookup(LOOKUP_GT, GT, nil))
	RegisterLookup(logicalLookup(LOOKUP_LT, LT, nil))
	RegisterLookup(logicalLookup(LOOKUP_GTE, GTE, nil))
	RegisterLookup(logicalLookup(LOOKUP_LTE, LTE, nil))

	//	RegisterLookup(logicalLookup(LOOKUP_ADD, ADD, nil))
	//	RegisterLookup(logicalLookup(LOOKUP_SUB, SUB, nil))
	//	RegisterLookup(logicalLookup(LOOKUP_MUL, MUL, nil))
	//	RegisterLookup(logicalLookup(LOOKUP_DIV, DIV, nil))
	//	RegisterLookup(logicalLookup(LOOKUP_MOD, MOD, nil))

	RegisterLookup(logicalLookup(LOOKUP_BITAND, BITAND, nil))
	RegisterLookup(logicalLookup(LOOKUP_BITOR, BITOR, nil))
	RegisterLookup(logicalLookup(LOOKUP_BITXOR, BITXOR, nil))
	RegisterLookup(logicalLookup(LOOKUP_BITLSH, BITLSH, nil))
	RegisterLookup(logicalLookup(LOOKUP_BITRSH, BITRSH, nil))
	RegisterLookup(logicalLookup(LOOKUP_BITNOT, BITNOT, nil))

	RegisterLookup(patternLookup(LOOKUP_IEXACT, "%s"))
	RegisterLookup(patternLookup(LOOKUP_CONTAINS, "%%%s%%"))
	RegisterLookup(patternLookup(LOOKUP_ICONTANS, "%%%s%%"))
	RegisterLookup(patternLookup(LOOKUP_STARTSWITH, "%s%%"))
	RegisterLookup(patternLookup(LOOKUP_ISTARTSWITH, "%s%%"))
	RegisterLookup(patternLookup(LOOKUP_IENDSWITH, "%%%s"))
	RegisterLookup(patternLookup(LOOKUP_ENDSWITH, "%%%s"))

	RegisterLookup(&InLookup{
		BaseLookup: BaseLookup{
			Identifier: LOOKUP_IN,
		},
	})

	RegisterLookup(&IsNullLookup{BaseLookup: BaseLookup{
		Identifier: LOOKUP_ISNULL,
	}})

	RegisterTransforms(&BaseTransform{
		Identifier: "lower",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return LOWER(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "upper",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return UPPER(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "length",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return LENGTH(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "count",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return COUNT(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "avg",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return AVG(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "sum",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return SUM(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "min",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return MIN(lhsResolved).Resolve(inf), nil
		},
	})
	RegisterTransforms(&BaseTransform{
		Identifier: "max",
		Transform: func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
			return MAX(lhsResolved).Resolve(inf), nil
		},
	})
}

const (
	ErrLookupNotFound    errs.Error = "lookup not found"
	ErrTransformNotFound errs.Error = "transform not found"
	ErrLookupArgsInvalid errs.Error = "lookup arguments invalid"

	LOOKUP_EXACT = "exact"
	LOOKUP_NOT   = "not"
	LOOKUP_GT    = "gt"
	LOOKUP_LT    = "lt"
	LOOKUP_GTE   = "gte"
	LOOKUP_LTE   = "lte"

	//	LOOKUP_ADD         = "add"
	//	LOOKUP_SUB         = "sub"
	//	LOOKUP_MUL         = "mul"
	//	LOOKUP_DIV         = "div"
	//	LOOKUP_MOD         = "mod"

	LOOKUP_BITAND = "bitand"
	LOOKUP_BITOR  = "bitor"
	LOOKUP_BITXOR = "bitxor"
	LOOKUP_BITLSH = "bitlsh"
	LOOKUP_BITRSH = "bitrsh"
	LOOKUP_BITNOT = "bitnot"

	LOOKUP_IEXACT      = "iexact"
	LOOKUP_CONTAINS    = "contains"
	LOOKUP_ICONTANS    = "icontains"
	LOOKUP_STARTSWITH  = "startswith"
	LOOKUP_ISTARTSWITH = "istartswith"
	LOOKUP_IENDSWITH   = "iendswith"
	LOOKUP_ENDSWITH    = "endswith"
	LOOKUP_IN          = "in"
	LOOKUP_ISNULL      = "isnull"
	LOOKUP_RANGE       = "range"

	DEFAULT_LOOKUP = LOOKUP_EXACT

	// NYI
	// LOOKUP_REGEX  = "regex"
	// LOOKUP_IREGEX = "iregex"
)

var lookupsRegistry = &lookupRegistry{
	lookupsLocal:  make(map[reflect.Type]map[string]Lookup),
	lookupsGlobal: make(map[string]Lookup),
}

func RegisterLookup(Lookup Lookup) {
	if Lookup == nil {
		panic("lookup cannot be nil")
	}

	lookupsRegistry.RegisterLookup(Lookup)
}

func RegisterTransforms(transforms ...LookupTransform) {
	if len(transforms) == 0 {
		panic("at least one transform must be provided")
	}

	for _, transform := range transforms {
		if transform == nil {
			panic("transform cannot be nil")
		}
		lookupsRegistry.RegisterTransform(transform)
	}
}

// GetLookup retrieves a lookup function based on the provided expression info, lookup name, inner expression, and arguments.
// It returns a function that can be used to build a SQL string with the lookup applied.
// The LHS will need to be either an Expression (RESOLVED ALREADY!) or a sql `table`.`column` pair.
func GetLookup(inf *ExpressionInfo, lookupName string, lhs any, args []any) (func(sb *strings.Builder) []any, error) {
	if inf == nil {
		return nil, fmt.Errorf("expression info cannot be nil")
	}

	var (
		transforms  []string
		lookupParts = strings.Split(lookupName, "__")
	)

	if len(lookupParts) > 1 {
		transforms = lookupParts[:len(lookupParts)-1]
		lookupName = lookupParts[len(lookupParts)-1]
	}

	if lookupName != "" && !lookupsRegistry.HasLookup(lookupName, inf.Driver) {
		transforms = append(transforms, lookupName)
		lookupName = DEFAULT_LOOKUP
	}

	return lookupsRegistry.Lookup(inf, transforms, lookupName, lhs, args)
}
