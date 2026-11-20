package expr_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"sync"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/resolver"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type constraintStubDB struct {
	drv driver.Driver
}

func (m *constraintStubDB) Unwrap() any             { return m }
func (m *constraintStubDB) Driver() driver.Driver   { return m.drv }
func (m *constraintStubDB) Ping(context.Context) error { return nil }
func (m *constraintStubDB) Close() error            { return nil }
func (m *constraintStubDB) Begin(context.Context) (drivers.Transaction, error) {
	return nil, nil
}
func (m *constraintStubDB) QueryContext(context.Context, string, ...any) (drivers.SQLRows, error) {
	return nil, nil
}
func (m *constraintStubDB) QueryRowContext(context.Context, string, ...any) drivers.SQLRow {
	return nil
}
func (m *constraintStubDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

type fkParentModel struct {
	models.Model `table:"c_test_parent"`
	ID           int64
	Code         string
}

func (m *fkParentModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Code", &attrs.Config{Column: "code"}),
	)
}

type fkChildModel struct {
	models.Model `table:"c_test_child"`
	ID           int64
	Age          int
	MinAge       int
	Score        float64
	Name         string
	Email        string
	Active       bool
	Parent       *fkParentModel
	ByCode       *fkParentModel
}

func (m *fkChildModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Age", &attrs.Config{Column: "age"}),
		attrs.NewField(m, "MinAge", &attrs.Config{Column: "min_age"}),
		attrs.NewField(m, "Score", &attrs.Config{Column: "score"}),
		attrs.NewField(m, "Name", &attrs.Config{Column: "name"}),
		attrs.NewField(m, "Email", &attrs.Config{Column: "email", Null: true}),
		attrs.NewField(m, "Active", &attrs.Config{Column: "active"}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			RelForeignKey: attrs.Relate(&fkParentModel{}, "", nil),
		}),
		attrs.NewField(m, "ByCode", &attrs.Config{
			Column:        "parent_code",
			ReverseAlias:  "ByCodeChildren",
			RelForeignKey: attrs.Relate(&fkParentModel{}, "Code", nil),
		}),
	)
}

var registerConstraintModelsOnce sync.Once

func setupDialectCompilers(t *testing.T) {
	t.Helper()
	registerConstraintModelsOnce.Do(func() {
		attrs.RegisterModel(&fkParentModel{})
		attrs.RegisterModel(&fkChildModel{})

		django.Global.Settings.Set("test_c_sqlite", &constraintStubDB{drv: &drivers.DriverSQLite{}})
		django.Global.Settings.Set("test_c_pg", &constraintStubDB{drv: &drivers.DriverPostgres{}})
		django.Global.Settings.Set("test_c_mysql", &constraintStubDB{drv: &drivers.DriverMySQL{}})
		django.Global.Settings.Set("test_c_mariadb", &constraintStubDB{drv: &drivers.DriverMariaDB{}})
	})
}

func renderConstraintSQL(t *testing.T, r *resolver.Resolver, c constraints.Constraint) string {
	t.Helper()
	var sb builder.BaseBuilder
	if err := r.ExpressionSQL(&sb, c); err != nil {
		t.Fatalf("ExpressionSQL(%s) failed: %v", c.ID(), err)
	}
	if err := sb.GetError(); err != nil {
		t.Fatalf("SQL builder error for %q: %v", c.ID(), err)
	}
	if len(sb.Vars) != 0 {
		t.Fatalf("constraint %q emitted unexpected bound vars: %v", c.ID(), sb.Vars)
	}
	return sb.String()
}

