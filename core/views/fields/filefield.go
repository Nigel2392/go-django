package fields

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/httputils/tags"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
)

type FormFile struct {
	Filename string
	OpenFunc func() (io.ReadSeekCloser, error)
}

func (f FormFile) Name() string {
	return f.Filename
}

func (f FormFile) Open() (io.ReadSeekCloser, error) {
	if f.OpenFunc == nil {
		return nil, errors.New("cannot open file, no open function provided")
	}
	return f.OpenFunc()
}

type FileField struct {
	Path string
	URL  string
	file interfaces.File
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

func NewFileField(name string, reader io.ReadSeekCloser) *FileField {
	return &FileField{
		Path: name,
		URL:  name,
		file: FormFile{
			Filename: name,
			OpenFunc: func() (io.ReadSeekCloser, error) {
				return reader, nil
			},
		},
	}
}

// Save the image to the filesystem, and update the Path and URL fields.
func (i *FileField) Save(mgr interfaces.MediaWriter) error {
	var (
		rd   io.ReadSeekCloser
		path string
		err  error
	)
	rd, err = i.file.Open()
	if err != nil {
		return err
	}
	path, err = mgr.WriteToMedia(i.file.Name(), rd)
	if err != nil {
		return err
	}
	i.Path = path
	i.URL, err = mgr.MediaPathToURL(path)
	return err
}

func (i *FileField) File(mgr *fs.Manager) (io.Reader, error) {
	return mgr.ReadFromMedia(i.Path)
}

func (i *FileField) FormFiles(files []interfaces.File) error {
	if len(files) == 0 {
		return nil
	}
	if len(files) >= 1 {
		i.file = files[0]
	}
	return nil
}

func (i FileField) LabelHTML(r *request.Request, form_name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, form_name, TagMapToElementAttributes(tags, AllTagsLabel...), i.URL))
}

func (i FileField) InputHTML(r *request.Request, form_name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="file" name="%s" %s>`, form_name, TagMapToElementAttributes(tags, AllTagsInput...)))
}
