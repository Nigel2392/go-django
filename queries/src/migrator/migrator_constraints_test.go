package migrator_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/mysql"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/postgres"
	testsql "github.com/Nigel2392/go-django/queries/src/migrator/sql/test_sql"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/deconstruct"
)

// --- Models & evolving state for TestMigrator_Constraints_FullLifecycle_LiveSQLite ---

var conEvoState struct {
	addRoleConstraint bool
	minAge            int64
	addScoreField     bool
	addScoreCheck     bool
	alterRoleCol      bool
	dropChecks        bool
	dropScoreField    bool
	renameTable       bool
}

func resetConEvoState() {
	conEvoState.addRoleConstraint = false
	conEvoState.minAge = 18
	conEvoState.addScoreField = false
	conEvoState.addScoreCheck = false
	conEvoState.alterRoleCol = false
	conEvoState.dropChecks = false
	conEvoState.dropScoreField = false
	conEvoState.renameTable = false
}

type MigConOrg struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *MigConOrg) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.UniqueConstraint("uq_con_org_name", "org name taken", "Name"),
	}
}

func (m *MigConOrg) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name", &attrs.Config{MaxLength: 80}),
	).WithTableName("mig_con_org")
}

type MigConMember struct {
	ID    int64 `attrs:"primary"`
	Age   int64
	Email string
	Role  string
	Score int64
	Org   *MigConOrg
}

func (m *MigConMember) Constraints() []constraints.Constraint {
	cs := []constraints.Constraint{
		constraints.UniqueConstraint("uq_con_member_org_email", "duplicate email per org", "Org", "Email"),
		constraints.ForeignKeyConstraint("fk_con_member_org", "invalid org", "Org", constraints.A_CASCADE, constraints.A_CASCADE),
	}
	if !conEvoState.dropChecks {
		cs = append(cs, constraints.CheckConstraint(
			"chk_con_member_age",
			"member too young",
			expr.Q("Age__gte", expr.V(conEvoState.minAge, true)),
		))
		if conEvoState.addRoleConstraint {
			cs = append(cs, constraints.CheckConstraint(
				"chk_con_member_role",
				"invalid role",
				expr.Raw("![Role] IN (?, ?)", expr.V("user", true), expr.V("admin", true)),
			))
		}
	}
	if conEvoState.addScoreCheck {
		cs = append(cs, constraints.CheckConstraint(
			"chk_con_member_score",
			"negative score",
			expr.Q("Score__gte", expr.V(0, true)),
		))
	}
	return cs
}

func (m *MigConMember) FieldDefs(ctx context.Context) attrs.Definitions {
	roleCfg := attrs.Config{MaxLength: 20, Default: "user"}
	if conEvoState.alterRoleCol {
		roleCfg.MaxLength = 40
		roleCfg.Default = "admin"
	}
	flds := []attrs.Field{
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Age", &attrs.Config{}),
		attrs.NewField(m, "Email", &attrs.Config{MaxLength: 120}),
		attrs.NewField(m, "Role", &roleCfg),
		attrs.NewField(m, "Org", &attrs.Config{
			Column:        "org_id",
			RelForeignKey: attrs.Relate(&MigConOrg{}, "", nil),
		}),
	}
	if conEvoState.addScoreField && !conEvoState.dropScoreField {
		flds = append(flds, attrs.NewField(m, "Score", &attrs.Config{Default: int64(5)}))
	}
	tbl := "mig_con_member"
	if conEvoState.renameTable {
		tbl = "mig_con_member_v2"
	}
	return attrs.Make(ctx, m, flds...).WithTableName(tbl)
}

