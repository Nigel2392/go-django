package filesystem

import (
	"errors"
	"io/fs"
	"sync"
)

// MultiFS is a filesystem that combines multiple filesystems.
//
// It can be used to combine multiple filesystems into a single filesystem.
//
// When opening a file, it will try to open the file in each filesystem in the order they were added.
//
// It is best (and automatically) used with the `MatchFS` filesystem to restrict access to files in the filesystems.
//
// This allows for faster skipping of filesystems that do not contain the file.
type MultiFS struct {
	fs     []fs.FS
	cached map[string]int
	mu     sync.Mutex
}

// NewMultiFS creates a new MultiFS filesystem that combines the given filesystems.
//
// If no filesystems are given, an empty MultiFS filesystem is created.
func NewMultiFS(fileSystems ...fs.FS) *MultiFS {
	if len(fileSystems) == 0 {
		fileSystems = make([]fs.FS, 0)
	}
	return &MultiFS{
		fs:     fileSystems,
		cached: make(map[string]int),
		mu:     sync.Mutex{},
	}
}

// Add adds the given filesystem to the MultiFS filesystem.
//
// If a matcher-func is given, it will only allow opening files that match the given matcher-func.
//
// This allows for restricting access to files in the filesystem.
//
// A regular `fs.FS` is added to the pool if no matcher-func is given.
func (m *MultiFS) Add(fs fs.FS, matches func(filepath string) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if matches == nil {
		m.fs = append(m.fs, fs)
	} else {
		m.fs = append(m.fs, &MatchFS{fs, matches})
	}
}

// Open opens the file at the given path.
//
// It will try to open the file in each filesystem in the order they were added.
func (m *MultiFS) Open(name string) (fs.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to open the file from the fs at the cached index
	// if exists and when Open is called the returned error
	// is fs.ErrNotFound - we will keep looking
	if fsIndex, ok := m.cached[name]; ok {
		var f, err = m.fs[fsIndex].Open(name)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				goto loopOverFilesystems
			}
			return nil, err
		}

		return f, nil
	}

loopOverFilesystems:
	for i := len(m.fs) - 1; i >= 0; i-- {
		var fsys = m.fs[i]
		var file, err = fsys.Open(name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}

		m.cached[name] = i

		return file, err
	}

	return nil, fs.ErrNotExist
}

// ForceOpen opens the file at the given path, even if it does not match the matcher-func.
//
// This allows for bypassing any restrictions that the matcher-func of any underlying `MatchFS` might impose.
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

// FS returns the list of filesystems that are being combined by the MultiFS.
func (m *MultiFS) FS() []fs.FS {
	return m.fs
}

// ReadDir reads the directory at the given path.
//
// It returns the list of files in the provided directory (if any).
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
