package deconstruct

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// -----------------------------------------------------------------------------
// Test Domain Fixtures (Django-like Fields, Validators, Operations, External)
// -----------------------------------------------------------------------------

type Validator interface {
	Deconstructible
	Validate(v any) error
}

type Field interface {
	Deconstructible
	FieldName() string
}

type Operation interface {
	Deconstructible
	OpKind() string
}

// Non-Deconstructible external type to test RegisterDestructibleType with plain types
type ExternalDriver struct {
	Name string
	DSN  string
}

// 1. Simple value-receiver Deconstructible with positional args + kwargs
type MaxLengthValidator struct {
	Limit   int
	Message string
}

func (v MaxLengthValidator) Deconstruct() (string, []any, map[string]any) {
	kw := Kwargs{}
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

// 2. Pointer-receiver Deconstructible with multiple args
type RangeValidator struct {
	Min int
	Max int
}

func (v *RangeValidator) Deconstruct() (string, []any, map[string]any) {
	return "validators.RangeValidator", []any{v.Min, v.Max}, nil
}

func (v *RangeValidator) Validate(val any) error {
	n, ok := val.(int)
	if !ok || n < v.Min || n > v.Max {
		return fmt.Errorf("out of range [%d, %d]", v.Min, v.Max)
	}
	return nil
}

// 3. Field containing nested Deconstructibles in Kwargs (Validators)
type CharField struct {
	Name       string
	MaxLength  int
	Nullable   bool
	Default    string
	Validators []Validator
}

func (f *CharField) FieldName() string { return f.Name }

func (f *CharField) Deconstruct() (string, []any, map[string]any) {
	kw := Kwargs{
		"max_length": f.MaxLength,
	}
	if f.Nullable {
		kw["nullable"] = true
	}
	if f.Default != "" {
		kw["default"] = f.Default
	}
	if len(f.Validators) > 0 {
		kw["validators"] = f.Validators
	}
	return "fields.CharField", []any{f.Name}, kw
}

// 4. Field containing a nested Deconstructible in Args (OnDelete policy)
type CascadePolicy struct {
	Name string
}

func (p CascadePolicy) Deconstruct() (string, []any, map[string]any) {
	return "models." + p.Name, nil, nil
}

type ForeignKey struct {
	Name     string
	To       string
	OnDelete Deconstructible
	Nullable bool
}

func (f *ForeignKey) FieldName() string { return f.Name }

func (f *ForeignKey) Deconstruct() (string, []any, map[string]any) {
	return "fields.ForeignKey", []any{f.To, f.OnDelete}, Kwargs{
		"name":     f.Name,
		"nullable": f.Nullable,
	}
}

// 5. Concrete (non-Deconstructible) wrapper struct holding Deconstructible interfaces
type Migration struct {
	App          string      `json:"app"`
	Name         string      `json:"name"`
	Dependencies [][2]string `json:"dependencies,omitempty"`
	Operations   []Operation `json:"operations"`
}

type AddFieldOp struct {
	ModelName string
	Field     Field
}

func (op *AddFieldOp) OpKind() string { return "add_field" }

func (op *AddFieldOp) Deconstruct() (string, []any, map[string]any) {
	return "migrations.AddField", nil, Kwargs{
		"model_name": op.ModelName,
		"field":      op.Field,
	}
}

// 6. Deconstructible with empty args and nil kwargs
type NoOp struct{}

func (NoOp) OpKind() string { return "nop" }

func (NoOp) Deconstruct() (string, []any, map[string]any) {
	return "migrations.NoOp", nil, nil
}

// -----------------------------------------------------------------------------
// Helper to build a populated registry for tests
// -----------------------------------------------------------------------------

func newTestRegistry(t testing.TB) *ConstructorRegistry {
	t.Helper()
	reg := &ConstructorRegistry{
		constructors: make(map[string]Reconstructor),
	}

	reg.RegisterDestructibleType[Validator]()
	reg.RegisterDestructibleType[Field]()
	reg.RegisterDestructibleType[Operation]()

	reg.Register("validators.MaxLengthValidator", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
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
	})

	reg.Register("validators.RangeValidator", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 args, got %d", len(args))
		}
		var minV, maxV int
		if err := json.Unmarshal(args[0], &minV, opts...); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(args[1], &maxV, opts...); err != nil {
			return nil, err
		}
		return &RangeValidator{Min: minV, Max: maxV}, nil
	})

	reg.Register("models.CASCADE", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return CascadePolicy{Name: "CASCADE"}, nil
	})

	reg.Register("models.RESTRICT", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return CascadePolicy{Name: "RESTRICT"}, nil
	})

	reg.Register("fields.CharField", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected 1 arg, got %d", len(args))
		}
		var name string
		if err := json.Unmarshal(args[0], &name, opts...); err != nil {
			return nil, err
		}
		maxLenFloat, ok := kwargs.GetOk[float64]("max_length")
		if !ok {
			return nil, errors.New("missing or invalid max_length")
		}

		var validators []Validator
		if rawVals, hasVals := kwargs["validators"]; hasVals {
			rawBytes, err := json.Marshal(rawVals)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(rawBytes, &validators, opts...); err != nil {
				return nil, err
			}
		}

		return &CharField{
			Name:       name,
			MaxLength:  int(maxLenFloat),
			Nullable:   kwargs.Get[bool]("nullable"),
			Default:    kwargs.Get[string]("default"),
			Validators: validators,
		}, nil
	})

	reg.Register("fields.ForeignKey", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 args, got %d", len(args))
		}
		var to string
		if err := json.Unmarshal(args[0], &to, opts...); err != nil {
			return nil, err
		}
		var onDelete Deconstructible
		if err := json.Unmarshal(args[1], &onDelete, opts...); err != nil {
			return nil, err
		}
		return &ForeignKey{
			Name:     kwargs.Get[string]("name"),
			To:       to,
			OnDelete: onDelete,
			Nullable: kwargs.Get[bool]("nullable"),
		}, nil
	})

	reg.Register("migrations.AddField", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		modelName, ok := kwargs.GetOk[string]("model_name")
		if !ok {
			return nil, errors.New("missing model_name")
		}
		rawField, ok := kwargs["field"]
		if !ok {
			return nil, errors.New("missing field")
		}
		b, err := json.Marshal(rawField)
		if err != nil {
			return nil, err
		}
		var field Field
		if err := json.Unmarshal(b, &field, opts...); err != nil {
			return nil, err
		}
		return &AddFieldOp{
			ModelName: modelName,
			Field:     field,
		}, nil
	})

	reg.Register("migrations.NoOp", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return NoOp{}, nil
	})

	return reg
}

