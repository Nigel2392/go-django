package embedder

import (
	"context"
	"reflect"
)

type Registry interface {
	Bind(ctx context.Context, obj any) error
	Count(obj any) int
	Register(obj any)
	Walk(obj any, walkFn func(depth int, field reflect.StructField) (stop bool)) bool
}

var (
	Init     Registry = NewRegistry("SetupEmbedded")
	PreSave  Registry = NewRegistry("PreSaveEmbedded")
	PostSave Registry = NewRegistry("PostSaveEmbedded")
)

//	// Bind ensures any embedded types can be properly initialized.
//	//
//	// when the obj param does not embed any such types, this function will be as fast as a single map read.
//	//
//	// !!! WARNING DANGER !!!
//	//
//	// Any objects that utilize the context cannot utilize the [context.Context]
//	// variable any longer than the lifetime of the bind function.
//	func Bind(ctx context.Context, obj any) {
//		initRegistry.Bind(ctx, obj)
//	}
//
//	// Register an object that might have an embedded type which requires setup.
//	//
//	// If no supported embedded types are found, the type is left out of the registry.
//	func Register(obj any) {
//		initRegistry.Register(obj)
//	}
//
//	// Count returns the amount of embedders that will be initialized.
//	//
//	// The obj must be registered before the count of initializers can be retrieved.
//	//
//	// When the obj param does not embed any supported embedded types,
//	// this function will return 0, and be as fast as a single map read.
//	func Count(obj any) int {
//		return initRegistry.Count(obj)
//	}
//
//	// Walk walks the initializer tree, calling a function with the depth and structfield of each initializer.
//	//
//	// If no initializers are defined for param `obj`, this function will do nothing.
//	func Walk(obj any, walkFn func(depth int, field reflect.StructField) (stop bool)) bool {
//		return initRegistry.Walk(obj, walkFn)
//	}
