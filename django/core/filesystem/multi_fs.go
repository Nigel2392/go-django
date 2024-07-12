package filesystem

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
	for i := len(m.fs) - 1; i >= 0; i-- {
		f := m.fs[i]
		file, err := f.Open(name)
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return file, err
	}

	return nil, fs.ErrNotExist
}

func (m *MultiFS) ForceOpen(name string) (fs.File, error) {
	for i := len(m.fs) - 1; i >= 0; i-- {
		f := m.fs[i]
		if forcer, ok := f.(interface{ ForceOpen(string) (fs.File, error) }); ok {
			file, err := forcer.ForceOpen(name)
			if err != nil && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return file, err
		}

		file, err := f.Open(name)
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return file, err
	}

	return nil, fs.ErrNotExist
}

func (m *MultiFS) FS() []fs.FS {
	return m.fs
}

func (m *MultiFS) ReadDir(name string) ([]fs.DirEntry, error) {
	for i := len(m.fs) - 1; i >= 0; i-- {
		f := m.fs[i]
		dir, err := fs.ReadDir(f, name)
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return dir, err
	}

	return nil, fs.ErrNotExist
}
