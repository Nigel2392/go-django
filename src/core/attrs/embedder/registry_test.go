package embedder

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

type ImplementedInterface interface {
	ImplementedThis() ImplementedInterface
}

var callCount int

type testNoEmbedded struct {
	a, b, c int
}

type testSetupEmbedded1 struct {
	embedded1[testSetupEmbedded1]
}

type testSetupEmbedded2 struct {
	embedded2[*testSetupEmbedded2]
}

type testSetupEmbedded3 struct {
	embedded3
}

type testSetupEmbedded4 struct {
	embedded4
}

type testSetupEmbeddedDeep1 struct {
	setup *testSetupEmbedded1
}

type testSetupEmbeddedDeep2 struct {
	setup testSetupEmbedded3
}

type testSetupEmbeddedDeep3 struct {
	setup *testSetupEmbeddedDeep2
}

type testSetupEmbeddedDeep4 struct {
	setup *testSetupEmbedded4
}

func (t *testSetupEmbeddedDeep2) ImplementedThis() ImplementedInterface {
	return t
}

func (t *testSetupEmbeddedDeep3) ImplementedThis() ImplementedInterface {
	return t
}

type testSetupEmbeddedMultiple struct {
	embedded1[testSetupEmbeddedMultiple]
	embedded2[*testSetupEmbeddedMultiple]
	embedded3
	a *embedded1[testSetupEmbeddedMultiple]
	b *embedded2[*testSetupEmbeddedMultiple]
	c *embedded3
	//	embedded4
	//	d embedded4
	//	e embedded4
	//	f embedded4
	g *testSetupEmbeddedDeep1
	h *testSetupEmbeddedDeep4
}

type testSetupEmbeddedPtr1 struct {
	*embedded1[testSetupEmbeddedPtr1]
}

type testSetupEmbeddedPtr2 struct {
	*embedded2[*testSetupEmbeddedPtr2]
}

type testSetupEmbeddedPtr3 struct {
	*embedded3
}

type testSetupEmbeddedField1 struct {
	field embedded1[testSetupEmbeddedField1]
}

type testSetupEmbeddedField2 struct {
	field embedded2[*testSetupEmbeddedField2]
}

type testSetupEmbeddedField3 struct {
	field embedded3
}

type testSetupEmbeddedFieldPtr1 struct {
	field *embedded1[testSetupEmbeddedFieldPtr1]
}

type testSetupEmbeddedFieldPtr2 struct {
	field *embedded2[*testSetupEmbeddedFieldPtr2]
}

type testSetupEmbeddedFieldPtr3 struct {
	field *embedded3
}

func (t *testSetupEmbedded4) ImplementedThis() ImplementedInterface         { return nil }
func (t *testSetupEmbeddedMultiple) ImplementedThis() ImplementedInterface  { return nil }
func (t *testSetupEmbedded3) ImplementedThis() ImplementedInterface         { return nil }
func (t *testSetupEmbeddedPtr3) ImplementedThis() ImplementedInterface      { return nil }
func (t *testSetupEmbeddedField3) ImplementedThis() ImplementedInterface    { return nil }
func (t *testSetupEmbeddedFieldPtr3) ImplementedThis() ImplementedInterface { return nil }

type embedded1[T any] struct {
	backRef *T
}

func (e *embedded1[T]) SetupEmbedded(embedder any) {
	e.backRef = embedder.(*T)
	callCount++
}

type embedded2[T any] struct {
	backRef T
}

func (e *embedded2[T]) SetupEmbedded(embedder T) {
	e.backRef = embedder
	callCount++
}

type embedded3 struct {
	backRef ImplementedInterface
}

func (e *embedded3) SetupEmbedded(embedder ImplementedInterface) {
	e.backRef = embedder
	callCount++
}

type embedded4 struct {
	backRef ImplementedInterface
}

func (e *embedded4) SetupEmbedded(ctx context.Context, embedder reflect.Value, structField reflect.StructField) {
	r := RootContext(ctx).Root
	// fmt.Printf("%T %v => %T %v\n", r, r, embedder.Interface(), embedder.Interface())
	_ = r
	e.backRef = embedder.Interface().(ImplementedInterface)
	callCount++
}

