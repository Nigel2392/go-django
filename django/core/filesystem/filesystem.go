package filesystem

import (
	"io/fs"
	"mime/multipart"

	"github.com/Nigel2392/django/core/assert"
)

type FileHeader interface {
	Name() string
	Size() int64
	Open() (multipart.File, error)
}

func Sub(fileSys fs.FS, path string) fs.FS {
	var f, err = fs.Sub(fileSys, path)
	if err != nil {
		assert.Fail(err)
	}
	return f
}