func TestModelTable_Constraints_DefinitionAndJSONRoundTrip(t *testing.T) {
	resetConEvoState()
	conEvoState.addRoleConstraint = true
	defer resetConEvoState()

	db, _ := openTestSQLite(t)
	registerModelsContentTypes(&MigConOrg{}, &MigConMember{})
	initMigTestApp(t, db, func() django.AppConfig {
		app := apps.NewDBAppConfig("con_rt_app")
		app.ModelObjects = []attrs.Definer{&MigConOrg{}, &MigConMember{}}
		return app
	})

	tbl := migrator.NewModelTable(&MigConMember{})
	var ifaceTbl migrator.Table = tbl
	cs := ifaceTbl.Constraints()
	if len(cs) != 4 {
		t.Fatalf("expected 4 constraints on MigConMember, got %d", len(cs))
	}

	raw, err := json.Marshal(tbl, deconstruct.Constructors.JSONOptions())
	if err != nil {
		t.Fatalf("Marshal ModelTable failed: %v", err)
	}
	var rt migrator.ModelTable
	if err := json.Unmarshal(raw, &rt, deconstruct.Constructors.JSONOptions()); err != nil {
		t.Fatalf("Unmarshal ModelTable failed: %v", err)
	}

	rtCs := rt.Constraints()
	if len(rtCs) != len(cs) {
		t.Fatalf("round-trip constraint count mismatch: got %d want %d", len(rtCs), len(cs))
	}
	for i := range cs {
		b1, _ := json.Marshal(cs[i], deconstruct.Constructors.JSONOptions(), json.Deterministic(true))
		b2, _ := json.Marshal(rtCs[i], deconstruct.Constructors.JSONOptions(), json.Deterministic(true))
		if string(b1) != string(b2) {
			t.Errorf("constraint %d round-trip mismatch:\nwant: %s\ngot:  %s", i, string(b1), string(b2))
		}
	}

	// Verify NeedsToMakeMigrations reports zero changes once initial migration exists
	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, testsql.NewTestMigrationEngine(t))
	if err := engine.MakeMigrations(context.Background()); err != nil {
		t.Fatalf("MakeMigrations: %v", err)
	}
	pending, err := engine.NeedsToMakeMigrations(context.Background())
	if err != nil {
		t.Fatalf("NeedsToMakeMigrations: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending migrations after JSON round-trip, got %d", len(pending))
	}
	if err := engine.MakeMigrations(context.Background()); err != migrator.ErrNoChanges {
		t.Errorf("expected ErrNoChanges on second MakeMigrations, got %v", err)
	}
}