func TestBind(t *testing.T) {
	var embedders = []any{
		&testSetupEmbedded1{},
		&testSetupEmbedded2{},
		&testSetupEmbedded3{},
		&testSetupEmbedded4{},
		&testSetupEmbeddedDeep1{},
		&testSetupEmbeddedDeep2{},
		&testSetupEmbeddedDeep3{},
		&testSetupEmbeddedDeep4{},
		&testSetupEmbeddedMultiple{},
		&testSetupEmbeddedPtr1{},
		&testSetupEmbeddedPtr2{},
		&testSetupEmbeddedPtr3{},
		&testSetupEmbeddedField1{},
		&testSetupEmbeddedField2{},
		&testSetupEmbeddedField3{},
		&testSetupEmbeddedFieldPtr1{},
		&testSetupEmbeddedFieldPtr2{},
		&testSetupEmbeddedFieldPtr3{},
	}

	var key = func(obj any) uintptr {
		return (*eface)(unsafe.Pointer(&obj)).typ
	}

	var embedderCount int
	for _, obj := range embedders {
		Init.Register(obj)
		count := Init.Count(obj)

		if count == 0 {
			t.Fatalf("Object %T has no embedders: %v", obj, Init.(*modelRegistry).typeReg[key(obj)])
		}

		embedderCount += count
	}

	if embedderCount < len(embedders) {
		t.Fatalf("expected at least %d embedders, got %d", len(embedders), embedderCount)
	}

	// reset the registry
	Init = &modelRegistry{
		method:  "SetupEmbedded",
		typeReg: make(map[uintptr][]*initializer, len(Init.(*modelRegistry).typeReg)),
	}

	t.Run("Init.Register", func(t *testing.T) {
		for _, embedder := range embedders {

			i, ok := Init.(*modelRegistry).typeReg[key(embedder)]
			if ok {
				t.Fatalf("retrieved %d initialisers", len(i))
			}

			Init.Register(embedder)

			i, ok = Init.(*modelRegistry).typeReg[key(embedder)]
			if !ok {
				t.Fatalf("could not retrieve initialisers for %T", embedder)
			}

			if len(i) != Init.Count(embedder) {
				t.Fatalf("expected 1 initialiser per type, got %d (%T)", len(i), embedder)
			}
		}
	})

	t.Run("Init.Walk", func(t *testing.T) {
		for _, embedder := range embedders {
			t.Logf("walking embedder tree for %T:", embedder)
			Init.Walk(embedder, func(depth int, field reflect.StructField) (stop bool) {
				t.Logf("%sStructField %s (%v)", strings.Repeat("  ", depth), field.Name, field.Index)
				return false
			})
		}
	})

	t.Run("Init.Bind", func(t *testing.T) {
		cC := callCount

		for _, embedder := range embedders {
			currCallCnt := callCount

			Init.Bind(t.Context(), embedder)

			count := Init.Count(embedder)

			if count == 0 {
				t.Fatalf("expected at least 1 embedded object to be initialized")
			}

			if currCallCnt+Init.Count(embedder) != callCount {
				t.Fatalf("expected bind func to be called %d times, got %d", Init.Count(embedder), callCount-currCallCnt)
			}
		}

		if cC+embedderCount != callCount {
			t.Fatalf("expected bind func to be called %d times, got %d", embedderCount, callCount-cC)
		}
	})

	t.Run("ReflectBind", func(t *testing.T) {
		cC := callCount

		for _, embedder := range embedders {
			currCallCnt := callCount

			Init.Bind(t.Context(), reflect.ValueOf(embedder))

			if currCallCnt+Init.Count(embedder) != callCount {
				t.Fatalf("expected bind func to be called %d times, got %d", Init.Count(embedder), callCount-currCallCnt)
			}
		}

		if cC+embedderCount != callCount {
			t.Fatalf("expected bind func to be called %d times, got %d", embedderCount, callCount-cC)
		}
	})

}

func BenchmarkBind(b *testing.B) {
	var embedders = []any{
		&testNoEmbedded{1, 2, 3},
		&testSetupEmbedded1{},
		&testSetupEmbedded2{},
		&testSetupEmbedded3{},
		&testSetupEmbedded4{},
		&testSetupEmbeddedDeep1{},
		&testSetupEmbeddedDeep2{},
		&testSetupEmbeddedDeep3{},
		&testSetupEmbeddedDeep4{},
		&testSetupEmbeddedMultiple{},
		&testSetupEmbeddedPtr1{},
		&testSetupEmbeddedPtr2{},
		&testSetupEmbeddedPtr3{},
		&testSetupEmbeddedField1{},
		&testSetupEmbeddedField2{},
		&testSetupEmbeddedField3{},
		&testSetupEmbeddedFieldPtr1{},
		&testSetupEmbeddedFieldPtr2{},
		&testSetupEmbeddedFieldPtr3{},
	}

	for _, embedder := range embedders {
		Init.Register(embedder)
	}

	for _, embedder := range embedders {
		b.Run(fmt.Sprintf("Init.Bind/%T", embedder), func(b *testing.B) {
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Sprintf("%T: %v", embedder, r))
				}
			}()
			for b.Loop() {
				Init.Bind(b.Context(), embedder)
			}
		})
	}

	for _, embedder := range embedders {
		b.Run(fmt.Sprintf("Init.Count/%T", embedder), func(b *testing.B) {
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Sprintf("%T: %v", embedder, r))
				}
			}()

			for b.Loop() {
				_ = Init.Count(embedder)
			}
		})
	}

	for _, embedder := range embedders {
		b.Run(fmt.Sprintf("Init.Bind/Reflect/%T", embedder), func(b *testing.B) {
			var embed any = reflect.ValueOf(embedder)
			for b.Loop() {
				Init.Bind(b.Context(), embed)
			}
		})
	}
}