// -----------------------------------------------------------------------------
// 1. Kwargs.Get & Kwargs.GetOk Unit Tests
// -----------------------------------------------------------------------------

func TestKwargsGetAndGetOk(t *testing.T) {
	kw := Kwargs{
		"str":     "hello",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"nil_val": nil,
		"slice":   []string{"a", "b"},
		"nested":  Kwargs{"inner": 99},
	}

	t.Run("GetOk existing matching types", func(t *testing.T) {
		if v, ok := kw.GetOk[string]("str"); !ok || v != "hello" {
			t.Fatalf("expected ('hello', true), got (%q, %v)", v, ok)
		}
		if v, ok := kw.GetOk[int]("int"); !ok || v != 42 {
			t.Fatalf("expected (42, true), got (%d, %v)", v, ok)
		}
		if v, ok := kw.GetOk[float64]("float"); !ok || v != 3.14 {
			t.Fatalf("expected (3.14, true), got (%v, %v)", v, ok)
		}
		if v, ok := kw.GetOk[bool]("bool"); !ok || !v {
			t.Fatalf("expected (true, true), got (%v, %v)", v, ok)
		}
		if v, ok := kw.GetOk[[]string]("slice"); !ok || len(v) != 2 {
			t.Fatalf("expected slice len 2, got (%v, %v)", v, ok)
		}
		if v, ok := kw.GetOk[Kwargs]("nested"); !ok || v["inner"] != 99 {
			t.Fatalf("expected nested Kwargs, got (%v, %v)", v, ok)
		}
	})

	t.Run("Get returns value or zero value", func(t *testing.T) {
		if got := kw.Get[string]("str"); got != "hello" {
			t.Fatalf("expected 'hello', got %q", got)
		}
		if got := kw.Get[string]("missing"); got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
		if got := kw.Get[int]("str"); got != 0 {
			t.Fatalf("expected 0 on type mismatch, got %d", got)
		}
		if got := kw.Get[bool]("nil_val"); got {
			t.Fatalf("expected false for nil value, got true")
		}
	})

	t.Run("missing key returns zero and false", func(t *testing.T) {
		if v, ok := kw.GetOk[string]("does_not_exist"); ok || v != "" {
			t.Fatalf("expected ('', false), got (%q, %v)", v, ok)
		}
	})

	t.Run("wrong type assertion returns zero and false", func(t *testing.T) {
		if v, ok := kw.GetOk[int]("str"); ok || v != 0 {
			t.Fatalf("expected (0, false), got (%d, %v)", v, ok)
		}
		if v, ok := kw.GetOk[int]("float"); ok || v != 0 {
			t.Fatalf("expected (0, false), got (%d, %v)", v, ok)
		}
	})

	t.Run("nil value returns zero and false for concrete types", func(t *testing.T) {
		if v, ok := kw.GetOk[string]("nil_val"); ok || v != "" {
			t.Fatalf("expected ('', false), got (%q, %v)", v, ok)
		}
	})

	t.Run("nil Kwargs map does not panic", func(t *testing.T) {
		var nilKw Kwargs
		if v, ok := nilKw.GetOk[string]("any"); ok || v != "" {
			t.Fatalf("expected ('', false), got (%q, %v)", v, ok)
		}
		if v := nilKw.Get[int]("any"); v != 0 {
			t.Fatalf("expected 0, got %d", v)
		}
	})
}

