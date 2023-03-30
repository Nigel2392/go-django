package fields

import (
	"errors"
	"io"

	"github.com/Nigel2392/go-django/core/app"
)

// The image struct represents an image in the database.
//
// The Path field is the path to the image on the filesystem.
//
// The URL field is the URL to the image.
type Image struct {
	Path string
	URL  string
}

// Create a new image from a reader.
func NewImage(filename string, r io.Reader) (*Image, error) {
	var i = &Image{}
	if err := i.Save(filename, r); err != nil {
		return nil, err
	}
	return i, nil
}

// Save the image to the filesystem, and update the Path and URL fields.
func (i *Image) Save(filename string, r io.Reader) error {
	if r == nil {
		return errors.New("reader is nil")
	}
	var fs = app.App().FS()
	var path, err = fs.WriteToMedia(filename, r)
	if err != nil {
		return err
	}
	i.Path = path
	i.URL, err = fs.MediaPathToURL(path)
	return err
}

func (i *Image) File() (io.Reader, error) {
	return app.App().FS().ReadFromMedia(i.Path)
}
