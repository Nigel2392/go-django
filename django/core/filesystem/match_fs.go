package filesystem

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// MatchFS is a filesystem that only allows opening files that match a given matcher-func.
//
// It can be used to restrict access to files in a filesystem.
//
// The matcher-func is called with the path of the file that is being opened.
type MatchFS struct {
	fs      fs.FS
	matches func(filepath string) bool
}

// NewMatchFS creates a new MatchFS filesystem that wraps the given filesystem and only allows opening files that match the given matcher-func.
func NewMatchFS(fs fs.FS, matches func(filepath string) bool) *MatchFS {
	return &MatchFS{fs, matches}
}

// Open opens the file at the given path if the path matches the matcher-func.
func (m *MatchFS) Open(name string) (fs.File, error) {
	if m.matches != nil && !m.matches(name) {
		return nil, fs.ErrNotExist
	}
	return m.fs.Open(name)
}

// ForceOpen opens the file at the given path, even if it does not match the matcher-func.
//
// This allows for bypassing any restrictions that the matcher-func might impose.
func (m *MatchFS) ForceOpen(name string) (fs.File, error) {
	if forcer, ok := m.fs.(interface{ ForceOpen(string) (fs.File, error) }); ok {
		return forcer.ForceOpen(name)
	}
	return m.fs.Open(name)
}

// ReadDir reads the directory at the given path.
//
// It returns the list of files in the directory if the path matches the matcher-func.
func (m *MatchFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(m.fs, name)
}

// FS returns the underlying filesystem that is being wrapped by the MatchFS.
func (m *MatchFS) FS() fs.FS {
	return m.fs
}

// MatchNever returns a matcher-func that never matches any file.
func MatchNever(string) bool {
	return false
}

// MatchAnd returns a matcher-func that matches a file if all the given matchers match.
//
// It can be passed any number of matchers.
//
// This allows for more complex logic when matching paths.
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

// MatchOr returns a matcher-func that matches a file if any of the given matchers match.
//
// It can be passed any number of matchers.
//
// This allows for more complex logic when matching paths.
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

// MatchPrefix returns a matcher-func that matches a file if the given prefix matches the file path.
//
// The prefix is normalized to use "/" as the path separator.
//
// If the prefix is not empty and does not end with a "." or "/", it is appended with a "/".
//
// When matching, the file path is compared to the prefix, the provided path either has to be the prefix or start with the prefix.
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

// MatchSuffix returns a matcher-func that matches a file if the given suffix matches the file path.
//
// The suffix is normalized to use "/" as the path separator.
//
// If the suffix is not empty and does not start with a "." or "/", it is prepended with a "/".
//
// When matching, the file path is compared to the suffix, the provided path either has to be the suffix or end with the suffix.
func MatchSuffix(suffix string) func(path string) bool {
	suffix = filepath.ToSlash(suffix)
	if suffix != "" && !strings.HasPrefix(suffix, ".") && !strings.HasPrefix(suffix, "/") {
		suffix += "/"
	}
	return func(path string) bool {
		return path == suffix || strings.HasSuffix(path, suffix)
	}
}

// MatchExt returns a matcher-func that matches a file if the given extension matches the file path.
//
// The extension passed to this function is normalized to start with a ".".
//
// When matching, the extension is retrieved from the file path with filepath.Ext and compared to the provided extension.
func MatchExt(extension string) func(path string) bool {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	return func(path string) bool {
		return filepath.Ext(path) == extension
	}
}