// -----------------------------------------------------------------------------
// 2. Basic & Polymorphic Round-Trip Tests
// -----------------------------------------------------------------------------

func TestMarshalUnmarshal_SimpleTypes(t *testing.T) {
	reg := newTestRegistry(t)

	tests := []struct {
		name  string
		input Deconstructible
	}{
		{
			name:  "value receiver with args and kwargs",
			input: MaxLengthValidator{Limit: 120, Message: "too long"},
		},
		{
			name:  "value receiver with empty kwargs",
			input: MaxLengthValidator{Limit: 50},
		},
		{
			name:  "pointer receiver with multiple positional args",
			input: &RangeValidator{Min: -10, Max: 100},
		},
		{
			name:  "empty args and nil kwargs",
			input: NoOp{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.input, reg.JSONOptions())
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var got Deconstructible
			if err := reg.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal failed (json=%s): %v", string(raw), err)
			}

			if !reflect.DeepEqual(tc.input, got) {
				t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", tc.input, got)
			}
		})
	}
}

func TestJSONEnvelopeStructure(t *testing.T) {
	reg := newTestRegistry(t)

	input := MaxLengthValidator{Limit: 255, Message: "max 255"}
	raw, err := json.Marshal(input, reg.JSONOptions())
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var env Deconstructed
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("failed to unmarshal into Deconstructed struct: %v", err)
	}

	if env.Path != "validators.MaxLengthValidator" {
		t.Errorf("unexpected path: got %q", env.Path)
	}
	if len(env.Args) != 1 || string(env.Args[0]) != "255" {
		t.Errorf("unexpected args: got %v", env.Args)
	}
	if msg, ok := env.Kwargs.GetOk[string]("message"); !ok || msg != "max 255" {
		t.Errorf("unexpected kwargs: got %#v", env.Kwargs)
	}
}

func TestTopLevelNull(t *testing.T) {
	reg := newTestRegistry(t)

	t.Run("Deconstructible becomes nil", func(t *testing.T) {
		var dst Deconstructible = NoOp{}
		if err := reg.Unmarshal([]byte("null"), &dst); err != nil {
			t.Fatalf("unexpected error unmarshaling null: %v", err)
		}
		if dst != nil {
			t.Fatalf("expected nil, got %#v", dst)
		}
	})

	t.Run("registered sub-interface becomes nil", func(t *testing.T) {
		var dst Operation = NoOp{}
		if err := reg.Unmarshal([]byte("null"), &dst); err != nil {
			t.Fatalf("unexpected error unmarshaling null into Operation: %v", err)
		}
		if dst != nil {
			t.Fatalf("expected nil Operation, got %#v", dst)
		}
	})
}

// -----------------------------------------------------------------------------
// 3. RegisterDestructibleType Tests (Sub-Interfaces & External Types)
// -----------------------------------------------------------------------------

