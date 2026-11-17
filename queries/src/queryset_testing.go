//go:build test
// +build test

package queries

// Internals exposes the underlying QuerySetInternals for test inspection.
func (qs *QuerySet[T]) Internals() *QuerySetInternals {
	return qs.internals
}
