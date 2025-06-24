package expr

import (
	"database/sql/driver"
	"fmt"
)

var _ LookupTransform = (*BaseTransform)(nil)

type BaseTransform struct {
	AllowedDrivers []driver.Driver
	//	Transforms     []string
	//	Lookups        []string
	Identifier string
	Transform  func(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error)
}

func (t *BaseTransform) Drivers() []driver.Driver {
	return t.AllowedDrivers
}

func (t *BaseTransform) Name() string {
	return t.Identifier
}

///	func (t *BaseTransform) AllowedTransforms() []string {
///		return t.Transforms
///	}
///
///	func (t *BaseTransform) AllowedLookups() []string {
///		return t.Lookups
///	}

func (t *BaseTransform) Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error) {
	if t.Transform == nil {
		return nil, fmt.Errorf(
			"transform %q does not have a resolve function defined, cannot be used",
			t.Identifier,
		)
	}
	return t.Transform(inf, lhsResolved)
}