var errTestSetup = errors.New("setup failed")

type errEmbedded1[T any] struct {
	backRef   *T
	shouldErr bool
	calls     int
}

func (e *errEmbedded1[T]) InitEmbedded(embedder any) error {
	e.calls++
	if e.shouldErr {
		return errTestSetup
	}
	e.backRef = embedder.(*T)
	return nil
}

type errEmbedded2[T any] struct {
	backRef   T
	shouldErr bool
	calls     int
}

func (e *errEmbedded2[T]) InitEmbedded(ctx context.Context, embedder T) error {
	e.calls++
	if e.shouldErr {
		return errTestSetup
	}
	e.backRef = embedder
	return nil
}

type errEmbedded3 struct {
	backRef   ImplementedInterface
	shouldErr bool
	calls     int
}

func (e *errEmbedded3) InitEmbedded(embedder ImplementedInterface, sf reflect.StructField) error {
	e.calls++
	if e.shouldErr {
		return errTestSetup
	}
	e.backRef = embedder
	return nil
}

type errEmbedded4 struct {
	backRef   ImplementedInterface
	fieldName string
	shouldErr bool
	calls     int
}

func (e *errEmbedded4) InitEmbedded(ctx context.Context, embedder reflect.Value, sf reflect.StructField) error {
	e.calls++
	if e.shouldErr {
		return errTestSetup
	}
	e.fieldName = sf.Name
	e.backRef = embedder.Interface().(ImplementedInterface)
	return nil
}

type testErrEmbedded1 struct {
	errEmbedded1[testErrEmbedded1]
}

type testErrEmbedded2 struct {
	errEmbedded2[*testErrEmbedded2]
}

type testErrEmbedded3 struct {
	errEmbedded3
}

func (t *testErrEmbedded3) ImplementedThis() ImplementedInterface { return t }

type testErrEmbedded4 struct {
	errEmbedded4
}

func (t *testErrEmbedded4) ImplementedThis() ImplementedInterface { return t }

type testErrEmbeddedPtr struct {
	*errEmbedded1[testErrEmbeddedPtr]
}

type testErrEmbeddedDeep1 struct {
	setup *testErrEmbedded1
}

type testErrEmbeddedDeep2 struct {
	outer *testErrEmbeddedDeep1
}

type testErrEmbeddedMultiple struct {
	first  errEmbedded1[testErrEmbeddedMultiple]
	second errEmbedded2[*testErrEmbeddedMultiple]
	third  errEmbedded3
}

func (t *testErrEmbeddedMultiple) ImplementedThis() ImplementedInterface { return t }

type testInvalidTooManyReturns struct {
	bad badReturnEmbedded
}

type badReturnEmbedded struct{}

func (b *badReturnEmbedded) InitEmbedded(embedder any) (int, error) {
	return 0, nil
}

