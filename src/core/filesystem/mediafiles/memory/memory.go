package memory

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
)

var (
	_               mediafiles.Backend    = (*Backend)(nil)
	_               mediafiles.File       = (*File)(nil)
	_               mediafiles.FileHeader = (*FileHeader)(nil)
	DEFAULT_RETRIES                       = 5
)

func init() {
	mediafiles.RegisterBackend("memory", &Backend{})
	mediafiles.SetDefault("memory")
}

type Backend struct {
	files   map[string]*File
	mu      *sync.RWMutex
	Retries int
}

func NewBackend(retries int) *Backend {
	return &Backend{
		files:   make(map[string]*File),
		mu:      new(sync.RWMutex),
		Retries: retries,
	}
}

// Deletes the file referenced by name. If deletion is not supported on the target storage system this will return ErrNotImplemented.
func (b *Backend) Delete(path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.files[path]; !ok {
		return mediafiles.ErrNotFound
	}
	delete(b.files, path)
	return nil
}

// Returns True if a file referenced by the given name already exists in the storage system.
func (b *Backend) Exists(path string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.files[path]
	return ok, nil
}

// Returns an alternative filename based on the file_root and file_ext parameters, an underscore plus a random 7 character alphanumeric string is appended to the filename before the extension.
func (b *Backend) GetAlternateName(fileRoot, fileExt string) string {
	var random = make([]byte, 7)
	for i := range random {
		random[i] = byte(65 + rand.Intn(25))
	}
	return fmt.Sprintf("%s_%s%s", fileRoot, string(random), fileExt)
}

// Returns a filename based on the name parameter that’s free and available for new content to be written to on the target storage system.
//
// The length of the filename will not exceed max_length, if provided. If a free unique filename cannot be found, ErrSuspiciousOperation will be returned.
func (b *Backend) GetAvailableName(path string, retries int, max_length int) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if _, ok := b.files[path]; !ok {
		return path, nil
	}

	if retries == 0 {
		retries = DEFAULT_RETRIES
	}

	var ext = filepath.Ext(path)
	var root = path[:len(path)-len(ext)]
	for i := 0; i < retries; i++ {
		var alt = b.GetAlternateName(root, ext)
		if _, ok := b.files[alt]; !ok {
			return alt, nil
		}
	}

	return "", mediafiles.ErrSuspiciousOperation
}

func (b *Backend) Stat(path string) (mediafiles.FileHeader, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var file, ok = b.files[path]
	if !ok {
		return nil, mediafiles.ErrNotFound
	}
	return file.hdr, nil
}

// Lists the contents of the specified path, returning a 2-tuple of lists; the first item being directories, the second item being files.
// For storage systems that aren’t able to provide such a listing, this will return ErrNotImplemented.
func (b *Backend) ListDir(path string) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var files []string
	switch path {
	case "", "/":
		files = make([]string, 0, len(b.files))
		for file := range b.files {
			files = append(
				files,
				filepath.Base(file),
			)
		}
	default:
		files = make([]string, 0)
		cleanedPath := strings.Trim(
			filepath.Clean(path), "/",
		)
		for file := range b.files {
			var base, name = filepath.Split(file)
			if base == "" {
				files = append(files, name)
				continue
			}

			base = strings.Trim(
				filepath.Clean(base), "/",
			)

			if base == cleanedPath {
				files = append(files, name)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

func (b *Backend) Open(name string) (mediafiles.StoredObject, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	file, ok := b.files[name]
	if !ok {
		return nil, mediafiles.ErrNotFound
	}
	var obj = mediafiles.SimpleStoredObject{
		Filepath: name,
		OpenFn: func(path string) (mediafiles.File, error) {
			return file, nil
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
	var buf = new(bytes.Buffer)
	var maxLen int
	if len(maxLength) > 0 {
		maxLen = maxLength[0]
	}

	var name, err = b.GetAvailableName(path, b.Retries, maxLen)
	if err != nil {
		return "", err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.files == nil {
		assert.Fail("files map for memory backend is nil")
	}
	if _, ok := b.files[name]; ok {
		return "", mediafiles.ErrExists
	}

	_, err = io.Copy(buf, file)
	var hdr = &FileHeader{
		created:  time.Now(),
		modified: time.Now(),
	}
	var f = &File{
		name:    filepath.Base(name),
		path:    name,
		content: buf,
		hdr:     hdr,
	}
	hdr.file = f
	b.files[name] = f
	return name, err
}
