package memory

import "time"

type FileHeader struct {
	file     *File
	created  time.Time
	modified time.Time
	accessed time.Time
}

func (h *FileHeader) Name() string {
	return h.file.name
}

func (h *FileHeader) Path() string {
	return h.file.path
}

func (h *FileHeader) Size() int64 {
	return int64(h.file.content.Len())
}

func (h *FileHeader) TimeAccessed() (t time.Time, err error) {
	return h.accessed, nil
}

func (h *FileHeader) TimeCreated() (t time.Time, err error) {
	return h.created, nil
}

func (h *FileHeader) TimeModified() (t time.Time, err error) {
	return h.modified, nil
}