func TestNewRegistryWithErrorReturns(t *testing.T) {
	reg := NewRegistry("InitEmbedded")

	embedders := []any{
		&testErrEmbedded1{},
		&testErrEmbedded2{},
		&testErrEmbedded3{},
		&testErrEmbedded4{},
		&testErrEmbeddedPtr{},
		&testErrEmbeddedDeep1{},
		&testErrEmbeddedDeep2{},
		&testErrEmbeddedMultiple{},
	}

	for _, obj := range embedders {
		reg.Register(obj)
		if reg.Count(obj) == 0 {
			t.Fatalf("expected %T to have registered initializers, got 0", obj)
		}
	}

	t.Run("BindSuccess", func(t *testing.T) {
		e1 := &testErrEmbedded1{}
		if err := reg.Bind(t.Context(), e1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e1.calls != 1 || e1.backRef != e1 {
			t.Fatalf("expected 1 call and valid backRef, got calls=%d backRef=%v", e1.calls, e1.backRef)
		}

		e2 := &testErrEmbedded2{}
		if err := reg.Bind(t.Context(), e2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e2.calls != 1 || e2.backRef != e2 {
			t.Fatalf("expected 1 call and valid backRef, got calls=%d backRef=%v", e2.calls, e2.backRef)
		}

		e3 := &testErrEmbedded3{}
		if err := reg.Bind(t.Context(), e3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e3.calls != 1 || e3.backRef != e3 {
			t.Fatalf("expected 1 call and valid backRef, got calls=%d backRef=%v", e3.calls, e3.backRef)
		}

		e4 := &testErrEmbedded4{}
		if err := reg.Bind(t.Context(), e4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e4.calls != 1 || e4.backRef != e4 || e4.fieldName != "errEmbedded4" {
			t.Fatalf("unexpected state: %+v", e4)
		}

		ePtr := &testErrEmbeddedPtr{}
		if err := reg.Bind(t.Context(), ePtr); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ePtr.errEmbedded1 == nil || ePtr.calls != 1 || ePtr.backRef != ePtr {
			t.Fatalf("expected allocated pointer and valid backRef, got %+v", ePtr.errEmbedded1)
		}
	})

	t.Run("BindDirectError", func(t *testing.T) {
		cases := []struct {
			name string
			obj  any
		}{
			{
				name: "signature (any) error",
				obj:  &testErrEmbedded1{errEmbedded1: errEmbedded1[testErrEmbedded1]{shouldErr: true}},
			},
			{
				name: "signature (ctx, T) error",
				obj:  &testErrEmbedded2{errEmbedded2: errEmbedded2[*testErrEmbedded2]{shouldErr: true}},
			},
			{
				name: "signature (iface, StructField) error",
				obj:  &testErrEmbedded3{errEmbedded3: errEmbedded3{shouldErr: true}},
			},
			{
				name: "signature (ctx, reflect.Value, StructField) error",
				obj:  &testErrEmbedded4{errEmbedded4: errEmbedded4{shouldErr: true}},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := reg.Bind(t.Context(), tc.obj)
				if err == nil {
					t.Fatalf("expected error for %T, got nil", tc.obj)
				}
				if !errors.Is(err, errTestSetup) {
					t.Fatalf("expected error wrapping %v, got %v", errTestSetup, err)
				}
			})
		}
	})

	t.Run("BindReflectValueError", func(t *testing.T) {
		obj := &testErrEmbedded1{
			errEmbedded1: errEmbedded1[testErrEmbedded1]{shouldErr: true},
		}
		err := reg.Bind(t.Context(), reflect.ValueOf(obj))
		if !errors.Is(err, errTestSetup) {
			t.Fatalf("expected error wrapping %v, got %v", errTestSetup, err)
		}
	})

	t.Run("BindDeepErrorPath", func(t *testing.T) {
		deep := &testErrEmbeddedDeep2{
			outer: &testErrEmbeddedDeep1{
				setup: &testErrEmbedded1{
					errEmbedded1: errEmbedded1[testErrEmbedded1]{shouldErr: true},
				},
			},
		}

		err := reg.Bind(t.Context(), deep)
		if err == nil {
			t.Fatal("expected error from deep embedded struct, got nil")
		}
		if !errors.Is(err, errTestSetup) {
			t.Fatalf("expected error wrapping %v, got %v", errTestSetup, err)
		}
		if !strings.Contains(err.Error(), "outer.setup") {
			t.Fatalf("expected error message to contain field path 'outer.setup', got %q", err.Error())
		}
	})

	t.Run("BindShortCircuitsOnFirstError", func(t *testing.T) {
		multi := &testErrEmbeddedMultiple{
			first:  errEmbedded1[testErrEmbeddedMultiple]{shouldErr: false},
			second: errEmbedded2[*testErrEmbeddedMultiple]{shouldErr: true},
			third:  errEmbedded3{shouldErr: false},
		}

		err := reg.Bind(t.Context(), multi)
		if !errors.Is(err, errTestSetup) {
			t.Fatalf("expected error wrapping %v, got %v", errTestSetup, err)
		}
		if multi.first.calls != 1 {
			t.Fatalf("expected first initializer to be called once, got %d", multi.first.calls)
		}
		if multi.second.calls != 1 {
			t.Fatalf("expected second initializer to be called once, got %d", multi.second.calls)
		}
		if multi.third.calls != 0 {
			t.Fatalf("expected third initializer not to be called after error, got %d", multi.third.calls)
		}
	})

	t.Run("RegisterPanicsOnMultipleReturnValues", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic when registering method with >1 return values, got nil")
			}
		}()

		reg.Register(&testInvalidTooManyReturns{})
	})
}
