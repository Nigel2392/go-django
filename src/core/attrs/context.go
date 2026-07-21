package attrs

import (
	"context"
	"fmt"
	"maps"
	"unsafe"

	"github.com/Nigel2392/go-django/internal/bitch"
)

type ContextFlag bitch.Flag

const (
	CtxFlagNone        ContextFlag = 0
	CtxFlagRegistering ContextFlag = 1 << (iota - 1)

	// CtxFlagDeferSignals should ensure that fields
	// inside a model will defer their signals (such as change)
	// until [SigFlagSignals] is sent through [ContextSignal.Flag]
	CtxFlagDeferSignals

	// CtxFlagDeferValidation should ensure that fields
	// inside a model will defer their validation
	// until [SigFlagValidate] is sent through [ContextSignal.Flag].
	//
	// This does (and should) not mean fields will immediately
	// validate when the signal is sent.
	CtxFlagDeferValidation
)

type ContextSignalFlag bitch.Flag

const (
	SigFlagNone    ContextSignalFlag = 0
	SigFlagSignals ContextSignalFlag = 1 << (iota - 1)
	SigFlagValidate
)

var signalMapping = map[ContextFlag]ContextSignalFlag{
	CtxFlagDeferSignals:    SigFlagValidate,
	CtxFlagDeferValidation: SigFlagValidate,
}

type attrsContextKey struct{}

// ContextSignal is used to provide a way to easily defer validation
// and other actions when required.
type ContextSignal struct {
	Flag ContextSignalFlag
}

// Context is used throughout the attrs package to make performance improvements
// and provide other QOL improvements for developers.
type Context struct {
	Flags ContextFlag

	// InitSize can be provided to provide
	// a default length for the [fieldsMap]
	// and certain listeners in the [signal] map.
	// this is useful when the amount of objects you
	// are defining are already known.
	InitSize int

	// fieldsMap provides a way to cache any objects' fielddefs
	// to which this context was provided.
	// the reference of fieldsMap is not taken across
	// the context boundaries.
	fieldsMap map[unsafe.Pointer]Definitions

	// signal allows for deferring validaton and other stuff
	// until the time is ready.
	//
	// This is a reference type, it will be
	// the same reference after [Context.Clone]
	// to allow for pushing values or other validation
	// all the way up to where this context was required.
	signal map[ContextSignalFlag][]func() error
}

// Reset provides a way to fully reset this context.
// This means clearing all flags, sending a RESET signal up
// through the stack, and clearing any caches that may have been filled.
func (c *Context) Reset() {
	c.Flags = CtxFlagNone
	c.InitSize = 0
	clear(c.fieldsMap)
}

// Clone provides a way to override flags while still
// being able to push or defer other validation until a later point.
func (c *Context) Clone() *Context {
	return &Context{
		Flags:     c.Flags,
		fieldsMap: maps.Clone(c.fieldsMap),
		signal:    c.signal,
	}
}

func (c *Context) Send(sig ContextSignalFlag) error {
	// nobody has actually connected to any signal.
	if c.signal == nil {
		return nil
	}

	funcs, ok := c.signal[sig]
	if !ok {
		return nil
	}

	for _, fn := range funcs {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Context) Listen(sig ContextFlag, fn func() error) {
	for src := range signalMapping {
		if bitch.Isnt(src, sig) {
			continue
		}

		out := signalMapping[src]

		if c.signal == nil {
			c.signal = make(map[ContextSignalFlag][]func() error)
		}

		l, ok := c.signal[out]
		if !ok {
			l = make([]func() error, c.InitSize)
		}

		c.signal[out] = append(l, fn)
	}
}

// This is basically required.
//
// If you pass a non- attribute context to a struct that requires it,
// you will panic.
func IsAttributeContext(ctx context.Context) bool {
	var val = ctx.Value(attrsContextKey{})
	if val == nil {
		return false
	}
	switch val.(type) {
	case ContextFlag, *Context:
		return true
	}
	panic(fmt.Sprintf("this shouldn't happen: unknown context type used: %T", val))
}

// ContextWithFlags sets the flags on the [AttsContext] if present,
// or dumps them in the context otherwise.
func ContextWithFlags(ctx context.Context, flag ContextFlag, set bool) context.Context {
	var val = ctx.Value(attrsContextKey{})
	if val == nil {
		if set {
			return context.WithValue(ctx, attrsContextKey{}, flag)
		}
		return context.WithValue(ctx, attrsContextKey{}, CtxFlagNone)
	}

	switch v := val.(type) {
	case ContextFlag:
		return context.WithValue(
			ctx, attrsContextKey{},
			bitch.Set(v, flag, set),
		)
	case *Context:
		// clone the context to preserve whatever
		// flags are already set / not set.
		// signals will not be cloned, their reference remains.
		newCtx := v.Clone()
		newCtx.Flags = bitch.Set(newCtx.Flags, flag, set)
		return context.WithValue(ctx, attrsContextKey{}, newCtx)
	}

	panic(fmt.Sprintf("this shouldn't happen: unknown context type used: %T", val))
}

func ContextHasFlag(ctx context.Context, flag ContextFlag) bool {
	var val = ctx.Value(attrsContextKey{})
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case ContextFlag:
		return bitch.Is(v, flag)
	case *Context:
		return bitch.Is(v.Flags, flag)
	}

	panic(fmt.Sprintf("this shouldn't happen: unknown context type used: %T", val))
}

func AttributeContext(ctx context.Context) (_ context.Context, c *Context) {
	var val = ctx.Value(attrsContextKey{})
	if val == nil {
		c = &Context{}
		return context.WithValue(ctx, attrsContextKey{}, c), c
	}

	switch v := val.(type) {
	case ContextFlag:
		c = &Context{
			Flags: v,
		}
	case *Context:
		// clone the context to preserve whatever
		// flags are already set / not set on the parent context.
		// the context is for all intents and purposes, immutable.
		// signals will not be cloned, their reference remains.
		// immutable only means in the direction of the parent.
		c = v.Clone()
	}

	return context.WithValue(ctx, attrsContextKey{}, c), c
}

type eface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

func Define(ctx context.Context, model Definer) Definitions {

	var val = ctx.Value(attrsContextKey{})
	if val == nil {
		// this prevents panics.
		// the only right way to call FieldDefs() on a model is though [Definition].
		ctx = context.WithValue(ctx, attrsContextKey{}, CtxFlagNone)
		return model.FieldDefs(ctx)
	}

	c, ok := val.(*Context)
	if !ok {
		// not *Context but still valid
		return model.FieldDefs(ctx)
	}

	if c.fieldsMap == nil {
		c.fieldsMap = make(map[unsafe.Pointer]Definitions)
	}

	ptr := (*eface)(unsafe.Pointer(&model)).data
	defs, ok := c.fieldsMap[ptr]
	if !ok {
		defs = model.FieldDefs(ctx)
		c.fieldsMap[ptr] = defs
	}

	return defs
}