func TestRegisterDestructibleType_DirectSubInterfaceUnmarshal(t *testing.T) {
	reg := newTestRegistry(t)

	t.Run("single Field interface", func(t *testing.T) {
		original := Field(&CharField{
			Name:      "slug",
			MaxLength: 80,
			Validators: []Validator{
				MaxLengthValidator{Limit: 80},
			},
		})

		raw, err := json.Marshal(original, reg.JSONOptions())
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var got Field
		if err := reg.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal into Field failed: %v", err)
		}
		if !reflect.DeepEqual(original, got) {
			t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", original, got)
		}
	})

	t.Run("slice of Operation with nil element", func(t *testing.T) {
		original := []Operation{
			NoOp{},
			nil,
			&AddFieldOp{
				ModelName: "Post",
				Field:     &CharField{Name: "title", MaxLength: 200},
			},
		}

		raw, err := json.Marshal(original, reg.JSONOptions())
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var got []Operation
		if err := reg.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal []Operation failed: %v", err)
		}
		if !reflect.DeepEqual(original, got) {
			t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", original, got)
		}
	})

	t.Run("map of Field interfaces", func(t *testing.T) {
		original := map[string]Field{
			"title":  &CharField{Name: "title", MaxLength: 100},
			"author": &ForeignKey{Name: "author", To: "auth.User", OnDelete: CascadePolicy{Name: "RESTRICT"}},
		}

		raw, err := json.Marshal(original, reg.JSONOptions())
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var got map[string]Field
		if err := reg.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal map[string]Field failed: %v", err)
		}
		if !reflect.DeepEqual(original, got) {
			t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", original, got)
		}
	})
}

func TestRegisterDestructibleType_NonDeconstructibleType(t *testing.T) {
	reg := &ConstructorRegistry{
		constructors: make(map[string]Reconstructor),
	}
	reg.RegisterDestructibleType[*ExternalDriver]()
	reg.Register("drivers.mysql", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return &ExternalDriver{
			Name: "mysql",
			DSN:  kwargs.Get[string]("dsn"),
		}, nil
	})

	payload := []byte(`{"path":"drivers.mysql","args":[],"kwargs":{"dsn":"root@tcp(localhost)/app"}}`)

	var drv *ExternalDriver
	if err := reg.Unmarshal(payload, &drv); err != nil {
		t.Fatalf("unmarshal into *ExternalDriver failed: %v", err)
	}
	want := &ExternalDriver{Name: "mysql", DSN: "root@tcp(localhost)/app"}
	if !reflect.DeepEqual(drv, want) {
		t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", want, drv)
	}

	// Null unmarshals to nil pointer
	if err := reg.Unmarshal([]byte("null"), &drv); err != nil {
		t.Fatalf("unmarshal null into *ExternalDriver failed: %v", err)
	}
	if drv != nil {
		t.Fatalf("expected nil pointer, got %#v", drv)
	}
}

