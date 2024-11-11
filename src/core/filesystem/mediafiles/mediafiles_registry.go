package mediafiles

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
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

type backendRegistry struct {
	registry map[string]Backend
}

func (r *backendRegistry) Register(name string, backend Backend) {
	r.registry[name] = backend
}

func (r *backendRegistry) Backend(name string) (Backend, bool) {
	backend, ok := r.registry[name]
	return backend, ok
}

func newBackendRegistry() *backendRegistry {
	return &backendRegistry{
		registry: make(map[string]Backend),
	}
}

var (
	backends       = newBackendRegistry()
	defaultBackend Backend
)

func RegisterBackend(name string, backend Backend) {
	backends.Register(name, backend)
}

func RetrieveBackend(name string) (Backend, bool) {
	return backends.Backend(name)
}

func SetDefault(backend string) Backend {
	if _, ok := backends.Backend(backend); !ok {
		panic(fmt.Sprintf(
			"mediafiles: backend %q is not registered",
			backend,
		))
	}
	var b = backends.registry[backend]
	defaultBackend = b
	return b
}

// Deletes the file referenced by name. If deletion is not supported on the target storage system this will return ErrNotImplemented.
func Delete(path string) error {
	return defaultBackend.Delete(path)
}

// Returns True if a file referenced by the given name already exists in the storage system.
func Exists(path string) (bool, error) {
	return defaultBackend.Exists(path)
}

// Returns an alternative filename based on the file_root and file_ext parameters, an underscore plus a random 7 character alphanumeric string is appended to the filename before the extension.
func GetAlternateName(fileRoot, fileExt string) string {
	return defaultBackend.GetAlternateName(fileRoot, fileExt)
}

// Returns a filename based on the name parameter that’s free and available for new content to be written to on the target storage system.
//
// The length of the filename will not exceed max_length, if provided. If a free unique filename cannot be found, ErrSuspiciousOperation will be returned.
func GetAvailableName(path string, retries int, max_length int) (string, error) {
	return defaultBackend.GetAvailableName(path, retries, max_length)
}

// Returns the file header for the file referenced by path.
func Stat(path string) (FileHeader, error) {
	return defaultBackend.Stat(path)
}

// Returns a filename based on the name parameter that’s suitable for use on the target storage system.
func GetValidName(name string) string {
	return defaultBackend.GetValidName(name)
}

// Validates the filename by calling GetValidName() and returns a filename to be passed to the Save() method.
func GenerateFilename(filename string) string {
	return defaultBackend.GenerateFilename(filename)
}

// Lists the contents of the specified path, returning a 2-tuple of lists; the first item being directories, the second item being files.
// For storage systems that aren’t able to provide such a listing, this will return ErrNotImplemented.
func ListDir(path string) ([]string, error) {
	return defaultBackend.ListDir(path)
}

// Opens the file given by name. Note that although the returned file is guaranteed to be a StoredObject interface object, it might actually be some subclass.
// In the case of remote file storage this means that reading/writing could be quite slow, so be warned.
func Open(path string) (StoredObject, error) {
	return defaultBackend.Open(path)
}

// Saves a new file using the storage system, preferably with the name specified. If there already exists a file with this name name, the storage system may modify the filename as necessary to get a unique name.
// The actual name of the stored file will be returned.
//
// The maxLength parameter is passed to GetAvailableName() and is used to limit the length of the filename before saving.
// If the file is too large to be saved, this will raise a SuspiciousOperation exception.
func Save(path string, file io.Reader, maxLength ...int) (string, error) {
	return defaultBackend.Save(path, file, maxLength...)
}
