package memory

import (
	"bytes"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
)

type File struct {
	name    string
	path    string
	content *bytes.Buffer
	hdr     *FileHeader
}

func (f *File) Read(p []byte) (n int, err error) {
	return f.content.Read(p)
}

func (f *File) Close() error {
	return nil
}

func (f *File) Stat() (mediafiles.FileHeader, error) {
	return f.hdr, nil
}