func TestRegisterDestructibleType_TypeMismatchAndNilReturn(t *testing.T) {
	reg := newTestRegistry(t)

	t.Run("wrong reconstructed type for target interface returns error", func(t *testing.T) {
		// migrations.NoOp implements Operation, NOT Field
		payload := []byte(`{"path":"migrations.NoOp","args":[],"kwargs":{}}`)
		var dst Field
		err := reg.Unmarshal(payload, &dst)
		if err == nil {
			t.Fatalf("expected assignment error when unmarshaling NoOp into Field")
		}
		if !strings.Contains(err.Error(), "cannot be assigned to") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("reconstructor returning (nil, nil) yields zero value", func(t *testing.T) {
		reg.Register("special.Nil", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
			return nil, nil
		})
		payload := []byte(`{"path":"special.Nil","args":[],"kwargs":{}}`)

		var dec Deconstructible = NoOp{}
		if err := reg.Unmarshal(payload, &dec); err != nil {
			t.Fatalf("unexpected error for Deconstructible nil return: %v", err)
		}
		if dec != nil {
			t.Fatalf("expected nil Deconstructible, got %#v", dec)
		}

		var op Operation = NoOp{}
		if err := reg.Unmarshal(payload, &op); err != nil {
			t.Fatalf("unexpected error for Operation nil return: %v", err)
		}
		if op != nil {
			t.Fatalf("expected nil Operation, got %#v", op)
		}
	})

	t.Run("reconstructor returning non-Deconstructible fails on *Deconstructible", func(t *testing.T) {
		reg.Register("special.External", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
			return &ExternalDriver{Name: "sqlite"}, nil
		})
		payload := []byte(`{"path":"special.External","args":[],"kwargs":{}}`)

		var dec Deconstructible
		err := reg.Unmarshal(payload, &dec)
		if err == nil || !strings.Contains(err.Error(), "returned non-Deconstructible") {
			t.Fatalf("expected non-Deconstructible error, got: %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// 4. Nested Deconstructibles (Args, Kwargs, Slices, Maps, Concrete Structs)
// -----------------------------------------------------------------------------

func TestNestedDeconstructibles_InArgs(t *testing.T) {
	reg := newTestRegistry(t)

	fk := &ForeignKey{
		Name:     "author_id",
		To:       "auth.User",
		OnDelete: CascadePolicy{Name: "RESTRICT"},
		Nullable: false,
	}

	raw, err := json.Marshal(fk, reg.JSONOptions())
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got Deconstructible
	if err := reg.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(fk, got) {
		t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", fk, got)
	}
}

func TestNestedDeconstructibles_InKwargsAndSlices(t *testing.T) {
	reg := newTestRegistry(t)

	field := &CharField{
		Name:      "username",
		MaxLength: 150,
		Nullable:  false,
		Default:   "guest",
		Validators: []Validator{
			MaxLengthValidator{Limit: 150, Message: "Username too long"},
			&RangeValidator{Min: 3, Max: 150},
		},
	}

	raw, err := json.Marshal(field, reg.JSONOptions())
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got Deconstructible
	if err := reg.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed (json=%s): %v", string(raw), err)
	}

	if !reflect.DeepEqual(field, got) {
		t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", field, got)
	}
}

func TestMixedConcreteStructTree_MigrationDirectUnmarshal(t *testing.T) {
	reg := newTestRegistry(t)

	// Unmarshals directly into Migration (Operations []Operation) via RegisterDestructibleType[Operation]()
	original := Migration{
		App:          "blog",
		Name:         "0002_add_comment_fields",
		Dependencies: [][2]string{{"blog", "0001_initial"}, {"auth", "0001_initial"}},
		Operations: []Operation{
			&AddFieldOp{
				ModelName: "Comment",
				Field: &CharField{
					Name:      "body",
					MaxLength: 2000,
					Validators: []Validator{
						MaxLengthValidator{Limit: 2000},
					},
				},
			},
			&AddFieldOp{
				ModelName: "Comment",
				Field: &ForeignKey{
					Name:     "author",
					To:       "auth.User",
					OnDelete: CascadePolicy{Name: "CASCADE"},
					Nullable: true,
				},
			},
			NoOp{},
		},
	}

	raw, err := json.Marshal(original, reg.JSONOptions(), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal migration failed: %v", err)
	}

	var decoded Migration
	if err := reg.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal migration failed:\n%s\nerror: %v", string(raw), err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("migration mismatch:\n  want: %#v\n  got:  %#v", original, decoded)
	}
}

func TestMapOfDeconstructibles(t *testing.T) {
	reg := newTestRegistry(t)

	original := map[string]Deconstructible{
		"title": &CharField{Name: "title", MaxLength: 100},
		"owner": &ForeignKey{Name: "owner", To: "auth.User", OnDelete: CascadePolicy{Name: "CASCADE"}},
		"noop":  NoOp{},
	}

	raw, err := json.Marshal(original, reg.JSONOptions())
	if err != nil {
		t.Fatalf("marshal map failed: %v", err)
	}

	var decoded map[string]Deconstructible
	if err := reg.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal map failed: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("map mismatch:\n  want: %#v\n  got:  %#v", original, decoded)
	}
}

func TestSliceWithNilAndMixedElements(t *testing.T) {
	reg := newTestRegistry(t)

	original := []Deconstructible{
		NoOp{},
		nil,
		MaxLengthValidator{Limit: 10},
	}

	raw, err := json.Marshal(original, reg.JSONOptions())
	if err != nil {
		t.Fatalf("marshal slice with nil failed: %v", err)
	}

	var decoded []Deconstructible
	if err := reg.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal slice with nil failed: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("slice mismatch:\n  want: %#v\n  got:  %#v", original, decoded)
	}
}

// -----------------------------------------------------------------------------
// 5. Parent json.Options Propagation & Registry Isolation Tests
// -----------------------------------------------------------------------------

func TestParentOptionsPropagation(t *testing.T) {
	reg := newTestRegistry(t)

	t.Run("marshal propagates StringifyNumbers into args and kwargs", func(t *testing.T) {
		v := MaxLengthValidator{Limit: 42, Message: "ok"}
		raw, err := json.Marshal(v, reg.JSONOptions(), json.StringifyNumbers(true))
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		// enc.Options() must have propagated StringifyNumbers(true) to args[0]
		if !strings.Contains(string(raw), `"42"`) {
			t.Fatalf("expected stringified number \"42\" in args, got: %s", string(raw))
		}

		// Without StringifyNumbers(true), decoding "42" into int fails
		var failDst Deconstructible
		if err := reg.Unmarshal(raw, &failDst); err == nil {
			t.Fatalf("expected error decoding stringified int without StringifyNumbers option")
		}

		// With StringifyNumbers(true), dec.Options() forwards it to Reconstructor's opts...
		var okDst Deconstructible
		if err := reg.Unmarshal(raw, &okDst, json.StringifyNumbers(true)); err != nil {
			t.Fatalf("unmarshal with propagated StringifyNumbers failed: %v", err)
		}
		if !reflect.DeepEqual(v, okDst) {
			t.Fatalf("mismatch:\n  want: %#v\n  got:  %#v", v, okDst)
		}
	})

	t.Run("unmarshal propagates RejectUnknownMembers into nested reconstructor args", func(t *testing.T) {
		// Nested OnDelete envelope inside ForeignKey's args[1] has an unknown member "extra"
		payload := []byte(`{
			"path": "fields.ForeignKey",
			"args": [
				"auth.User",
				{"path": "models.CASCADE", "args": [], "kwargs": {}, "extra": true}
			],
			"kwargs": {"name": "author"}
		}`)

		// Allowed by default
		var okDst Deconstructible
		if err := reg.Unmarshal(payload, &okDst); err != nil {
			t.Fatalf("expected default unmarshal to ignore unknown member, got: %v", err)
		}

		// Rejected when caller passes RejectUnknownMembers(true)
		var strictDst Deconstructible
		if err := reg.Unmarshal(payload, &strictDst, json.RejectUnknownMembers(true)); err == nil {
			t.Fatalf("expected RejectUnknownMembers(true) to reject unknown field in nested arg")
		}
	})

	t.Run("unmarshal respects AllowDuplicateNames(false)", func(t *testing.T) {
		dupJSON := []byte(`{"path":"migrations.NoOp","path":"migrations.NoOp","args":null,"kwargs":null}`)
		var dst Deconstructible
		if err := reg.Unmarshal(dupJSON, &dst, jsontext.AllowDuplicateNames(false)); err == nil {
			t.Fatalf("expected error when duplicate JSON names are disallowed")
		}
	})
}

func TestRegistryIsolation(t *testing.T) {
	regA := &ConstructorRegistry{constructors: make(map[string]Reconstructor)}
	regB := &ConstructorRegistry{constructors: make(map[string]Reconstructor)}

	regA.Register("migrations.NoOp", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return NoOp{}, nil
	})

	raw, err := json.Marshal(NoOp{}, regA.JSONOptions())
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var dstA Deconstructible
	if err := regA.Unmarshal(raw, &dstA); err != nil {
		t.Fatalf("regA should unmarshal its own registered type: %v", err)
	}

	var dstB Deconstructible
	err = regB.Unmarshal(raw, &dstB)
	if err == nil {
		t.Fatalf("expected regB to fail for unregistered path, but it succeeded")
	}
	if !strings.Contains(err.Error(), "unregistered deconstruct path") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRegistryOverwrite(t *testing.T) {
	reg := &ConstructorRegistry{constructors: make(map[string]Reconstructor)}

	reg.Register("validators.MaxLengthValidator", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return MaxLengthValidator{Limit: 1}, nil
	})
	reg.Register("validators.MaxLengthValidator", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return MaxLengthValidator{Limit: 999}, nil
	})

	raw, _ := json.Marshal(MaxLengthValidator{Limit: 50}, reg.JSONOptions())
	var got Deconstructible
	if err := reg.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.(MaxLengthValidator).Limit != 999 {
		t.Fatalf("expected overwritten constructor to return Limit=999, got %d", got.(MaxLengthValidator).Limit)
	}
}

