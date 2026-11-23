# Deconstruction and Reconstruction

The `github.com/Nigel2392/go-django/src/core/deconstruct` package provides Django-style object serialization and polymorphic deserialization using Go's `encoding/json/v2` and `encoding/json/jsontext` packages. Types expose a canonical constructor path, positional arguments, and keyword arguments so interfaces and concrete values can round-trip cleanly through JSON.

---

## Core Types and Interfaces

```go
package deconstruct

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

type Deconstructible interface {
	Deconstruct() (path string, args []any, kwargs map[string]any)
}

type Kwargs map[string]any

func (k Kwargs) Get[T any](name string) (val T)
func (k Kwargs) GetOk[T any](name string) (val T, ok bool)

type Deconstructed struct {
	Path   string           `json:"path"`
	Args   []jsontext.Value `json:"args"`
	Kwargs Kwargs           `json:"kwargs"`
}

type Reconstructor func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error)
```

- **`Deconstructible`**: Implemented by values that can serialize themselves into `(path, args, kwargs)`.
- **`Kwargs`**: Convenience map wrapper with generic `Get[T]` and `GetOk[T]` accessors that safely handle `nil` maps and mismatched types.
- **`Deconstructed`**: JSON wire envelope (`{"path": ..., "args": [...], "kwargs": {...}}`).
- **`Reconstructor`**: Factory function that reconstructs a Go value from decoded JSON `args`, `kwargs`, and inherited `json.Options`.

---

## `ConstructorRegistry` and Package Helpers

The package maintains a global `Constructors` registry (also used by built-in types such as [`trans.Lazy`](./translations.md)):

```go
var Constructors = ConstructorRegistry{ /* ... */ }

func (c *ConstructorRegistry) Register(path string, fn Reconstructor)
func (c *ConstructorRegistry) RegisterDestructibleType[T any]()
func (c *ConstructorRegistry) JSONOptions() json.Options
func (c *ConstructorRegistry) Unmarshal(data []byte, dst any, opts ...json.Options) error

func UnmarshalDefault[OUT any, DATA ~[]byte](data DATA, opts ...json.Options) (OUT, error)
```

- **`Register(path, fn)`**: Associates a deconstruction `path` with its `Reconstructor` function.
- **`RegisterDestructibleType[T]()`**: Registers a `json/v2` unmarshaler for custom interfaces (such as `Validator` or `Field`) or non-`Deconstructible` types so JSON unmarshaling into `*T`, `[]T`, `map[string]T`, or struct fields of type `T` invokes the registry automatically.
- **`JSONOptions()`**: Returns `json.Options` combining the registry's custom marshaler (for `Deconstructible`) and unmarshalers (for `Deconstructible` plus every type registered via `RegisterDestructibleType`). Pass this to `json.Marshal` when serializing `Deconstructible` graphs.
- **`Unmarshal(data, dst, opts...)`**: Unmarshals `data` into `dst` with `c.JSONOptions()` prepended to any caller-supplied `opts...`.
- **`UnmarshalDefault[OUT, DATA](data, opts...)`**: Generic wrapper around `json.Unmarshal` returning `(OUT, error)`.

---

## Usage Examples

### Implementing `Deconstructible` and Registering a Reconstructor

```go
package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/Nigel2392/go-django/src/core/deconstruct"
)

type Validator interface {
	deconstruct.Deconstructible
	Validate(v any) error
}

type MaxLengthValidator struct {
	Limit   int
	Message string
}

func (v MaxLengthValidator) Deconstruct() (string, []any, map[string]any) {
	kw := deconstruct.Kwargs{}
	if v.Message != "" {
		kw["message"] = v.Message
	}
	return "validators.MaxLengthValidator", []any{v.Limit}, kw
}

func (v MaxLengthValidator) Validate(val any) error {
	s, ok := val.(string)
	if !ok || len(s) > v.Limit {
		return fmt.Errorf("exceeds max length %d", v.Limit)
	}
	return nil
}

func init() {
	// Register custom sub-interface so []Validator / Validator targets unmarshal polymorphically
	deconstruct.Constructors.RegisterDestructibleType[Validator]()

	deconstruct.Constructors.Register(
		"validators.MaxLengthValidator",
		func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 arg, got %d", len(args))
			}

			var limit int
			if err := json.Unmarshal(args[0], &limit, opts...); err != nil {
				return nil, err
			}

			return MaxLengthValidator{
				Limit:   limit,
				Message: kwargs.Get[string]("message"),
			}, nil
		},
	)
}
```

### Polymorphic Marshaling and Unmarshaling

Once types and interfaces are registered, structs containing concrete fields, `Deconstructible` interfaces, or custom interface slices round-trip directly:

```go
type FieldSpec struct {
	Name       string      `json:"name"`
	Validators []Validator `json:"validators"`
}

func main() {
	spec := FieldSpec{
		Name: "username",
		Validators: []Validator{
			MaxLengthValidator{Limit: 150, Message: "Username too long"},
		},
	}

	raw, err := json.Marshal(spec, deconstruct.Constructors.JSONOptions())
	if err != nil {
		panic(err)
	}

	var decoded FieldSpec
	if err := deconstruct.Constructors.Unmarshal(raw, &decoded); err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", decoded)
}
```

### Handling Nested `Deconstructible` Values Inside `Kwargs`

When a reconstructor reads nested `Deconstructible` fields from `kwargs` (where JSON objects have already been decoded into `map[string]any`), re-marshal the raw `kwargs["field"]` value and unmarshal it using the inherited `opts...`:

```go
if rawNested, ok := kwargs["field"]; ok {
	b, err := json.Marshal(rawNested, opts...)
	if err != nil {
		return nil, err
	}
	var field MyFieldInterface
	if err := json.Unmarshal(b, &field, opts...); err != nil {
		return nil, err
	}
}
```

---

## Caveats

- **JSON numbers in `Kwargs` decode as `float64`**: Because `Kwargs` is `map[string]any`, numeric values inside `kwargs` decode as `float64` (not `int`). Retrieve them with `kwargs.GetOk[float64](...)` and cast to `int`, or pass integers in positional `Args` where `json.Unmarshal(args[i], &target, opts...)` decodes directly into `int`.
- **Custom interface targets require `RegisterDestructibleType[T]()`**: By default, `JSONOptions()` only attaches an unmarshaler for `deconstruct.Deconstructible`. If you unmarshal into a sub-interface such as `Validator` or `Field` (or a slice/map of that interface), you must call `RegisterDestructibleType[Validator]()` first.
- **`Deconstructible` target enforcement**: Unmarshaling directly into a `deconstruct.Deconstructible` variable requires the reconstructor's returned value (if non-nil) to implement `Deconstructible`; otherwise `Unmarshal` returns `"reconstructor for ... returned non-Deconstructible ..."`.
