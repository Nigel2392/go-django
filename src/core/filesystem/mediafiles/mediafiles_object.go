package mediafiles

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
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

func (s *SimpleStoredObject) Scan(value interface{}) error {
	s.Filepath = value.(string)
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
