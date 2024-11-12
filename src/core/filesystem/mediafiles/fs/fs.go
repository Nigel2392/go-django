package fs

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
)

var (
	_               mediafiles.Backend    = (*Backend)(nil)
	_               mediafiles.File       = (*File)(nil)
	_               mediafiles.FileHeader = (*FileHeader)(nil)
	DEFAULT_RETRIES                       = 5
)

func init() {
	mediafiles.RegisterBackend("filesystem", &Backend{})
}

type Backend struct {
	BaseDir string
	Retries int
}

func NewBackend(baseDir string, retries int) *Backend {
	return &Backend{
		BaseDir: baseDir,
		Retries: retries,
	}
}

// Deletes the file referenced by name. If deletion is not supported on the target storage system this will return ErrNotImplemented.
func (b *Backend) Delete(path string) error {
	if _, err := b.Stat(path); err != nil {
		return err
	}

	var deletePath = path
	if b.BaseDir != "" {
		deletePath = filepath.Join(b.BaseDir, path)
	}

	return os.Remove(deletePath)
}

// Returns True if a file referenced by the given name already exists in the storage system.
func (b *Backend) Exists(path string) (bool, error) {
	var _, err = b.Stat(path)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Returns an alternative filename based on the file_root and file_ext parameters, an underscore plus a random 7 character alphanumeric string is appended to the filename before the extension.
func (b *Backend) GetAlternateName(fileRoot, fileExt string) string {
	var random = make([]byte, 7)
	for i := range random {
		random[i] = byte(65 + rand.Intn(25))
	}
	return fmt.Sprintf("%s_%s.%s", fileRoot, string(random), fileExt)
}

// Returns a filename based on the name parameter that’s free and available for new content to be written to on the target storage system.
//
// The length of the filename will not exceed max_length, if provided. If a free unique filename cannot be found, ErrSuspiciousOperation will be returned.
func (b *Backend) GetAvailableName(path string, retries int, max_length int) (string, error) {
	if _, err := b.Stat(path); err != nil && errors.Is(err, mediafiles.ErrNotFound) {
		return path, nil
	}

	if retries == 0 {
		retries = DEFAULT_RETRIES
	}

	// "my/path/to/file.txt" -> ["file", "txt"]
	var ext = filepath.Ext(path)
	path = strings.TrimSuffix(
		path, ext,
	)
	for i := 0; i < retries; i++ {
		var alt = b.GetAlternateName(path, ext)
		if _, err := b.Stat(alt); err != nil && errors.Is(err, mediafiles.ErrNotFound) {
			return alt, nil
		}
	}

	return "", mediafiles.ErrSuspiciousOperation
}

func (b *Backend) Stat(path string) (mediafiles.FileHeader, error) {
	var f, err = b.Open(path)
	if err != nil {
		return nil, err
	}
	file, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Stat()
}

// Lists the contents of the specified path, returning a 2-tuple of lists; the first item being directories, the second item being files.
// For storage systems that aren’t able to provide such a listing, this will return ErrNotImplemented.
func (b *Backend) ListDir(path string) ([]string, error) {
	var readPath = path
	if b.BaseDir != "" {
		readPath = filepath.Join(b.BaseDir, path)
	}

	var files []string
	var entries, err = os.ReadDir(
		readPath,
	)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return files, nil
}

func (b *Backend) Open(name string) (mediafiles.StoredObject, error) {
	var openPath = name
	if b.BaseDir != "" {
		openPath = filepath.Join(b.BaseDir, name)
	}

	var obj = mediafiles.SimpleStoredObject{
		Filepath: name,
		OpenFn: func(path string) (mediafiles.File, error) {
			var f, err = os.Open(openPath)
			if err != nil {
				return nil, err
			}
			return &File{
				name: filepath.Base(path),
				path: path,
				file: f,
			}, nil
		},
	}

	return &obj, nil
}

// Saves a new file using the storage system, preferably with the name specified. If there already exists a file with this name name, the storage system may modify the filename as necessary to get a unique name.
// The actual name of the stored file will be returned.
//
// The maxLength parameter is passed to GetAvailableName() and is used to limit the length of the filename before saving.
// If the file is too large to be saved, this will raise a SuspiciousOperation exception.
func (b *Backend) Save(path string, file io.Reader, maxLength ...int) (string, error) {
	// do not lock here, GetAvailableName will lock
	var (
		maxLen int
		err    error
	)
	if len(maxLength) > 0 {
		maxLen = maxLength[0]
	}

	path, err = b.GetAvailableName(
		path, b.Retries, maxLen,
	)
	if err != nil {
		return "", err
	}

	var createPath = path
	if b.BaseDir != "" {
		createPath = filepath.Join(
			b.BaseDir, path,
		)
	}

	err = os.MkdirAll(
		filepath.Dir(createPath),
		os.ModePerm,
	)
	if err != nil {
		return "", err
	}

	f, err := os.Create(createPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		return "", err
	}

	return path, nil
}