func TestMigrator_Constraints_FullLifecycle_LiveSQLite(t *testing.T) {
	resetConEvoState()
	ctx := context.Background()
	defer func() {
		resetConEvoState()
		_ = attrs.ResetDefinitions.Send(ctx, nil)
	}()
	db, ed := openTestSQLite(t)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("enable foreign_keys: %v", err)
	}

	registerModelsContentTypes(&MigConOrg{}, &MigConMember{})
	initMigTestApp(t, db, func() django.AppConfig {
		app := apps.NewDBAppConfig("con_evo_app")
		app.ModelObjects = []attrs.Definer{&MigConOrg{}, &MigConMember{}}
		return app
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, ed)

	// Step 1: Initial CreateTable + Constraints
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_org (id, name) VALUES (1, 'Acme')"); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_org (id, name) VALUES (2, 'Acme')"); err == nil {
		t.Errorf("expected UNIQUE uq_con_org_name to reject duplicate 'Acme'")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (1, 16, 'a@x.com', 'user', 1)"); err == nil {
		t.Errorf("expected CHECK chk_con_member_age to reject age=16")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (1, 25, 'a@x.com', 'user', 999)"); err == nil {
		t.Errorf("expected FK fk_con_member_org to reject org_id=999")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (1, 25, 'a@x.com', 'user', 1)"); err != nil {
		t.Fatalf("insert valid member: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (2, 25, 'a@x.com', 'user', 1)"); err == nil {
		t.Errorf("expected composite UNIQUE uq_con_member_org_email to reject duplicate (1, a@x.com)")
	}

	// Step 2: Add chk_con_member_role
	conEvoState.addRoleConstraint = true
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 2 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 2 Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (2, 25, 'b@x.com', 'guest', 1)"); err == nil {
		t.Errorf("expected chk_con_member_role to reject role='guest'")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (2, 25, 'b@x.com', 'admin', 1)"); err != nil {
		t.Fatalf("insert valid admin role: %v", err)
	}

	// Step 3: Modify chk_con_member_age from >= 18 to >= 21 -> emits DropConstraint + AddConstraint
	conEvoState.minAge = 21
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 3 MakeMigrations: %v", err)
	}
	migs, err := engine.ReadMigrations("con_evo_app")
	if err != nil {
		t.Fatalf("ReadMigrations step 3: %v", err)
	}
	lastMemberMig := migs[len(migs)-1]
	for _, m := range migs {
		if m.ModelName == "MigConMember" {
			lastMemberMig = m
		}
	}
	if len(lastMemberMig.Actions) != 2 ||
		lastMemberMig.Actions[0].ActionType != migrator.ActionDropConstraint ||
		lastMemberMig.Actions[1].ActionType != migrator.ActionAddConstraint {
		t.Fatalf("expected [DropConstraint, AddConstraint] for modified CHECK, got %+v", lastMemberMig.Actions)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 3 Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (3, 19, 'c@x.com', 'user', 1)"); err == nil {
		t.Errorf("expected updated chk_con_member_age (>= 21) to reject age=19")
	}

	// Step 4: Add Score field + chk_con_member_score in same migration
	conEvoState.addScoreField = true
	conEvoState.addScoreCheck = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 4 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 4 Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id, score) VALUES (3, 25, 'c@x.com', 'user', 1, -5)"); err == nil {
		t.Errorf("expected chk_con_member_score to reject score=-5")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id, score) VALUES (3, 25, 'c@x.com', 'user', 1, 10)"); err != nil {
		t.Fatalf("insert valid score=10: %v", err)
	}

	// Step 5: AlterField on Role while keeping all constraints active
	conEvoState.alterRoleCol = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 5 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 5 Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id, score) VALUES (4, 19, 'd@x.com', 'user', 1, 1)"); err == nil {
		t.Errorf("expected chk_con_member_age still enforced after AlterField")
	}

	// Step 6: Replay migrations 0001..0005 onto a brand-new empty SQLite database
	freshDB, freshEd := openTestSQLite(t)
	if _, err := freshDB.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("fresh enable foreign_keys: %v", err)
	}
	freshEngine := migrator.NewMigrationEngine(tmpDir, freshEd)
	if err := freshEngine.Migrate(ctx); err != nil {
		t.Fatalf("Fresh replay Migrate(0001..0005): %v", err)
	}
	if _, err := freshDB.ExecContext(ctx, "INSERT INTO mig_con_org (id, name) VALUES (1, 'ReplayOrg')"); err != nil {
		t.Fatalf("fresh insert org: %v", err)
	}
	if _, err := freshDB.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id, score) VALUES (1, 19, 'x@x.com', 'user', 1, 5)"); err == nil {
		t.Errorf("expected replayed chk_con_member_age (>= 21) to reject age=19")
	}
	if _, err := freshDB.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id, score) VALUES (1, 30, 'x@x.com', 'user', 1, 5)"); err != nil {
		t.Fatalf("fresh valid insert failed: %v", err)
	}

	// Step 7: Drop chk_con_member_score AND remove Score field in same migration (DropConstraint must precede RemoveField)
	conEvoState.addScoreCheck = false
	conEvoState.dropScoreField = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 7 MakeMigrations: %v", err)
	}
	migs, _ = engine.ReadMigrations("con_evo_app")
	var dropStep *migrator.MigrationFile
	for _, m := range migs {
		if m.ModelName == "MigConMember" {
			dropStep = m
		}
	}
	if len(dropStep.Actions) < 2 ||
		dropStep.Actions[0].ActionType != migrator.ActionDropConstraint ||
		dropStep.Actions[1].ActionType != migrator.ActionRemoveField {
		t.Fatalf("expected ActionDropConstraint before ActionRemoveField, got %+v", dropStep.Actions)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 7 Migrate: %v", err)
	}

	// Step 8: Drop remaining CHECK constraints -> inserts with low age and custom role now succeed
	conEvoState.dropChecks = true
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 8 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 8 Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (5, 5, 'e@x.com', 'custom', 1)"); err != nil {
		t.Fatalf("insert after dropping CHECK constraints failed: %v", err)
	}
	// Unique & FK still active
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (6, 5, 'e@x.com', 'custom', 1)"); err == nil {
		t.Errorf("expected composite UNIQUE uq_con_member_org_email still enforced")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (6, 5, 'f@x.com', 'custom', 999)"); err == nil {
		t.Errorf("expected FK fk_con_member_org still enforced")
	}
}

