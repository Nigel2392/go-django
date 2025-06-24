package postgres_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	pg_stdlib "github.com/jackc/pgx/v5/stdlib"
)

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

var postgresTests = []test{
	&tableTypeTest[int8]{
		Expect: "SMALLINT",
	},
	&tableTypeTest[int16]{
		Expect: "INTEGER",
	},
	&tableTypeTest[int32]{
		Expect: "BIGINT",
	},
	&tableTypeTest[int64]{
		Expect: "BIGINT",
	},
	&tableTypeTest[float32]{
		Expect: "REAL",
	},
	&tableTypeTest[float64]{
		Expect: "DOUBLE PRECISION",
	},
	&tableTypeTest[string]{
		fieldConfig: attrs.FieldConfig{
			MaxLength: 255,
			MinLength: 0,
		},
		Expect: "VARCHAR(255)",
	},
	&tableTypeTest[string]{
		fieldConfig: attrs.FieldConfig{
			MaxLength: 5,
			MinLength: 0,
		},
		Expect: "VARCHAR(5)",
	},
	&tableTypeTest[drivers.Text]{
		Expect: "TEXT",
	},
	&tableTypeTest[drivers.String]{
		Expect: "VARCHAR(255)",
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
}

func TestTableTypes(t *testing.T) {
	var driver = &pg_stdlib.Driver{}
	for _, test := range postgresTests {
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
