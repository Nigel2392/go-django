package tpl

import (
	"errors"
	"io/fs"
)

type MultiFS struct {
	fs []fs.FS
}

func NewMultiFS(fileSystems ...fs.FS) *MultiFS {
	if len(fileSystems) == 0 {
		fileSystems = make([]fs.FS, 0)
	}
	return &MultiFS{fs: fileSystems}
}

func (m *MultiFS) Add(fs fs.FS, matches func(filepath string) bool) {
	if matches == nil {
		m.fs = append(m.fs, fs)
	} else {
		m.fs = append(m.fs, &MatchFS{fs, matches})
	}
}

func (m *MultiFS) Open(name string) (fs.File, error) {
	for _, f := range m.fs {
		file, err := f.Open(name)
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return file, err
	}

	return nil, fs.ErrNotExist
}