func TestMigrator_Constraints_RollbackOnViolatingExistingData(t *testing.T) {
	resetConEvoState()
	conEvoState.dropChecks = true // Start without CHECKs
	defer resetConEvoState()

	ctx := context.Background()
	db, ed := openTestSQLite(t)
	registerModelsContentTypes(&MigConOrg{}, &MigConMember{})
	initMigTestApp(t, db, func() django.AppConfig {
		app := apps.NewDBAppConfig("con_rb_app")
		app.ModelObjects = []attrs.Definer{&MigConOrg{}, &MigConMember{}}
		return app
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, ed)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations 1: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Migrate 1: %v", err)
	}

	// Insert violating row (role = 'invalid_role', age = 10)
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_org (id, name) VALUES (1, 'Org1')"); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (1, 10, 'r@x.com', 'invalid_role', 1)"); err != nil {
		t.Fatalf("insert row: %v", err)
	}

	// Now generate migration adding CHECK (age >= 18) and CHECK (role IN ...)
	conEvoState.dropChecks = false
	conEvoState.minAge = 18
	conEvoState.addRoleConstraint = true
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations 2: %v", err)
	}

	if err := engine.Migrate(ctx); err == nil {
		t.Fatalf("expected Migrate 2 to fail because existing row violates CHECK constraint")
	}

	// Migration transaction must have rolled back: row still exists, migration NOT recorded, and new inserts with age=10 still succeed
	var migCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM migrations WHERE app_name='con_rb_app' AND model_name='MigConMember'").Scan(&migCount); err != nil || migCount != 1 {
		t.Errorf("expected only 1 recorded migration after rollback, got %d (err=%v)", migCount, err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member (id, age, email, role, org_id) VALUES (2, 10, 'r2@x.com', 'invalid_role', 1)"); err != nil {
		t.Errorf("expected schema unchanged after rollback, got insert error: %v", err)
	}
}

func TestMigrator_Constraints_TableRenameWithConstraints_LiveSQLite(t *testing.T) {
	resetConEvoState()
	ctx := context.Background()
	defer func() {
		resetConEvoState()
		_ = attrs.ResetDefinitions.Send(ctx, nil)
	}()
	db, ed := openTestSQLite(t)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("enable fk: %v", err)
	}

	registerModelsContentTypes(&MigConOrg{}, &MigConMember{})
	initMigTestApp(t, db, func() django.AppConfig {
		app := apps.NewDBAppConfig("con_ren_app")
		app.ModelObjects = []attrs.Definer{&MigConOrg{}, &MigConMember{}}
		return app
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, ed)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations 1: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Migrate 1: %v", err)
	}

	// Rename table AND add role constraint simultaneously
	conEvoState.renameTable = true
	conEvoState.addRoleConstraint = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations 2: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Migrate 2: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_org (id, name) VALUES (1, 'OrgR')"); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member_v2 (id, age, email, role, org_id) VALUES (1, 10, 'a@r.com', 'user', 1)"); err == nil {
		t.Errorf("expected preserved chk_con_member_age on renamed table")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member_v2 (id, age, email, role, org_id) VALUES (1, 25, 'a@r.com', 'bad', 1)"); err == nil {
		t.Errorf("expected added chk_con_member_role on renamed table")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_member_v2 (id, age, email, role, org_id) VALUES (1, 25, 'a@r.com', 'admin', 1)"); err != nil {
		t.Fatalf("valid insert on renamed table failed: %v", err)
	}
}

type migComplexCheckModel struct {
	ID       int64 `attrs:"primary"`
	Min      int64
	Max      int64
	Tag      string
	Name     string
	Code     string
	Optional string
}

