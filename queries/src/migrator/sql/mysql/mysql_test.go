package mysql_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	mysql_driver "github.com/go-sql-driver/mysql"
)

type User struct {
	models.Model
	ID   int `attr:"primary;read_only"`
	Name string
}

func (m *User) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
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

func (m *Todo) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,

		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Column:   "id", // can be inferred, but explicitly set for clarity
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Column: "title", // can be inferred, but explicitly set for clarity
		}),
		attrs.NewField(m, "Description", &attrs.FieldConfig{
			Column: "description", // can be inferred, but explicitly set for clarity
		}),
		attrs.NewField(m, "Done", &attrs.FieldConfig{}),
		attrs.NewField(m, "User", &attrs.FieldConfig{
			Column:      "user_id",
			RelOneToOne: attrs.Relate(&User{}, "", nil),
		}),
	).WithTableName("todos")
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

func (t *tableTypeTest[T]) getCol() *migrator.Column {
	return &migrator.Column{}
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

var mySQLTests = []test{
	&tableTypeTest[int8]{
		Expect: "SMALLINT",
	},
	&tableTypeTest[int16]{
		Expect: "INT",
	},
	&tableTypeTest[int32]{
		Expect: "BIGINT",
	},
	&tableTypeTest[int64]{
		Expect: "BIGINT",
	},
	&tableTypeTest[float32]{
		Expect: "FLOAT",
	},
	&tableTypeTest[float64]{
		Expect: "DOUBLE",
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
	&tableTypeTest[string]{
		Expect: "TEXT",
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
		Expect: "DATETIME",
	},
}

func TestTableTypes(t *testing.T) {
	var driver = &mysql_driver.MySQLDriver{}
	for _, test := range mySQLTests {
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
