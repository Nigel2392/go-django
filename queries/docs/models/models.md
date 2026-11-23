# Defining Models

Models in `queries` are standard Go structs implementing `attrs.Definer` (`FieldDefs(ctx context.Context) attrs.Definitions`). See [Attrs](../../../docs/attrs/attrs.md) and [Models](../../../docs/models.md) for how Go-Django discovers and registers models across apps.

Embedding `models.Model` is optional, though recommended: it adds `table:"..."` tag support, automatic reverse-relation injection, proxy model wiring, annotation storage, dirty-state tracking, and lifecycle helpers (`Save`, `Create`, `Update`, `Delete`, `Load`). See [The `models.Model` Base](./model.md) for full details.

---

## Basic Model Definitions

```go
package models

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Profile struct {
	models.Model `table:"profiles"`
	ID           int
	Name         string
	Email        string
}

func (m *Profile) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{
			Primary: true,
		}),
		attrs.NewField(m, "Name", nil),
		attrs.NewField(m, "Email", nil),
	)
}

type User struct {
	models.Model
	ID      int
	Name    string
	Profile *Profile
}

func (m *User) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{
			Primary: true,
		}),
		attrs.NewField(m, "Name", nil),
		attrs.NewField(m, "Profile", &attrs.Config{
			RelForeignKey: attrs.Relate(&Profile{}, "", nil),
			Column:        "profile_id",
		}),
	).WithTableName("users")
}

type Todo struct {
	models.Model
	ID          int
	Title       string
	Description string
	Done        bool
	User        *User
}

func (m *Todo) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Title", nil),
		attrs.NewField(m, "Description", nil),
		attrs.NewField(m, "Done", nil),
		attrs.NewField(m, "User", &attrs.Config{
			RelForeignKey: attrs.Relate(&User{}, "", nil),
			Column:        "user_id",
		}),
	).WithTableName("todos")
}
```

---

## Setup for Standalone Structs (`models.Setup`)

When you construct a struct literal in Go instead of fetching it from the database or calling `attrs.NewObject(ctx, ...)`, embedded `models.Model` internals and relation values (`*RelRevFK`, `*RelM2M`, `*RelO2O`) are not yet initialized. Wrap the pointer with `models.Setup(ctx, obj)` before calling `Save`, `Load`, `State`, or `.Objects()` on relations:

```go
user := models.Setup(ctx, &User{
	Name: "Alice",
})
```

---

## Schema, Validation, and Setup Interfaces

Models can implement optional interfaces recognized by `queries` and `migrator`:

### `UniqueTogetherDefiner`

Declares multi-column unique constraints and enables `WHERE` clause / unique key generation when a model has no primary key:

```go
func (m *Membership) UniqueTogether() [][]string {
	return [][]string{
		{"User", "Organization"},
	}
}
```

### `IndexDefiner`

Declares database indexes used by `migrator`:

```go
func (m *User) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "idx_user_email", Fields: []string{"Email"}, Unique: true},
	}
}
```

### `ConstraintDefiner`

Declares `CHECK`, `UNIQUE`, or `FOREIGN KEY` constraints from `queries/src/expr/constraints` (see [Constraints](../expressions/constraints.md)):

```go
func (m *Order) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.NewCheckConstraint("check_positive_total", expr.Q("Total__gt", 0)),
	}
}
```

### `CanMigrate` and `migrator.CantMigrate`

Controls whether `migrator` generates migrations or tables for a model or field:

```go
type CanMigrate interface {
	CanMigrate() bool
}
```

Embed `migrator.CantMigrate` (or store `false` under `migrator.MetaAllowMigrateKey` on model metadata) to exclude a read-only view/model from migrations.

### `CanSetup`

Runs custom setup logic when a model is initialized by `queries` (`attrs.Make` / `models.Setup` / `QuerySet` creation):

```go
type CanSetup interface {
	Setup(ctx context.Context, value attrs.Definer) error
}
```

### `ContextValidator`

Validates a model instance before `Create`, `BulkCreate`, `Update`, or `models.Model.SaveObject`:

```go
type ContextValidator interface {
	Validate(ctx context.Context) error
}
```

---

## Primary Key / Unique Field Registration Requirement

When `queries/src` is imported, its `attrs.OnModelRegister` listener inspects every registered model:

- If the model defines a primary key field (`Primary: true`), registration succeeds normally.
- If the model has **no primary key**, it **must** define at least one unique field (`attrs.AttrUniqueKey` set to `true` in `Attributes`) or implement `UniqueTogetherDefiner` returning at least one non-empty field group. Otherwise, `attrs.OnModelRegister` returns `errors.FieldNotFound` because the ORM cannot generate row-targeting `WHERE` clauses or cache keys. (Through models without primary keys automatically use their `(SourceField, TargetField)` pair.)

---

## Next Steps

- Learn about embedded lifecycle and dirty tracking in [The `models.Model` Base](./model.md).
- Share base tables automatically across queries with [Proxy Models](./proxy_models.md).
- Load concrete polymorphic subtypes in batches with [Specific Models](./specific_models.md).
- Configure schema migrations in [Migrations](../migrations.md) or begin [Querying Objects](../querying.md).
