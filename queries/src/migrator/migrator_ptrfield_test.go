package migrator_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

type ptrFieldDefaultsModel struct {
	ID            drivers.ULID
	Amount        decimal.Decimal
	Fee           decimal.Decimal
	TransactionID sql.NullString
}

func (m *ptrFieldDefaultsModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, drivers.ULID] {
			return fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, drivers.ULID]{Config: attrs.Config{Primary: true}}
		}),
		fattrs.Field(m, "Amount", &m.Amount, func() fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, decimal.Decimal]{Config: attrs.Config{Precision: 13, Scale: 4}}
		}),
		fattrs.Field(m, "Fee", &m.Fee, func() fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, decimal.Decimal]{Config: attrs.Config{Precision: 13, Scale: 4, Default: decimal.Zero}}
		}),
		fattrs.Field(m, "TransactionID", &m.TransactionID, func() fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, sql.NullString] {
			return fattrs.PtrFieldConfig[*ptrFieldDefaultsModel, sql.NullString]{Config: attrs.Config{Null: true, Blank: true, MaxLength: 255}}
		}),
	)
}

func TestMakeMigrations_PtrFieldDefaultsAreStable(t *testing.T) {
	const appName = "ptrfielddefaults"
	ctx := context.Background()
	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig(appName)
		cfg.ModelObjects = []attrs.Definer{&ptrFieldDefaultsModel{}}
		return cfg
	})

	dir := t.TempDir()
	engine := migrator.NewMigrationEngine(dir, editor)
	if err := engine.MakeMigrations(ctx, appName); err != nil {
		t.Fatalf("create initial migration: %v", err)
	}

	for run := 2; run <= 3; run++ {
		// Use a fresh engine to compare the model against the snapshot on disk.
		engine := migrator.NewMigrationEngine(dir, editor)
		err := engine.MakeMigrations(ctx, appName)
		if errors.Is(err, migrator.ErrNoChanges) {
			continue
		}
		if err != nil {
			t.Fatalf("MakeMigrations run %d: %v", run, err)
		}

		migrations, err := engine.ReadMigrations(appName)
		if err != nil {
			t.Fatalf("read migrations after run %d: %v", run, err)
		}
		latest := migrations[len(migrations)-1]
		t.Errorf("MakeMigrations run %d wrote %s for an unchanged model", run, latest.FileName())
		for _, action := range latest.Actions {
			if action.ActionType == migrator.ActionAlterField {
				old, new := action.Field.Old, action.Field.New
				t.Logf("altered %s: default %#v -> %#v", new.Name, old.Default.V, new.Default.V)
			}
		}
	}
}
