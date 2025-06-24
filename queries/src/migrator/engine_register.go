package migrator

import (
	"database/sql/driver"
	"reflect"

	"github.com/pkg/errors"
)

var schemaEditorRegistry = make(map[reflect.Type]func() (SchemaEditor, error))

func RegisterSchemaEditor(driver driver.Driver, fn func() (SchemaEditor, error)) {
	t := reflect.TypeOf(driver)
	schemaEditorRegistry[t] = fn
}

func GetSchemaEditor(driver driver.Driver) (SchemaEditor, error) {
	t := reflect.TypeOf(driver)
	if t == nil {
		return nil, errors.New("driver is nil")
	}

	var editor, ok = schemaEditorRegistry[t]
	if !ok {
		return nil, errors.Errorf("no schema editor registered for driver %T", driver)
	}

	return editor()
}
