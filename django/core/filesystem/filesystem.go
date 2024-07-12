package filesystem

import "mime/multipart"

type FileHeader interface {
	Name() string
	Size() int64
	Open() (multipart.File, error)
}