func TestConstraints_MetadataAndTypes(t *testing.T) {
	chk := constraints.CheckConstraint("chk_age", "age too low", expr.Q("Age__gte", expr.V(18, true)))
	if chk.ID() != "chk_age" || chk.ErrorMessage() != "age too low" || chk.Type() != constraints.C_CHECK {
		t.Errorf("unexpected CheckConstraint meta: id=%q msg=%q type=%v", chk.ID(), chk.ErrorMessage(), chk.Type())
	}

	uq := constraints.UniqueConstraint("uq_pair", "dup pair", "Name", "Email")
	if uq.ID() != "uq_pair" || uq.ErrorMessage() != "dup pair" || uq.Type() != constraints.C_UNIQUE {
		t.Errorf("unexpected UniqueConstraint meta: id=%q msg=%q type=%v", uq.ID(), uq.ErrorMessage(), uq.Type())
	}

	fk := constraints.ForeignKeyConstraint("fk_parent", "bad parent", "Parent", constraints.A_CASCADE, constraints.A_RESTRICT)
	if fk.ID() != "fk_parent" || fk.ErrorMessage() != "bad parent" || fk.Type() != constraints.C_FOREIGN_KEY {
		t.Errorf("unexpected ForeignKeyConstraint meta: id=%q msg=%q type=%v", fk.ID(), fk.ErrorMessage(), fk.Type())
	}
	if got := fk.Clone().(constraints.Constraint).ID(); got != "fk_parent" {
		t.Errorf("cloned fk ID = %q", got)
	}
	if got := constraints.C_PRIMARY_KEY.String(); got != "PRIMARY KEY" {
		t.Errorf("C_PRIMARY_KEY.String() = %q", got)
	}
	if got := constraints.C_INVALID.String(); got != "INVALID" {
		t.Errorf("C_INVALID.String() = %q", got)
	}
}

