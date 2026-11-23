# Hooks and Signals for the `attrs` Package

The `attrs` package exposes [signals](https://github.com/Nigel2392/go-signals) for model registration lifecycle events and [goldcrest](https://github.com/Nigel2392/goldcrest) hooks for customizing form field generation. See [Hooks](../hooks.md) for general project hook conventions.

---

## Signals

All registration signals are defined in `src/core/attrs/signals.go` using `github.com/Nigel2392/go-signals`. Listeners take `(ctx context.Context, s signals.Signal[T], data T) error`.

### Signal Payloads

```go
type SignalInitMeta struct {
	Definer Definer
	Meta    ModelMeta
}

type SignalModelMeta struct {
	SignalInitMeta
	Definitions StaticDefinitions
}

type SignalThroughModelMeta struct {
	Source      Definer
	Target      Definer
	ThroughInfo Through
	Meta        ModelMeta
}
```

### Available Signals

- `InitModelRegister` (`signals.Signal[SignalInitMeta]`): Sent in `registerModel` before `FieldDefs` is called for a model.
- `OnBeforeModelRegister` (`signals.Signal[SignalModelMeta]`): Sent after `FieldDefs` and `StaticDefinitions` are constructed, before processing field registrations and relations (only when `sendSignals` is `true`, i.e., during initial `RegisterModel`).
- `OnModelRegister` (`signals.Signal[SignalModelMeta]`): Sent at the end of initial `RegisterModel` once field attributes and relations are stored.
- `OnThroughModelRegister` (`signals.Signal[SignalThroughModelMeta]`): Sent when a relation's `Through` model is encountered on the forward relation side.
- `ResetDefinitions` (`signals.Signal[any]`): Triggers a topological dependency rebuild across all registered models, calls `CanOnModelRegister.OnModelRegister` once per model type, and rebuilds `StaticDefinitions` so reverse relation fields are visible.
- `PreResetDefinitions` (`signals.Signal[any]`): Sent at the start of the `ResetDefinitions` listener before rebuilding the dependency graph.
- `PostResetDefinitions` (`signals.Signal[any]`): Sent at the end of the `ResetDefinitions` listener after all static definitions are rebuilt.

### Example Signal Usage

Register signal listeners in an `init()` function or application setup:

```go
package myapp

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
)

func init() {
	attrs.InitModelRegister.Listen(context.Background(), func(ctx context.Context, s signals.Signal[attrs.SignalInitMeta], data attrs.SignalInitMeta) error {
		// Runs before FieldDefs is evaluated on data.Definer
		return nil
	})

	attrs.OnModelRegister.Listen(context.Background(), func(ctx context.Context, s signals.Signal[attrs.SignalModelMeta], data attrs.SignalModelMeta) error {
		// Runs after initial registration; data.Definitions and data.Meta are available
		return nil
	})

	attrs.PostResetDefinitions.Listen(context.Background(), func(ctx context.Context, s signals.Signal[any], _ any) error {
		// Runs after all models and reverse relations have been finalized
		return nil
	})
}
```

---

## Hooks

`attrs` defines two `goldcrest` hook names and function types in `src/core/attrs/signals.go`:

```go
const (
	HookFormFieldForType = "attrs.FormFieldForType"
	DefaultForType       = "attrs.DefaultForType"
)

type FormFieldGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)
type DefaultGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool)
```

### Form Field Hooks

#### `RegisterFormFieldType`

Registers a simple form field constructor for a Go type (accepting a sample value, `reflect.Type`, or `reflect.Value`):

```go
func RegisterFormFieldType(valueOfType any, getField func(opts ...func(fields.Field)) fields.Field)
```

Example:

```go
attrs.RegisterFormFieldType(
	json.RawMessage{},
	func(opts ...func(fields.Field)) fields.Field {
		return fields.JSONField[json.RawMessage](opts...)
	},
)
```

#### `RegisterFormFieldGetter`

Registers a full `FormFieldGetter` callback when the generated `fields.Field` depends on the `attrs.Field` metadata (such as attributes or field value):

```go
func RegisterFormFieldGetter(valueOfType any, getField FormFieldGetter)
```

For example, the `filefield` subpackage uses `RegisterFormFieldGetter` on `mediafiles.SimpleStoredObject{}` to read `attrs.AttrUploadToKey` from `f.Attrs()` and configure `UploadTo`.

Both helpers register into an internal type map consulted by a single `goldcrest` listener on `HookFormFieldForType` (checking `field_v.Type()` first, then `new_field_t_indirected`). You can also register directly against `HookFormFieldForType` via `goldcrest.Register` with a custom priority.

---

### `RegisterDefaultType` (Default Value Hook)

`RegisterDefaultType` registers a `DefaultGetter` on the `DefaultForType` hook for a given type:

```go
func RegisterDefaultType(
	valueOfType any,
	getDefault func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool),
)
```

**Caveat**: `FieldDef.GetDefault()` in `src/core/attrs/fields.go` currently returns `nil` early whenever `f.Config.Default == nil` (right after optional `ALLOW_METHOD_CHECKS` checks). As a result, the `goldcrest.Get[DefaultGetter](DefaultForType)` loop inside `FieldDef.GetDefault()` is unreachable dead code. To provide a default value for a field, set `Config.Default` directly on the field definition or via the `attrs:"default=..."` struct tag.
