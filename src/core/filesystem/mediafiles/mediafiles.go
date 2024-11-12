package mediafiles

import (
	"io"
	"os"
	"time"

	"github.com/Nigel2392/go-django/src/core/errs"
)

var (
	// ErrNotImplemented is returned when an operation is not supported by the storage backend.
	ErrNotImplemented = errs.Error("unsupported operation for this storage backend")

	// ErrNotFound is returned when a file is not found.
	ErrNotFound = os.ErrNotExist

	// ErrExists is returned when a file already exists.
	ErrExists = os.ErrExist

	// ErrSuspiciousOperation is returned when an operation is considered suspicious.
	// For example, when a file is too large to be saved.
	ErrSuspiciousOperation = os.ErrInvalid
)

type FileHeader interface {
	Name() string
	Path() string
	Size() int64

	// Returns the time the file was last accessed.
	//
	// If the backend storage system does not support this operation, err will be ErrNotImplemented.
	TimeAccessed() (t time.Time, err error)

	// Returns the time the file was created.
	//
	// If the backend storage system does not support this operation, err will be ErrNotImplemented.
	TimeCreated() (t time.Time, err error)

	// Returns the time the file was last modified.
	//
	// If the file was never modified, the returned time will be the same as the time the file was created.
	//
	// If the backend storage system does not support this operation, err will be ErrNotImplemented.
	TimeModified() (t time.Time, err error)
}

type File interface {
	// Reads content from the file.
	io.Reader

	// Closes the file.
	io.Closer

	// Returns the file header.
	Stat() (FileHeader, error)
}

type StoredObject interface {
	// Returns the name of the file, excluding path information.
	Name() string

	// Returns the full path of the file.
	Path() string

	// Returns the size of the file in bytes.
	Open() (File, error)
}

type Backend interface {
	// Deletes the file referenced by name. If deletion is not supported on the target storage system this will return ErrNotImplemented.
	Delete(path string) error

	// Returns True if a file referenced by the given name already exists in the storage system.
	Exists(path string) (bool, error)

	// Returns an alternative filename based on the file_root and file_ext parameters, an underscore plus a random 7 character alphanumeric string is appended to the filename before the extension.
	GetAlternateName(fileRoot, fileExt string) string

	// Returns a filename based on the name parameter that’s free and available for new content to be written to on the target storage system.
	//
	// The length of the filename will not exceed max_length, if provided. If a free unique filename cannot be found, ErrSuspiciousOperation will be returned.
	GetAvailableName(path string, retries, maxLength int) (string, error)

	// Stat returns the FileInfo structure describing the file.
	Stat(path string) (FileHeader, error)

	// Lists the contents of the specified path, returning a 2-tuple of lists; the first item being directories, the second item being files.
	// For storage systems that aren’t able to provide such a listing, this will return ErrNotImplemented.
	ListDir(path string) ([]string, error)

	// Opens the file given by name. Note that although the returned file is guaranteed to be a StoredObject interface object, it might actually be some subclass.
	// In the case of remote file storage this means that reading/writing could be quite slow, so be warned.
	Open(path string) (StoredObject, error)

	// Saves a new file using the storage system, preferably with the name specified. If there already exists a file with this name name, the storage system may modify the filename as necessary to get a unique name.
	// The actual name of the stored file will be returned.
	//
	// The maxLength parameter is passed to GetAvailableName() and is used to limit the length of the filename before saving.
	// If the file is too large to be saved, this will raise a SuspiciousOperation exception.
	Save(path string, file io.Reader, maxLength ...int) (string, error)
}