func TestConstraints_ResolveAcrossDialects(t *testing.T) {
	setupDialectCompilers(t)
	ctx := context.Background()

	dialects := []struct {
		name     string
		dbKey    string
		q        string
	}{
		{"SQLite", "test_c_sqlite", "`"},
		{"Postgres", "test_c_pg", `"`},
		{"MySQL", "test_c_mysql", "`"},
		{"MariaDB", "test_c_mariadb", "`"},
	}

	// Build unresolved constraints once to verify Resolve clones cleanly per dialect.
	chkGte := constraints.CheckConstraint("chk_age_gte", "too young", expr.Q("Age__gte", expr.V(18, true)))
	chkCompound := constraints.CheckConstraint(
		"chk_compound",
		"invalid",
		expr.And[expr.Expression](
			expr.Q("Age__gte", expr.Field("MinAge")),
			expr.Or(
				expr.Q("Score__gt", expr.V(0, true)),
				expr.Q("Active", expr.V(true, true)),
			),
		),
	)
	chkIsNull := constraints.CheckConstraint("chk_email_nn", "need email", expr.Q("Email__isnull", false))
	chkNot := constraints.CheckConstraint("chk_not_banned", "banned", expr.Q("Name", expr.V("banned", true)).Not(true))
	chkRaw := constraints.CheckConstraint("chk_len", "short name", expr.Raw("LENGTH(![Name]) >= ?", expr.V(3, true)))
	chkUnnamed := constraints.CheckConstraint("", "", expr.Q("Age__lt", expr.V(200, true)))

	uqSingle := constraints.UniqueConstraint("uq_name", "dup name", "Name")
	uqMulti := constraints.UniqueConstraint("uq_name_email", "dup combo", []string{"Name", "Email"})
	uqExpr := constraints.UniqueConstraint("uq_lower_email", "dup lower email", expr.LOWER("Email"))
	uqUnnamed := constraints.UniqueConstraint("", "", "Email")

	fkPrimary := constraints.ForeignKeyConstraint("fk_parent", "missing parent", "Parent", constraints.A_CASCADE, constraints.A_SET_NULL)
	fkCustomCol := constraints.ForeignKeyConstraint("fk_by_code", "missing code", "ByCode", constraints.A_RESTRICT, constraints.A_SET_DEFAULT)
	fkUnnamed := constraints.ForeignKeyConstraint("", "", "Parent", constraints.A_CASCADE, constraints.A_CASCADE)

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			comp := queries.Compiler(d.dbKey)
			res := resolver.New(ctx, &fkChildModel{}, comp)
			col := func(name string) string { return d.q + name + d.q }

			gotGte := renderConstraintSQL(t, res, chkGte)
			wantGte := "CONSTRAINT " + col("chk_age_gte") + " CHECK (" + col("age") + " >= 18)"
			if gotGte != wantGte {
				t.Errorf("chkGte:\n  got:  %s\n  want: %s", gotGte, wantGte)
			}

			gotComp := renderConstraintSQL(t, res, chkCompound)
			wantComp := "CONSTRAINT " + col("chk_compound") + " CHECK ((" + col("age") + " >= " + col("min_age") + " AND (" + col("score") + " > 0 OR " + col("active") + " = 1)))"
			if gotComp != wantComp {
				t.Errorf("chkCompound:\n  got:  %s\n  want: %s", gotComp, wantComp)
			}

			gotNull := renderConstraintSQL(t, res, chkIsNull)
			wantNull := "CONSTRAINT " + col("chk_email_nn") + " CHECK (" + col("email") + " IS NOT NULL)"
			if gotNull != wantNull {
				t.Errorf("chkIsNull:\n  got:  %s\n  want: %s", gotNull, wantNull)
			}

			gotNot := renderConstraintSQL(t, res, chkNot)
			wantNot := "CONSTRAINT " + col("chk_not_banned") + " CHECK (NOT (" + col("name") + " = 'banned'))"
			if gotNot != wantNot {
				t.Errorf("chkNot:\n  got:  %s\n  want: %s", gotNot, wantNot)
			}

			gotRaw := renderConstraintSQL(t, res, chkRaw)
			wantRaw := "CONSTRAINT " + col("chk_len") + " CHECK (LENGTH(" + col("name") + ") >= 3)"
			if gotRaw != wantRaw {
				t.Errorf("chkRaw:\n  got:  %s\n  want: %s", gotRaw, wantRaw)
			}

			gotUnnamedChk := renderConstraintSQL(t, res, chkUnnamed)
			wantUnnamedChk := "CHECK (" + col("age") + " < 200)"
			if gotUnnamedChk != wantUnnamedChk {
				t.Errorf("chkUnnamed:\n  got:  %s\n  want: %s", gotUnnamedChk, wantUnnamedChk)
			}

			gotUq1 := renderConstraintSQL(t, res, uqSingle)
			wantUq1 := "CONSTRAINT " + col("uq_name") + " UNIQUE (" + col("name") + ")"
			if gotUq1 != wantUq1 {
				t.Errorf("uqSingle:\n  got:  %s\n  want: %s", gotUq1, wantUq1)
			}

			gotUq2 := renderConstraintSQL(t, res, uqMulti)
			wantUq2 := "CONSTRAINT " + col("uq_name_email") + " UNIQUE (" + col("name") + ", " + col("email") + ")"
			if gotUq2 != wantUq2 {
				t.Errorf("uqMulti:\n  got:  %s\n  want: %s", gotUq2, wantUq2)
			}

			gotUqExpr := renderConstraintSQL(t, res, uqExpr)
			wantUqExpr := "CONSTRAINT " + col("uq_lower_email") + " UNIQUE (LOWER(" + col("email") + "))"
			if gotUqExpr != wantUqExpr {
				t.Errorf("uqExpr:\n  got:  %s\n  want: %s", gotUqExpr, wantUqExpr)
			}

			gotUnnamedUq := renderConstraintSQL(t, res, uqUnnamed)
			wantUnnamedUq := "UNIQUE (" + col("email") + ")"
			if gotUnnamedUq != wantUnnamedUq {
				t.Errorf("uqUnnamed:\n  got:  %s\n  want: %s", gotUnnamedUq, wantUnnamedUq)
			}

			gotFK1 := renderConstraintSQL(t, res, fkPrimary)
			wantFK1 := "CONSTRAINT " + col("fk_parent") + " FOREIGN KEY (" + col("parent_id") + ") REFERENCES " + col("c_test_parent") + " (" + col("id") + ") ON DELETE SET NULL ON UPDATE CASCADE"
			if gotFK1 != wantFK1 {
				t.Errorf("fkPrimary:\n  got:  %s\n  want: %s", gotFK1, wantFK1)
			}

			gotFK2 := renderConstraintSQL(t, res, fkCustomCol)
			wantFK2 := "CONSTRAINT " + col("fk_by_code") + " FOREIGN KEY (" + col("parent_code") + ") REFERENCES " + col("c_test_parent") + " (" + col("code") + ") ON DELETE SET DEFAULT ON UPDATE RESTRICT"
			if gotFK2 != wantFK2 {
				t.Errorf("fkCustomCol:\n  got:  %s\n  want: %s", gotFK2, wantFK2)
			}

			gotFKUnnamed := renderConstraintSQL(t, res, fkUnnamed)
			wantFKUnnamed := "FOREIGN KEY (" + col("parent_id") + ") REFERENCES " + col("c_test_parent") + " (" + col("id") + ") ON DELETE CASCADE ON UPDATE CASCADE"
			if gotFKUnnamed != wantFKUnnamed {
				t.Errorf("fkUnnamed:\n  got:  %s\n  want: %s", gotFKUnnamed, wantFKUnnamed)
			}
		})
	}
}