func (m *migComplexCheckModel) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.CheckConstraint("chk_min_lte_max", "", expr.Q("Min__lte", expr.Field("Max"))),
		constraints.CheckConstraint("chk_name_prefix", "", expr.Q("Name__startswith", expr.V("ok-", true))),
		constraints.CheckConstraint("chk_tag_or", "", expr.Or(
			expr.Q("Tag", expr.V("alpha", true)),
			expr.Q("Tag", expr.V("beta", true)),
		)),
		constraints.CheckConstraint("chk_code_len", "", expr.Raw("LENGTH(![Code]) >= ?", expr.V(4, true))),
		constraints.CheckConstraint("chk_opt_nn", "", expr.Q("Optional__isnull", false)),
	}
}

func (m *migComplexCheckModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Min"),
		attrs.NewField(m, "Max"),
		attrs.NewField(m, "Tag"),
		attrs.NewField(m, "Name"),
		attrs.NewField(m, "Code"),
		attrs.NewField(m, "Optional", &attrs.Config{Null: true}),
	).WithTableName("mig_con_complex")
}

func TestMigrator_Constraints_ComplexCheckExpressions_LiveSQLite(t *testing.T) {
	ctx := context.Background()
	db, ed := openTestSQLite(t)
	registerModelsContentTypes(&migComplexCheckModel{})
	initMigTestApp(t, db, func() django.AppConfig {
		app := apps.NewDBAppConfig("con_cplx_app")
		app.ModelObjects = []attrs.Definer{&migComplexCheckModel{}}
		return app
	})

	engine := migrator.NewMigrationEngine(t.TempDir(), ed)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Valid row satisfies all 5 CHECK constraints
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_con_complex (id, min, max, tag, name, code, optional) VALUES (1, 10, 20, 'alpha', 'ok-item', 'ABCD', 'present')"); err != nil {
		t.Fatalf("valid insert failed: %v", err)
	}
	badCases := []struct {
		name string
		sql  string
	}{
		{"min_gt_max", "INSERT INTO mig_con_complex (id, min, max, tag, name, code, optional) VALUES (2, 30, 20, 'alpha', 'ok-item', 'ABCD', 'present')"},
		{"bad_prefix", "INSERT INTO mig_con_complex (id, min, max, tag, name, code, optional) VALUES (2, 10, 20, 'alpha', 'no-item', 'ABCD', 'present')"},
		{"bad_tag", "INSERT INTO mig_con_complex (id, min, max, tag, name, code, optional) VALUES (2, 10, 20, 'gamma', 'ok-item', 'ABCD', 'present')"},
		{"short_code", "INSERT INTO mig_con_complex (id, min, max, tag, name, code, optional) VALUES (2, 10, 20, 'beta', 'ok-item', 'AB', 'present')"},
		{"null_opt", "INSERT INTO mig_con_complex (id, min, max, tag, name, code, optional) VALUES (2, 10, 20, 'beta', 'ok-item', 'ABCD', NULL)"},
	}
	for _, tc := range badCases {
		if _, err := db.ExecContext(ctx, tc.sql); err == nil {
			t.Errorf("%s: expected CHECK constraint failure", tc.name)
		}
	}
}

type dialectMockRows struct{}

func (r *dialectMockRows) Close() error               { return nil }
func (r *dialectMockRows) Columns() ([]string, error) { return nil, nil }
func (r *dialectMockRows) Err() error                 { return nil }
func (r *dialectMockRows) Next() bool                 { return false }
func (r *dialectMockRows) NextResultSet() bool        { return false }
func (r *dialectMockRows) Scan(...any) error          { return nil }

type dialectMockRow struct{ val int64 }

func (r dialectMockRow) Err() error { return nil }
func (r dialectMockRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		switch p := dest[0].(type) {
		case *int:
			*p = int(r.val)
		case *int64:
			*p = r.val
		case *bool:
			*p = r.val > 0
		}
	}
	return nil
}

type dialectMockDB struct {
	drv     driver.Driver
	queries []string
}

