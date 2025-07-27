package filesystem

import (
	"bytes"
	"io"
	"io/fs"
	"path/filepath"
	"time"
)

type stat[T fs.FS] struct {
	file    *cachedFile[T]
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (s *stat[T]) Name() string {
	return s.name
}

func (s *stat[T]) Size() int64 {
	return s.size
}

func (s *stat[T]) Mode() fs.FileMode {
	return s.mode
}

func (s *stat[T]) ModTime() time.Time {
	return s.modTime
}

func (s *stat[T]) IsDir() bool {
	return s.isDir
}

func (s *stat[T]) Sys() any {
	return s.file
}

type cachedFile[T fs.FS] struct {
	fs   *CacheFS[T]
	path string
	name string
	buf  []byte
	stat *stat[T]

	_reader io.Reader
}

func (c *cachedFile[T]) clone() *cachedFile[T] {
	return &cachedFile[T]{
		fs:      c.fs,
		path:    c.path,
		name:    c.name,
		buf:     c.buf,
		stat:    c.stat,
		_reader: bytes.NewReader(c.buf),
	}
}

func (c *cachedFile[T]) Read(p []byte) (n int, err error) {
	return c._reader.Read(p)
}

func (c *cachedFile[T]) Close() error {
	return nil // No-op, as we don't need to close a bytes.Buffer
}

func (c *cachedFile[T]) Stat() (fs.FileInfo, error) {
	return c.stat, nil
}

type CacheFS[T fs.FS] struct {
	FS    T
	Files map[string]*cachedFile[T]
}

func NewCacheFS[T fs.FS](fsys T) *CacheFS[T] {
	return &CacheFS[T]{
		FS:    fsys,
		Files: make(map[string]*cachedFile[T]),
	}
}

// Changed should be called when the underlying filesystem changes,
// to invalidate the cache, when no paths are provided it will clear the entire cache.
func (c *CacheFS[T]) Changed(paths ...string) {
	if len(paths) == 0 {
		clear(c.Files)
		return
	}

	for _, path := range paths {
		delete(c.Files, path)
	}
}

func (c *CacheFS[T]) Open(path string) (fs.File, error) {
	if cached, ok := c.Files[path]; ok {

		if cached == nil {
			return nil, fs.ErrNotExist
		}

		return cached.clone(), nil
	}

	file, err := c.FS.Open(path)
	if err != nil {
		return nil, err
	}

	var f = &cachedFile[T]{
		fs:   c,
		path: path,
		name: filepath.Base(path),
	}

	var buf = new(bytes.Buffer)
	if _, err = io.Copy(buf, file); err != nil {
		return nil, err
	}
	f.buf = buf.Bytes()

	fStat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	f.stat = &stat[T]{
		file:    f,
		name:    fStat.Name(),
		size:    fStat.Size(),
		mode:    fStat.Mode(),
		modTime: fStat.ModTime(),
		isDir:   fStat.IsDir(),
	}

	c.Files[path] = f

	return f.clone(), nil
}

func (c *CacheFS[T]) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(c.FS, name)
}
