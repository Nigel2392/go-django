package sqlite_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/mattn/go-sqlite3"
)

var db drivers.Database

func init() {
	var err error
	db, err = drivers.Open(context.Background(), "sqlite3", "file:migrator_test?mode=memory&cache=shared")
	if err != nil {
		panic(fmt.Sprintf("failed to open db: %v", err))
	}

	var editor = sqlite.NewSQLiteSchemaEditor(db)
	if err = editor.Setup(context.Background()); err != nil {
		panic(fmt.Sprintf("failed to setup db: %v", err))
	}
}

type test interface {
	getField() attrs.Field
	getDriver() driver.Driver
	getValue() any
	expected() string
}

type tableTypeTest[T any] struct {
	fieldConfig attrs.FieldConfig
	driver      driver.Driver
	Val         T
	Expect      string
}

func (t *tableTypeTest[T]) FieldDefs() attrs.Definitions {
	return nil
}

func (t *tableTypeTest[T]) getField() attrs.Field {
	return attrs.NewField(t, "Val", &t.fieldConfig)
}

func (t *tableTypeTest[T]) getDriver() driver.Driver {
	return t.driver
}

func (t *tableTypeTest[T]) getValue() any {
	return t.Val
}

func (t *tableTypeTest[T]) expected() string {
	return t.Expect
}

var sqliteTests = []test{
	&tableTypeTest[int8]{
		Expect: "INTEGER",
	},
	&tableTypeTest[int16]{
		Expect: "INTEGER",
	},
	&tableTypeTest[int32]{
		Expect: "INTEGER",
	},
	&tableTypeTest[int64]{
		Expect: "INTEGER",
	},
	&tableTypeTest[float32]{
		Expect: "REAL",
	},
	&tableTypeTest[float64]{
		Expect: "REAL",
	},
	&tableTypeTest[string]{
		fieldConfig: attrs.FieldConfig{
			MaxLength: 255,
			MinLength: 0,
		},
		Expect: "TEXT",
	},
	&tableTypeTest[string]{
		fieldConfig: attrs.FieldConfig{
			MaxLength: 5,
			MinLength: 0,
		},
		Expect: "TEXT",
	},
	&tableTypeTest[string]{
		Expect: "TEXT",
	},
	&tableTypeTest[bool]{
		Expect: "BOOLEAN",
	},
	&tableTypeTest[sql.NullBool]{
		Expect: "BOOLEAN",
	},
	&tableTypeTest[time.Time]{
		Expect: "TIMESTAMP",
	},
	&tableTypeTest[[]byte]{
		Expect: "BLOB",
	},
	&tableTypeTest[json.RawMessage]{
		Expect: "TEXT", // SQLite does not have a native JSON type, so we use TEXT for JSON
	},
}

func TestTableTypes(t *testing.T) {
	var driver = &sqlite3.SQLiteDriver{}
	for _, test := range sqliteTests {
		var rT = reflect.TypeOf(test.getValue())
		t.Run(fmt.Sprintf("%T.%s", driver, rT.Name()), func(t *testing.T) {
			var field = test.getField()
			var expect = test.expected()
			var col = migrator.NewTableColumn(nil, field)
			var typ = migrator.GetFieldType(driver, &col)
			if typ != expect {
				t.Errorf("expected %q, got %q for %T", expect, typ, test.getValue())
			}
		})
	}
}

func TestCreateMigrationEntry(t *testing.T) {
	var editor = sqlite.NewSQLiteSchemaEditor(db)

	if err := editor.StoreMigration(context.Background(), "test_migration_app", "test_migration_model", "test_migration_name"); err != nil {
		t.Errorf("failed to store migration: %v", err)
	}

	var has, err = editor.HasMigration(context.Background(), "test_migration_app", "test_migration_model", "test_migration_name")
	if err != nil {
		t.Errorf("failed to check migration: %v", err)
	}

	if !has {
		t.Errorf("expected migration to exist, but it does not")
	}

	err = editor.RemoveMigration(context.Background(), "test_migration_app", "test_migration_model", "test_migration_name")
	if err != nil {
		t.Errorf("failed to delete migration: %v", err)
	}

	hasAfterDelete, err := editor.HasMigration(context.Background(), "test_migration_app", "test_migration_model", "test_migration_name")
	if err != nil {
		t.Errorf("failed to check migration after delete: %v", err)
	}

	if hasAfterDelete {
		t.Errorf("expected migration to not exist after delete, but it does")
	}
}