func TestConstraints_ErrorCases(t *testing.T) {
	setupDialectCompilers(t)
	ctx := context.Background()
	res := resolver.New(ctx, &fkChildModel{}, queries.Compiler(django.APPVAR_DATABASE))

	assertSQLErr := func(name string, c constraints.Constraint, substr string) {
		t.Helper()
		var sb builder.BaseBuilder
		c.SQL(&sb)
		if err := sb.GetError(); err == nil || !strings.Contains(err.Error(), substr) {
			t.Errorf("%s: expected error containing %q, got %v", name, substr, err)
		}
	}

	// Unresolved constraints must error on SQL()
	assertSQLErr("unresolved_check", constraints.CheckConstraint("c1", "", expr.Q("Age__gt", expr.V(0, true))), "must be resolved")
	assertSQLErr("unresolved_unique", constraints.UniqueConstraint("u1", "", "Name"), "must be resolved")
	assertSQLErr("unresolved_fk", constraints.ForeignKeyConstraint("f1", "", "Parent", constraints.A_CASCADE, constraints.A_CASCADE), "must be resolved")

	// Nil statement / empty fields / missing resolver
	assertSQLErr("nil_stmt", constraints.CheckConstraint("c_nil", "", nil).Resolve(res.ExpressionInfo()).(constraints.Constraint), "nil statement")
	assertSQLErr("empty_unique", constraints.UniqueConstraint("u_empty", "").Resolve(res.ExpressionInfo()).(constraints.Constraint), "no fields")
	assertSQLErr("fk_no_resolver", constraints.ForeignKeyConstraint("f_nores", "", "Parent", constraints.A_CASCADE, constraints.A_CASCADE).Resolve(&expr.ExpressionInfo{}).(constraints.Constraint), "Resolver")
	assertSQLErr("fk_non_rel_field", constraints.ForeignKeyConstraint("f_nonrel", "", "Name", constraints.A_CASCADE, constraints.A_CASCADE).Resolve(res.ExpressionInfo()).(constraints.Constraint), "no relation")
	assertSQLErr("fk_missing_field", constraints.ForeignKeyConstraint("f_missing", "", "DoesNotExist", constraints.A_CASCADE, constraints.A_CASCADE).Resolve(res.ExpressionInfo()).(constraints.Constraint), "resolve field")

	// Normal resolved FK retains OnUpdate and OnDelete through Clone()
	fkOrig := constraints.ForeignKeyConstraint("fk_clone", "msg", "Parent", constraints.A_SET_DEFAULT, constraints.A_CASCADE)
	fkResolved := res.ResolveExpression(fkOrig).(constraints.Constraint)
	var sb builder.BaseBuilder
	fkResolved.SQL(&sb)
	if err := sb.GetError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sb.String(), "ON DELETE CASCADE ON UPDATE SET DEFAULT") {
		t.Errorf("Clone/Resolve lost mutation actions: %s", sb.String())
	}
	// Resolve idempotency on already-resolved constraint returns itself
	if same := fkResolved.Resolve(res.ExpressionInfo()); same != fkResolved {
		t.Errorf("expected Resolve on already-resolved constraint to be idempotent")
	}
}