func (m *dialectMockDB) Unwrap() any                    { return m }
func (m *dialectMockDB) Driver() driver.Driver          { return m.drv }
func (m *dialectMockDB) Ping(context.Context) error     { return nil }
func (m *dialectMockDB) Close() error                   { return nil }
func (m *dialectMockDB) Finished() bool                 { return false }
func (m *dialectMockDB) Commit(context.Context) error   { return nil }
func (m *dialectMockDB) Rollback(context.Context) error { return nil }
func (m *dialectMockDB) Begin(context.Context) (drivers.Transaction, error) {
	return m, nil
}
func (m *dialectMockDB) QueryContext(_ context.Context, query string, _ ...any) (drivers.SQLRows, error) {
	m.queries = append(m.queries, query)
	return &dialectMockRows{}, nil
}
func (m *dialectMockDB) QueryRowContext(_ context.Context, query string, _ ...any) drivers.SQLRow {
	m.queries = append(m.queries, query)
	return dialectMockRow{val: 0}
}
func (m *dialectMockDB) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	m.queries = append(m.queries, query)
	return driver.RowsAffected(1), nil
}

type recordingConstraintLog struct {
	actions []migrator.ActionType
}

func (l *recordingConstraintLog) Log(action migrator.ActionType, file *migrator.MigrationFile, table *migrator.Changed[*migrator.ModelTable], column *migrator.Changed[*migrator.Column], index *migrator.Changed[*migrator.Index], constraint *migrator.Changed[constraints.Constraint]) {
	l.actions = append(l.actions, action)
}

