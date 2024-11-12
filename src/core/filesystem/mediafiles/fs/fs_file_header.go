package fs

import (
	"io/fs"
	"time"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
)

type FileHeader struct {
	file *File
	stat fs.FileInfo
}

func newFileHeader(f *File) (*FileHeader, error) {
	stat, err := f.file.Stat()
	if err != nil {
		return nil, err
	}
	return &FileHeader{
		file: f,
		stat: stat,
	}, nil
}

func (h *FileHeader) Name() string {
	return h.file.name
}

func (h *FileHeader) Path() string {
	return h.file.path
}

func (h *FileHeader) Size() int64 {
	return h.stat.Size()
}

func (h *FileHeader) TimeAccessed() (t time.Time, err error) {
	return time.Time{}, mediafiles.ErrNotImplemented
}

func (h *FileHeader) TimeCreated() (t time.Time, err error) {
	return time.Time{}, mediafiles.ErrNotImplemented
}

func (h *FileHeader) TimeModified() (t time.Time, err error) {
	return h.stat.ModTime(), nil
}
