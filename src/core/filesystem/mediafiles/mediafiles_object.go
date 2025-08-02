package mediafiles

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"

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
	switch v := value.(type) {
	case string:
		filepath = v
	case []byte:
		filepath = string(v)
	case []rune:
		filepath = string(v)
	default:
		return errors.New("invalid type")
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