func TestMigrator_Constraints_AllDialectsAndLogsViaEngine(t *testing.T) {
	dialects := []struct {
		name      string
		drv       driver.Driver
		newEditor func(drivers.Database) migrator.SchemaEditor
		wantAdd   string
		wantDrop  string
	}{
		{
			name:      "Postgres",
			drv:       &drivers.DriverPostgres{},
			newEditor: func(d drivers.Database) migrator.SchemaEditor { return postgres.NewPostgresSchemaEditor(d) },
			wantAdd:   `ALTER TABLE "mig_con_member" ADD CONSTRAINT "chk_con_member_role" CHECK ("role" IN ('user', 'admin'));`,
			wantDrop:  `ALTER TABLE "mig_con_member" DROP CONSTRAINT IF EXISTS "chk_con_member_age";`,
		},
		{
			name:      "MySQL",
			drv:       &drivers.DriverMySQL{},
			newEditor: func(d drivers.Database) migrator.SchemaEditor { return mysql.NewMySQLSchemaEditor(d) },
			wantAdd:   "ALTER TABLE `mig_con_member` ADD CONSTRAINT `chk_con_member_role` CHECK (`role` IN ('user', 'admin'));",
			wantDrop:  "ALTER TABLE `mig_con_member` DROP CHECK `chk_con_member_age`;",
		},
		{
			name:      "MariaDB",
			drv:       &drivers.DriverMariaDB{},
			newEditor: func(d drivers.Database) migrator.SchemaEditor { return mysql.NewMySQLSchemaEditor(d) },
			wantAdd:   "ALTER TABLE `mig_con_member` ADD CONSTRAINT `chk_con_member_role` CHECK (`role` IN ('user', 'admin'));",
			wantDrop:  "ALTER TABLE `mig_con_member` DROP CONSTRAINT `chk_con_member_age`;",
		},
	}

	for _, tc := range dialects {
		t.Run(tc.name, func(t *testing.T) {
			resetConEvoState()
			defer resetConEvoState()

			ctx := context.Background()
			mockDB := &dialectMockDB{drv: tc.drv}
			ed := tc.newEditor(mockDB)
			registerModelsContentTypes(&MigConOrg{}, &MigConMember{})
			initMigTestApp(t, mockDB, func() django.AppConfig {
				app := apps.NewDBAppConfig("con_eng_dialect")
				app.ModelObjects = []attrs.Definer{&MigConOrg{}, &MigConMember{}}
				return app
			})

			tmpDir := t.TempDir()
			recLog := &recordingConstraintLog{}
			engine := migrator.NewMigrationEngine(tmpDir, ed)
			engine.MigrationLog = recLog

			// Initial migration
			if err := engine.MakeMigrations(ctx); err != nil {
				t.Fatalf("MakeMigrations 1: %v", err)
			}
			// Add role constraint & drop age check via dropChecks=false -> step 2 add role, step 3 modify age
			conEvoState.addRoleConstraint = true
			if err := engine.MakeMigrations(ctx); err != nil {
				t.Fatalf("MakeMigrations 2: %v", err)
			}
			conEvoState.minAge = 21
			if err := engine.MakeMigrations(ctx); err != nil {
				t.Fatalf("MakeMigrations 3: %v", err)
			}

			// Verify filenames generated for constraint migrations
			if _, err := os.Stat(filepath.Join(tmpDir, "con_eng_dialect", "MigConMember", "0002_add_constraint_chk_con_member_role.mig")); err != nil {
				t.Errorf("expected 0002_add_constraint_chk_con_member_role.mig: %v", err)
			}
			if _, err := os.Stat(filepath.Join(tmpDir, "con_eng_dialect", "MigConMember", "0003_drop_constraint_chk_con_member_age_and_1_more.mig")); err != nil {
				t.Errorf("expected 0003_drop_constraint_chk_con_member_age_and_1_more.mig: %v", err)
			}

			// Apply all 3 migrations via MigrationEngine
			if err := engine.Migrate(ctx); err != nil {
				t.Fatalf("Migrate: %v", err)
			}

			allSQL := strings.Join(mockDB.queries, "\n")
			if !strings.Contains(allSQL, tc.wantAdd) {
				t.Errorf("expected %q in executed SQL, got:\n%s", tc.wantAdd, allSQL)
			}
			if !strings.Contains(allSQL, tc.wantDrop) {
				t.Errorf("expected %q in executed SQL, got:\n%s", tc.wantDrop, allSQL)
			}
			var sawAdd, sawDrop bool
			for _, a := range recLog.actions {
				if a == migrator.ActionAddConstraint {
					sawAdd = true
				}
				if a == migrator.ActionDropConstraint {
					sawDrop = true
				}
			}
			if !sawAdd || !sawDrop {
				t.Errorf("expected both ActionAddConstraint and ActionDropConstraint in MigrationLog, got %v", recLog.actions)
			}
		})
	}

	// Verify TestMigrationEngine records ActionAddConstraint & ActionDropConstraint correctly
	t.Run("TestMigrationEngine_Actions", func(t *testing.T) {
		resetConEvoState()
		defer resetConEvoState()

		ctx := context.Background()
		db, _ := openTestSQLite(t)
		registerModelsContentTypes(&MigConOrg{}, &MigConMember{})
		initMigTestApp(t, db, func() django.AppConfig {
			app := apps.NewDBAppConfig("con_testsql_app")
			app.ModelObjects = []attrs.Definer{&MigConOrg{}, &MigConMember{}}
			return app
		})

		ts := testsql.NewTestMigrationEngine(t)
		engine := migrator.NewMigrationEngine(t.TempDir(), ts)
		engine.MigrationLog = &migrator.MigrationEngineConsoleLog{}
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("MakeMigrations 1: %v", err)
		}
		conEvoState.minAge = 25
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("MakeMigrations 2: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Migrate: %v", err)
		}

		var addCount, dropCount int
		for _, a := range ts.Actions {
			switch a.Type {
			case migrator.ActionAddConstraint:
				addCount++
				if a.Constraint == nil {
					t.Errorf("expected non-nil Constraint on ActionAddConstraint")
				}
			case migrator.ActionDropConstraint:
				dropCount++
				if a.Constraint == nil {
					t.Errorf("expected non-nil Constraint on ActionDropConstraint")
				}
			}
		}
		if addCount < 5 || dropCount != 1 {
			t.Errorf("unexpected TestMigrationEngine constraint action counts: add=%d drop=%d", addCount, dropCount)
		}
	})
}