// -----------------------------------------------------------------------------
// 6. Error & Edge Case Tests
// -----------------------------------------------------------------------------

type failingValue struct{}

func (failingValue) MarshalJSON() ([]byte, error) {
	return nil, errors.New("boom during marshal")
}

type brokenArgDeconstructible struct {
	arg any
}

func (b brokenArgDeconstructible) Deconstruct() (string, []any, map[string]any) {
	return "broken.Arg", []any{b.arg}, nil
}

type brokenKwargDeconstructible struct {
	val any
}

func (b brokenKwargDeconstructible) Deconstruct() (string, []any, map[string]any) {
	return "broken.Kwarg", nil, Kwargs{"bad": b.val}
}

func TestErrors(t *testing.T) {
	reg := newTestRegistry(t)

	t.Run("marshal propagates error from failing arg", func(t *testing.T) {
		_, err := json.Marshal(brokenArgDeconstructible{arg: failingValue{}}, reg.JSONOptions())
		if err == nil || !strings.Contains(err.Error(), "boom during marshal") {
			t.Fatalf("expected arg marshal error, got: %v", err)
		}
	})

	t.Run("marshal propagates error from failing kwarg", func(t *testing.T) {
		_, err := json.Marshal(brokenKwargDeconstructible{val: failingValue{}}, reg.JSONOptions())
		if err == nil || !strings.Contains(err.Error(), "boom during marshal") {
			t.Fatalf("expected kwarg marshal error, got: %v", err)
		}
	})

	t.Run("unmarshal unregistered path returns descriptive error", func(t *testing.T) {
		payload := []byte(`{"path":"unknown.Widget","args":[],"kwargs":{}}`)
		var dst Deconstructible
		err := reg.Unmarshal(payload, &dst)
		if err == nil || !strings.Contains(err.Error(), `unregistered deconstruct path: "unknown.Widget"`) {
			t.Fatalf("expected unregistered path error, got: %v", err)
		}

		// Same check through RegisterDestructibleType path
		var op Operation
		err = reg.Unmarshal(payload, &op)
		if err == nil || !strings.Contains(err.Error(), `unregistered deconstruct path: "unknown.Widget"`) {
			t.Fatalf("expected unregistered path error for Operation, got: %v", err)
		}
	})

	t.Run("unmarshal propagates reconstructor error", func(t *testing.T) {
		payload := []byte(`{"path":"validators.MaxLengthValidator","args":[],"kwargs":{}}`)

		var dst Deconstructible
		err := reg.Unmarshal(payload, &dst)
		if err == nil || !strings.Contains(err.Error(), "expected 1 arg, got 0") {
			t.Fatalf("expected reconstructor error, got: %v", err)
		}

		var val Validator
		err = reg.Unmarshal(payload, &val)
		if err == nil || !strings.Contains(err.Error(), "expected 1 arg, got 0") {
			t.Fatalf("expected reconstructor error via Validator, got: %v", err)
		}
	})

	t.Run("unmarshal invalid JSON syntax fails", func(t *testing.T) {
		var dst Deconstructible
		if err := reg.Unmarshal([]byte(`{"path":`), &dst); err == nil {
			t.Fatalf("expected syntax error on truncated JSON")
		}
	})

	t.Run("unmarshal wrong JSON root type fails", func(t *testing.T) {
		for _, bad := range []string{`123`, `"hello"`, `[1,2,3]`, `true`} {
			var dst Deconstructible
			if err := reg.Unmarshal([]byte(bad), &dst); err == nil {
				t.Errorf("expected error for JSON %s into Deconstructible, got nil", bad)
			}
			var op Operation
			if err := reg.Unmarshal([]byte(bad), &op); err == nil {
				t.Errorf("expected error for JSON %s into Operation, got nil", bad)
			}
		}
	})

	t.Run("unmarshal empty object fails with empty path", func(t *testing.T) {
		var dst Deconstructible
		err := reg.Unmarshal([]byte(`{}`), &dst)
		if err == nil || !strings.Contains(err.Error(), `unregistered deconstruct path: ""`) {
			t.Fatalf("expected empty unregistered path error, got: %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// 7. Stress, Concurrency, Fuzz & Benchmark Tests
// -----------------------------------------------------------------------------

func TestDeeplyNestedTree_50Levels(t *testing.T) {
	reg := newTestRegistry(t)

	var current Deconstructible = CascadePolicy{Name: "CASCADE"}
	for i := range 50 {
		current = &ForeignKey{
			Name:     fmt.Sprintf("level_%d", i),
			To:       "app.Node",
			OnDelete: current,
		}
	}

	raw, err := json.Marshal(current, reg.JSONOptions())
	if err != nil {
		t.Fatalf("failed to marshal 50-level tree: %v", err)
	}

	var decoded Deconstructible
	if err := reg.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal 50-level tree: %v", err)
	}

	if !reflect.DeepEqual(current, decoded) {
		t.Fatalf("50-level deep tree did not round-trip identically")
	}
}

func TestConcurrentMarshalUnmarshal(t *testing.T) {
	reg := newTestRegistry(t)

	sample := Migration{
		App:  "shop",
		Name: "0001_initial",
		Operations: []Operation{
			&AddFieldOp{
				ModelName: "Product",
				Field: &CharField{
					Name:      "sku",
					MaxLength: 64,
					Validators: []Validator{
						MaxLengthValidator{Limit: 64, Message: "too long"},
						&RangeValidator{Min: 1, Max: 64},
					},
				},
			},
			NoOp{},
		},
	}

	const goroutines = 16
	const iterations = 50

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				raw, err := json.Marshal(sample, reg.JSONOptions())
				if err != nil {
					errCh <- err
					return
				}
				var got Migration
				if err := reg.Unmarshal(raw, &got); err != nil {
					errCh <- err
					return
				}
				if !reflect.DeepEqual(sample, got) {
					errCh <- fmt.Errorf("concurrent round-trip mismatch")
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent error: %v", err)
	}
}

func FuzzDeconstructUnmarshal(f *testing.F) {
	reg := &ConstructorRegistry{constructors: make(map[string]Reconstructor)}
	reg.RegisterDestructibleType[Operation]()
	reg.Register("migrations.NoOp", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		return NoOp{}, nil
	})
	reg.Register("validators.MaxLengthValidator", func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 1 {
			return nil, errors.New("bad args")
		}
		var n int
		if err := json.Unmarshal(args[0], &n, opts...); err != nil {
			return nil, err
		}
		return MaxLengthValidator{Limit: n}, nil
	})

	f.Add([]byte(`{"path":"migrations.NoOp","args":null,"kwargs":null}`))
	f.Add([]byte(`{"path":"validators.MaxLengthValidator","args":[42],"kwargs":{"message":"hi"}}`))
	f.Add([]byte(`{"path":"unknown","args":[],"kwargs":{}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var dst Deconstructible
		_ = reg.Unmarshal(data, &dst)

		var op Operation
		_ = reg.Unmarshal(data, &op)
	})
}

func BenchmarkMarshalDeconstructible(b *testing.B) {
	reg := newTestRegistry(b)
	field := &CharField{
		Name:      "email",
		MaxLength: 254,
		Default:   "user@example.com",
		Validators: []Validator{
			MaxLengthValidator{Limit: 254, Message: "Invalid email length"},
			&RangeValidator{Min: 5, Max: 254},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := json.Marshal(field, reg.JSONOptions()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalDeconstructible(b *testing.B) {
	reg := newTestRegistry(b)
	field := &CharField{
		Name:      "email",
		MaxLength: 254,
		Default:   "user@example.com",
		Validators: []Validator{
			MaxLengthValidator{Limit: 254, Message: "Invalid email length"},
			&RangeValidator{Min: 5, Max: 254},
		},
	}
	raw, err := json.Marshal(field, reg.JSONOptions())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var dst Deconstructible
		if err := reg.Unmarshal(raw, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalMigration_Direct(b *testing.B) {
	reg := newTestRegistry(b)
	m := Migration{
		App:  "blog",
		Name: "0001_initial",
		Operations: []Operation{
			&AddFieldOp{
				ModelName: "Post",
				Field: &CharField{
					Name:      "title",
					MaxLength: 200,
					Validators: []Validator{
						MaxLengthValidator{Limit: 200},
					},
				},
			},
			&AddFieldOp{
				ModelName: "Post",
				Field: &ForeignKey{
					Name:     "author",
					To:       "auth.User",
					OnDelete: CascadePolicy{Name: "CASCADE"},
				},
			},
			NoOp{},
		},
	}
	raw, err := json.Marshal(m, reg.JSONOptions())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var dst Migration
		if err := reg.Unmarshal(raw, &dst); err != nil {
			b.Fatal(err)
		}
	}
}
