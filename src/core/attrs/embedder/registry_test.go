package embedder

import (
	"context"
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
		Register(obj)
		count := Count(obj)

		if count == 0 {
			t.Fatalf("Object %T has no embedders: %v", obj, initRegistry.typeReg[key(obj)])
		}

		embedderCount += count
	}

	if embedderCount < len(embedders) {
		t.Fatalf("expected at least %d embedders, got %d", len(embedders), embedderCount)
	}

	// reset the registry
	initRegistry = &modelRegistry{
		method:  "SetupEmbedded",
		typeReg: make(map[uintptr][]*initializer, len(initRegistry.typeReg)),
	}

	t.Run("Register", func(t *testing.T) {
		for _, embedder := range embedders {

			i, ok := initRegistry.typeReg[key(embedder)]
			if ok {
				t.Fatalf("retrieved %d initialisers", len(i))
			}

			Register(embedder)

			i, ok = initRegistry.typeReg[key(embedder)]
			if !ok {
				t.Fatalf("could not retrieve initialisers for %T", embedder)
			}

			if len(i) != Count(embedder) {
				t.Fatalf("expected 1 initialiser per type, got %d (%T)", len(i), embedder)
			}
		}
	})

	t.Run("Walk", func(t *testing.T) {
		for _, embedder := range embedders {
			t.Logf("walking embedder tree for %T:", embedder)
			Walk(embedder, func(depth int, field reflect.StructField) (stop bool) {
				t.Logf("%sStructField %s (%v)", strings.Repeat("  ", depth), field.Name, field.Index)
				return false
			})
		}
	})

	t.Run("Bind", func(t *testing.T) {
		cC := callCount

		for _, embedder := range embedders {
			currCallCnt := callCount

			Bind(t.Context(), embedder)

			count := Count(embedder)

			if count == 0 {
				t.Fatalf("expected at least 1 embedded object to be initialized")
			}

			if currCallCnt+Count(embedder) != callCount {
				t.Fatalf("expected bind func to be called %d times, got %d", Count(embedder), callCount-currCallCnt)
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

			Bind(t.Context(), reflect.ValueOf(embedder))

			if currCallCnt+Count(embedder) != callCount {
				t.Fatalf("expected bind func to be called %d times, got %d", Count(embedder), callCount-currCallCnt)
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
		Register(embedder)
	}

	for _, embedder := range embedders {
		b.Run(fmt.Sprintf("Bind/%T", embedder), func(b *testing.B) {
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Sprintf("%T: %v", embedder, r))
				}
			}()
			for b.Loop() {
				Bind(b.Context(), embedder)
			}
		})
	}

	for _, embedder := range embedders {
		b.Run(fmt.Sprintf("Count/%T", embedder), func(b *testing.B) {
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Sprintf("%T: %v", embedder, r))
				}
			}()

			for b.Loop() {
				_ = Count(embedder)
			}
		})
	}

	for _, embedder := range embedders {
		b.Run(fmt.Sprintf("Bind/Reflect/%T", embedder), func(b *testing.B) {
			var embed any = reflect.ValueOf(embedder)
			for b.Loop() {
				Bind(b.Context(), embed)
			}
		})
	}
}
