package fs

import (
	"errors"
	"io"
	"strings"

	"github.com/Nigel2392/go-django/core/httputils"
)

type FileField struct {
	Path string
	URL  string
}

func (i FileField) String() string {
	return httputils.CutFrontPath(i.Path, 20)
}

func (i FileField) ListDisplay() string {
	var b strings.Builder
	b.Grow(len("<a href=\"\"></a>") + len(i.URL) + len(i.String()))
	b.WriteString("<a href=\"")
	b.WriteString(i.URL)
	b.WriteString("\" target=\"_blank\">")
	b.WriteString(i.String())
	b.WriteString("</a>")
	return b.String()
}

// Create a new image from a reader.
func NewFile(mgr *Manager, filename string, r io.Reader) (*FileField, error) {
	var i = &FileField{}
	if err := i.Save(mgr, filename, r); err != nil {
		return nil, err
	}
	return i, nil
}

// Save the image to the filesystem, and update the Path and URL fields.
func (i *FileField) Save(mgr *Manager, filename string, r io.Reader) error {
	if r == nil {
		return errors.New("reader is nil")
	}
	var path, err = mgr.WriteToMedia(filename, r)
	if err != nil {
		return err
	}
	i.Path = path
	i.URL, err = mgr.MediaPathToURL(path)
	return err
}

func (i *FileField) File(mgr *Manager) (io.Reader, error) {
	return mgr.ReadFromMedia(i.Path)
}
