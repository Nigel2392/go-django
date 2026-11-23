# `attrs` Package Documentation

The `attrs` package (`github.com/Nigel2392/go-django/src/core/attrs`) provides a structured way to define, inspect, validate, and manipulate model fields at runtime, along with automatic form generation and relationship metadata.

## Table of Contents

- [Functions](./functions.md)
- [Interfaces](./interfaces.md)
- [Implementation](./implementation.md)
- [Model Meta](./model-meta.md)
- [Hooks and Signals](./hooks_signals.md)
- [Deconstruction and Serialization](../deconstruct.md)

---

## Overview

Models implement the [`Definer`](./interfaces.md#definer) interface by exposing a `FieldDefs(ctx context.Context) attrs.Definitions` method. Packages such as [`queries`](../../queries/docs/README.md), [`modelforms`](../forms/modelforms.md), and [admin](../apps/admin/models.md) build on `attrs` to map database columns, resolve relations, and generate forms.

When models are returned from a `django.AppConfig`'s `Models()` method during application startup, `go-django` automatically registers them with [`attrs.RegisterModel`](./model-meta.md#registermodel) and triggers [`attrs.ResetDefinitions`](./hooks_signals.md#available-signals). Once registered, a global registry stores each model's [`ModelMeta`](./model-meta.md#modelmeta) and static field definitions so metadata can be read without instantiating per-instance definitions.

---

## `attrs.Define` and `AttributeContext`

Always retrieve a model's runtime field definitions through `attrs.Define(ctx, model)` rather than calling `model.FieldDefs(ctx)` directly:

```go
ctx, attrCtx := attrs.AttributeContext(r.Context())
defs := attrs.Define(ctx, myModel)
```

`attrs.Make` enforces that the passed `context.Context` is an attribute context (panicking otherwise). `attrs.Define` handles two important responsibilities:

- **Safe context initialization**: If `ctx` does not already carry attribute context state, `attrs.Define` attaches `attrs.CtxFlagNone` automatically so `FieldDefs` will not panic.
- **Per-context definition caching and signals**: When `ctx` has been initialized with `attrs.AttributeContext(ctx)`, `attrs.Define` caches the returned `Definitions` keyed by the model pointer (up to 4096 entries per context) and exposes `*attrs.Context` flags and deferred signal listeners across nested calls.

For full details on context flags (`CtxFlagRegistering`, `CtxFlagDeferSignals`, `CtxFlagDeferValidation`, `CtxFlagEmbedded`) and deferred signals, see [Functions](./functions.md#attribute-context-functions).

---

## Subpackages

The `attrs` package is organized with four specialized subpackages:

- **`fattrs` (`github.com/Nigel2392/go-django/src/core/attrs/fattrs`)**: Pointer-backed (`fattrs.Field`) and reflectless (`fattrs.NewField`) field definitions that avoid struct-field reflection overhead and cache field options per model type. See [Implementation](./implementation.md#fattrs-subpackage).
- **`embedder` (`github.com/Nigel2392/go-django/src/core/attrs/embedder`)**: Automatic discovery and lifecycle hook binding (`SetupEmbedded`, `PreSaveEmbedded`, `PostSaveEmbedded`) for embedded structs and model mixins. See [Implementation](./implementation.md#embedder-subpackage).
- **`changes` (`github.com/Nigel2392/go-django/src/core/attrs/changes`)**: Produces HTML diffs (`changes.Changes`) comparing two instances of the same `attrs.Definer` model. See [Functions](./functions.md#changes-subpackage).
- **`attrutils` (`github.com/Nigel2392/go-django/src/core/attrs/attrutils`)**: Helpers for map attribute extraction, numeric casting, slice conversion, struct field/method caching, and a reflection-based fluent `Builder`. See [Functions](./functions.md#attrutils-subpackage).

---

## Related Documentation

- [Models Overview](../models.md)
- [Content Types](../contenttypes.md)
- [Queries Documentation](../../queries/docs/README.md)
