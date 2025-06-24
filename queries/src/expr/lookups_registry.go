package expr

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
)

func registerToMap[T _sharedLookupTransform](local map[reflect.Type]map[string]T, global map[string]T, lookup T) (map[reflect.Type]map[string]T, map[string]T) {
	if local == nil {
		local = make(map[reflect.Type]map[string]T)
	}

	if global == nil {
		global = make(map[string]T)
	}

	var drivers = lookup.Drivers()
	var name = lookup.Name()
	if len(drivers) == 0 {
		// register globally
		global[name] = lookup
		return local, global
	}

	for _, drv := range drivers {
		var t = reflect.TypeOf(drv)
		if _, ok := local[t]; !ok {
			local[t] = make(map[string]T)
		}
		local[t][name] = lookup
	}

	return local, global
}

func retrieveFromMap[T _sharedLookupTransform](local map[reflect.Type]map[string]T, global map[string]T, lookupName string, driver driver.Driver) (T, bool) {
	var t = reflect.TypeOf(driver)
	if local != nil {
		if localLookups, ok := local[t]; ok {
			if lookup, ok := localLookups[lookupName]; ok {
				return lookup, true
			}
		}
	}

	if global != nil {
		if lookup, ok := global[lookupName]; ok {
			return lookup, true
		}
	}

	var zero T
	return zero, false
}

type lookupRegistry struct {
	transformsLocal  map[reflect.Type]map[string]LookupTransform
	transformsGlobal map[string]LookupTransform
	lookupsLocal     map[reflect.Type]map[string]Lookup
	lookupsGlobal    map[string]Lookup
}

// lookups and transforms both adhere to this interface
type _sharedLookupTransform interface {
	// returns the drivers that support this transform
	// if empty, the transform is supported by all drivers
	Drivers() []driver.Driver

	// name of the transform
	Name() string
}

func (r *lookupRegistry) RegisterLookup(lookup Lookup) {
	if lookup == nil {
		panic("lookup cannot be nil")
	}

	var name = lookup.Name()
	if name == "" {
		panic("lookup name cannot be empty")
	}

	r.lookupsLocal, r.lookupsGlobal = registerToMap(
		r.lookupsLocal, r.lookupsGlobal, lookup,
	)
}

func (r *lookupRegistry) RegisterTransform(transform LookupTransform) {
	if transform == nil {
		panic("transform cannot be nil")
	}
	var name = transform.Name()
	if name == "" {
		panic("transform name cannot be empty")
	}
	r.transformsLocal, r.transformsGlobal = registerToMap(
		r.transformsLocal, r.transformsGlobal, transform,
	)
}

func (r *lookupRegistry) HasLookup(lookupName string, driver driver.Driver) bool {
	if lookupName == "" {
		return false
	}

	var _, ok = retrieveFromMap(r.lookupsLocal, r.lookupsGlobal, lookupName, driver)
	return ok
}

func (r *lookupRegistry) HasTransform(transformName string, driver driver.Driver) bool {
	if transformName == "" {
		return false
	}
	var _, ok = retrieveFromMap(r.transformsLocal, r.transformsGlobal, transformName, driver)
	return ok
}

func (r *lookupRegistry) Lookup(inf *ExpressionInfo, transforms []string, lookupName string, lhs any, args []any) (func(sb *strings.Builder) []any, error) {
	var lookup, ok = retrieveFromMap(r.lookupsLocal, r.lookupsGlobal, lookupName, inf.Driver)
	if !ok || lookup == nil {
		return nil, fmt.Errorf(
			"no lookup %q found for driver %T: %w",
			lookupName, inf.Driver, ErrLookupNotFound,
		)
	}

	var min, max = lookup.Arity()
	if len(args) < min || (max >= 0 && len(args) > max) {
		return nil, fmt.Errorf(
			"lookup %s requires between %d and %d arguments, got %d: %w",
			lookup.Name(), min, max, len(args), ErrLookupArgsInvalid,
		)
	}

	var normalizedArgs, err = lookup.NormalizeArgs(inf, args)
	if err != nil {
		return nil, fmt.Errorf(
			"error normalizing args for lookup %s: %w",
			lookup.Name(), err,
		)
	}

	var lhsExpr ResolvedExpression
	switch lhs := lhs.(type) {
	case string:
		lhsExpr = String(lhs).Resolve(inf)
	case Expression:
		lhsExpr = lhs.Resolve(inf)
	default:
		return nil, fmt.Errorf("unsupported type for lhs: %T", lhs)
	}

	//var (
	//	allowedTransformsMap = make(map[string]struct{})
	//	allowedLookupsMap    = make(map[string]struct{})
	//	transformsMap        = make(map[string]LookupTransform)
	//)

	for _, transformName := range transforms {
		var transform, ok = retrieveFromMap(r.transformsLocal, r.transformsGlobal, transformName, inf.Driver)
		if !ok || transform == nil {
			return nil, fmt.Errorf(
				"no transform %q found for driver %T: %w",
				transformName, inf.Driver, ErrTransformNotFound,
			)
		}

		//for _, transform := range transform.AllowedTransforms() {
		//	allowedTransformsMap[transform] = struct{}{}
		//}
		//
		//for _, lookup := range transform.AllowedLookups() {
		//	allowedLookupsMap[lookup] = struct{}{}
		//}

		// transformsMap[transformName] = transform
		lhsExpr, err = transform.Resolve(inf, lhsExpr)
		if err != nil {
			return nil, fmt.Errorf(
				"error resolving transform %q for lookup %q: %w",
				transformName, lookup.Name(), err,
			)
		}
	}

	var expr = lookup.Resolve(inf, lhsExpr, normalizedArgs)
	if expr == nil {
		return nil, fmt.Errorf("lookup %s returned nil expression", lookup.Name())
	}

	return expr, nil
}
