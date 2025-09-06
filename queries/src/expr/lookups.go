package expr

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
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

var (
	ErrLookupNotFound    errors.Error = errors.New("LookupNotFound", "lookup not found")
	ErrTransformNotFound errors.Error = errors.New("TransformNotFound", "transform not found")
	ErrLookupArgsInvalid errors.Error = errors.New("LookupArgsInvalid", "lookup arguments invalid")
)

type LookupFilter = string

const (
	LOOKUP_EXACT LookupFilter = "exact"
	LOOKUP_NOT   LookupFilter = "not"
	LOOKUP_GT    LookupFilter = "gt"
	LOOKUP_LT    LookupFilter = "lt"
	LOOKUP_GTE   LookupFilter = "gte"
	LOOKUP_LTE   LookupFilter = "lte"

	//	LOOKUP_ADD         = "add"
	//	LOOKUP_SUB         = "sub"
	//	LOOKUP_MUL         = "mul"
	//	LOOKUP_DIV         = "div"
	//	LOOKUP_MOD         = "mod"

	LOOKUP_BITAND LookupFilter = "bitand"
	LOOKUP_BITOR  LookupFilter = "bitor"
	LOOKUP_BITXOR LookupFilter = "bitxor"
	LOOKUP_BITLSH LookupFilter = "bitlsh"
	LOOKUP_BITRSH LookupFilter = "bitrsh"
	LOOKUP_BITNOT LookupFilter = "bitnot"

	LOOKUP_IEXACT      LookupFilter = "iexact"
	LOOKUP_CONTAINS    LookupFilter = "contains"
	LOOKUP_ICONTANS    LookupFilter = "icontains"
	LOOKUP_STARTSWITH  LookupFilter = "startswith"
	LOOKUP_ISTARTSWITH LookupFilter = "istartswith"
	LOOKUP_IENDSWITH   LookupFilter = "iendswith"
	LOOKUP_ENDSWITH    LookupFilter = "endswith"
	LOOKUP_IN          LookupFilter = "in"
	LOOKUP_ISNULL      LookupFilter = "isnull"
	LOOKUP_RANGE       LookupFilter = "range"

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
