package fs

import (
	"io/fs"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
)

type File struct {
	name string
	path string
	file fs.File
	hdr  *FileHeader
}

func (f *File) Read(p []byte) (n int, err error) {
	return f.file.Read(p)
}

func (f *File) Close() error {
	return f.file.Close()
}

func (f *File) Stat() (mediafiles.FileHeader, error) {
	var err error
	if f.hdr == nil {
		f.hdr, err = newFileHeader(f)
		if err != nil {
			return nil, err
		}
	}
	return f.hdr, nil
}
