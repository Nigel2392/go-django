package quest

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type DBTables[T testing.TB] struct {
	tables []*migrator.ModelTable
	schema migrator.SchemaEditor
	t      T
}

func Table[T testing.TB](t T, model ...attrs.Definer) *DBTables[T] {
	if len(model) == 0 {
		panic("No model provided to Table()")
	}

	var (
		db    interface{ Driver() driver.Driver }
		err   error
		table = &DBTables[T]{}
	)
	if django.Global != nil && django.Global.Settings != nil {
		db = django.ConfigGet[interface{ Driver() driver.Driver }](
			django.Global.Settings,
			django.APPVAR_DATABASE,
		)
	} else {
		db, err = drivers.Open(context.Background(), "sqlite3", "file:quest_memory?mode=memory")
		if err != nil {
			table.fatalf("Failed to open database: %v", err)
			return nil
		}
	}

	schemaEditor, err := migrator.GetSchemaEditor(db.Driver())
	if err != nil {
		table.fatalf("Failed setup SchemaEditor: %v", err)
		return nil
	}

	table.tables = make([]*migrator.ModelTable, len(model))
	for _, m := range model {
		attrs.RegisterModel(m)
	}
	for i, m := range model {
		table.tables[i] = migrator.NewModelTable(m)
	}
	table.schema = schemaEditor
	table.t = t
	return table
}

func (t *DBTables[T]) fatal(args ...interface{}) {
	if reflect.ValueOf(t.t).IsNil() {
		panic(fmt.Sprint(args...))
	}
	t.t.Helper()
	t.t.Fatal(args...)
}

func (t *DBTables[T]) fatalf(format string, args ...interface{}) {
	if reflect.ValueOf(t.t).IsNil() {
		panic(fmt.Sprintf(format, args...))
	}
	t.t.Helper()
	t.t.Fatalf(format, args...)
}

func (t *DBTables[T]) Create() {
	if t.schema == nil {
		t.fatal("SchemaEditor is not initialized")
		return
	}

	for _, table := range t.tables {

		if !reflect.ValueOf(t.t).IsNil() {
			t.t.Logf("Creating table: %s", table.TableName())
		} else {
			fmt.Printf("Creating table: %s\n", table.TableName())
		}

		err := t.schema.CreateTable(context.Background(), table, false)
		if err != nil {
			t.fatalf("Failed to create table (%s): %v", table.ModelName(), err)
			return
		}

	}
	return
}

func (t *DBTables[T]) Drop() {
	if t.schema == nil {
		t.fatal("SchemaEditor is not initialized")
	}

	for _, table := range t.tables {
		err := t.schema.DropTable(context.Background(), table, false)
		if err != nil {
			t.fatalf("Failed to drop table (%s): %v", table.ModelName(), err)
		}
	}
	return
}
