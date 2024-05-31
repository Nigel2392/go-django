package tpl

import (
	"io/fs"
	"path/filepath"
	"strings"
)

type MatchFS struct {
	fs      fs.FS
	matches func(filepath string) bool
}

func NewMatchFS(fs fs.FS, matches func(filepath string) bool) *MatchFS {
	return &MatchFS{fs, matches}
}

func (m *MatchFS) Open(name string) (fs.File, error) {
	if m.matches != nil && !m.matches(name) {
		return nil, fs.ErrNotExist
	}
	return m.fs.Open(name)
}

func (m *MatchFS) ForceOpen(name string) (fs.File, error) {
	if forcer, ok := m.fs.(interface{ ForceOpen(string) (fs.File, error) }); ok {
		return forcer.ForceOpen(name)
	}
	return m.fs.Open(name)
}

func (m *MatchFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(m.fs, name)
}

func (m *MatchFS) FS() fs.FS {
	return m.fs
}

func MatchNever(string) bool {
	return false
}

func MatchAnd(matches ...func(filepath string) bool) func(filepath string) bool {
	return func(filepath string) bool {
		for _, match := range matches {
			if !match(filepath) {
				return false
			}
		}
		return true
	}
}

func MatchOr(matches ...func(filepath string) bool) func(filepath string) bool {
	return func(filepath string) bool {
		for _, match := range matches {
			if match(filepath) {
				return true
			}
		}
		return false
	}
}

func MatchPrefix(prefix string) func(filepath string) bool {
	// Not ""
	// Not ending with "."
	// Not ending with "/"
	prefix = filepath.ToSlash(prefix)
	if prefix != "" && !strings.HasSuffix(prefix, ".") && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return func(filepath string) bool {
		return filepath == prefix || strings.HasPrefix(filepath, prefix)
	}
}

func MatchSuffix(suffix string) func(path string) bool {
	suffix = filepath.ToSlash(suffix)
	if suffix != "" && !strings.HasPrefix(suffix, ".") && !strings.HasPrefix(suffix, "/") {
		suffix += "/"
	}
	return func(path string) bool {
		return path == suffix || strings.HasSuffix(path, suffix)
	}
}

func MatchExt(extension string) func(path string) bool {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	return func(path string) bool {
		return filepath.Ext(path) == extension
	}
}
