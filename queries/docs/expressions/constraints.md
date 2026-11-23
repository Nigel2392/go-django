# Constraints

The `github.com/Nigel2392/go-django/queries/src/expr/constraints` package defines declarative database constraints (`CHECK`, `UNIQUE`, and `FOREIGN KEY`) that models expose to the migration system through `migrator.ConstraintDefiner`.

---

## Core Interfaces and Types

```go
package constraints

type ConstraintType uint

const (
	C_INVALID ConstraintType = iota
	C_PRIMARY_KEY
	C_UNIQUE
	C_FOREIGN_KEY
	C_CHECK
)

type MutationAction int

const (
	A_RESTRICT MutationAction = iota
	A_CASCADE
	A_SET_NULL
	A_SET_DEFAULT
)

type Constraint interface {
	expr.Expression
	ID() string
	Type() ConstraintType
	ErrorMessage() string
}

type Base struct {
	Identifier string
	Error      string
}
```

- `ConstraintType.String()` formats to `"PRIMARY KEY"`, `"UNIQUE"`, `"FOREIGN KEY"`, `"CHECK"`, or `"INVALID"`.
- `MutationAction.String()` formats to `"RESTRICT"`, `"CASCADE"`, `"SET NULL"`, `"SET DEFAULT"`, or `"<ACTION_UNKNOWN>"`.
- `Constraint` extends `expr.Expression` ([Expressions](./expressions.md)), so every constraint supports `Clone()`, `Resolve(inf)`, and `SQL(sb)`.

---

## Built-In Constraint Constructors

### `CheckConstraint`

```go
func CheckConstraint(id string, errorMessage string, statement expr.Expression) Constraint
```

Generates `[CONSTRAINT "id" ]CHECK (<statement>)`. During `Resolve`, `ExpressionInfo` sets `ForUpdate: true`, `SupportsWhereAlias: false`, and `SupportsAsExpr: false`.

### `UniqueConstraint`

```go
func UniqueConstraint(id string, errorMessage string, fields ...any) Constraint
```

Generates `[CONSTRAINT "id" ]UNIQUE (<field1>, <field2>, ...)`. `fields` are converted via `expr.ExpressionFromInterface[expr.Expression](fields, false)`, so each item can be a field name (`string`), an `attrs.FieldDefinition`, or an `expr.Expression`.

### `ForeignKeyConstraint`

```go
func ForeignKeyConstraint(id string, errorMessage string, field string, onUpdate, onDelete MutationAction) Constraint
```

Generates `[CONSTRAINT "id" ]FOREIGN KEY ("col") REFERENCES "target_table" ("target_col") ON DELETE <action> ON UPDATE <action>`. During `Resolve`, it inspects the named field's `attrs.Relation` (`fieldDef.Rel()`) via `inf.Resolver` to resolve the target table and target column (falling back to the target model's primary key column).

---

## Attaching Constraints to Models

Implement `queries.ConstraintDefiner` (an alias for `migrator.ConstraintDefiner`) on your model struct:

```go
package models

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Org struct {
	ID   int64
	Name string
}

func (o *Org) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, o,
		attrs.NewField(o, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(o, "Name", &attrs.Config{MaxLength: 80}),
	)
}

func (o *Org) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.UniqueConstraint("uq_org_name", "org name already taken", "Name"),
	}
}

type Member struct {
	ID    int64
	Age   int64
	Email string
	Org   *Org
}

func (m *Member) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Age", &attrs.Config{}),
		attrs.NewField(m, "Email", &attrs.Config{MaxLength: 120}),
		attrs.NewField(m, "Org", &attrs.Config{
			Column:        "org_id",
			RelForeignKey: attrs.Relate(&Org{}, "", nil),
		}),
	)
}

func (m *Member) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.UniqueConstraint("uq_member_org_email", "duplicate email in org", "Org", "Email"),
		constraints.ForeignKeyConstraint("fk_member_org", "invalid org", "Org", constraints.A_CASCADE, constraints.A_CASCADE),
		constraints.CheckConstraint(
			"chk_member_age",
			"member must be 18 or older",
			expr.Q("Age__gte", expr.V(18, true)),
		),
	}
}
```

When generating table schemas, `migrator` also automatically creates a `CheckConstraint` named `constraint_<field>_options` for any column that defines `Options` (`attrs.Option`), and `SQLite` / `PostgreSQL` schema editors suppress duplicate inline `REFERENCES` clauses on `CREATE TABLE` when a `ForeignKeyConstraint` targets the same column.

---

## Deconstruction

`CheckConstraint`, `UniqueConstraint`, and `ForeignKeyConstraint` implement `deconstruct.Deconstructible` (`"queries.expr.constraints.Check"`, `"queries.expr.constraints.Unique"`, `"queries.expr.constraints.ForeignKey"`) so `.mig` migration files can serialize and replay constraint changes across schema diffs. See [Deconstruction](../../../docs/deconstruct.md) and [Migrations](../migrations.md).

---

## Caveats

- **No bound parameters inside DDL `CHECK` constraints:** Schema editors across SQLite, MySQL/MariaDB, and PostgreSQL reject any constraint whose SQL produces bound variables (`len(sb.Vars) > 0`). Inside `CheckConstraint`, pass `unsafe = true` literals (`expr.Value(val, true)` or `expr.V(val, true)`) so values inline directly into DDL.
- **Non-empty `ID` required to drop constraints:** `DropConstraint` requires a non-empty `constraint.ID()` across drivers (other than `C_PRIMARY_KEY` on MySQL). Always give every constraint a stable, unique identifier.
- **SQLite constraint changes rebuild tables:** SQLite does not support adding or dropping arbitrary named constraints in-place via `ALTER TABLE`, so `SQLiteSchemaEditor.AddConstraint` and `DropConstraint` recreate the table when applying constraint changes.
- **Resolution requirements:** Constraints must be resolved via `Resolve(inf)` with a non-nil `ExpressionInfo` and `QuoteIdentifier` before calling `SQL()`, and `ForeignKeyConstraint` additionally requires a non-nil `inf.Resolver` (provided during migrations via `migrator.CompilerForModel`).
