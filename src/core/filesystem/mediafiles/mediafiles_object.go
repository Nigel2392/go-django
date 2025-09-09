package mediafiles

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"path/filepath"
	"reflect"

	"github.com/pkg/errors"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/except"
)

var (
	_ driver.Valuer = (*SimpleStoredObject)(nil)
	_ sql.Scanner   = (*SimpleStoredObject)(nil)
	_ StoredObject  = (*SimpleStoredObject)(nil)
)

type SimpleStoredObject struct {
	Filepath string
	OpenFn   func(path string) (File, error)
}

func (s *SimpleStoredObject) DBType() dbtype.Type {
	return dbtype.String
}

func (s *SimpleStoredObject) String() string {
	return s.Filepath
}

func (s *SimpleStoredObject) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var filepath string
	var rVal = reflect.ValueOf(value)
	switch rVal.Kind() {
	case reflect.Slice, reflect.Array:
		switch rVal.Type().Elem().Kind() {
		case reflect.Uint8:
			rVal = rVal.Convert(reflect.TypeOf([]byte{}))
			filepath = string(rVal.Bytes())
		case reflect.Int32:
			rVal = rVal.Convert(reflect.TypeOf([]rune{}))
			filepath = string(rVal.Interface().([]rune))
		default:
			return errors.Errorf("invalid type for scan %T", value)
		}
	case reflect.String:
		filepath = rVal.String()
	case reflect.Ptr:
		if rVal.IsNil() {
			return nil
		}
		return s.Scan(rVal.Elem().Interface())
	default:
		return errors.Errorf("invalid type for scan %T", value)
	}

	s.Filepath = filepath

	except.Assert(
		defaultBackend != nil, 500,
		"defaultBackend is nil",
	)

	var s2, err = defaultBackend.Open(
		s.Filepath,
	)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	s.OpenFn = func(path string) (File, error) {
		return s2.Open()
	}
	return nil
}

func (s *SimpleStoredObject) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return s.Filepath, nil
}

func (s *SimpleStoredObject) Name() string {
	return filepath.Base(s.Filepath)
}

func (s *SimpleStoredObject) Path() string {
	return s.Filepath
}

func (s *SimpleStoredObject) Open() (File, error) {
	return s.OpenFn(s.Filepath)
}

func (s *SimpleStoredObject) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal(s.Filepath)
}

func (s *SimpleStoredObject) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.Filepath = ""
		s.OpenFn = nil
		return nil
	}

	var filepath string
	if err := json.Unmarshal(data, &filepath); err != nil {
		return err
	}
	s.Filepath = filepath

	except.Assert(
		defaultBackend != nil, 500,
		"defaultBackend is nil",
	)

	var s2, err = defaultBackend.Open(
		s.Filepath,
	)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	s.OpenFn = func(path string) (File, error) {
		return s2.Open()
	}
	return nil
}
