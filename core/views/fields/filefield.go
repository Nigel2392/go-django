package fields

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
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

// Easily create files inside of forms.
//
// FileField will save as JSON inside a database.
type FileField struct {
	Path string          `json:"path"`
	URL  string          `json:"url"`
	File interfaces.File `json:"-"`
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

//	func NewFileField(path, url, name string, reader io.ReadSeekCloser) *FileField {
//		return &FileField{
//			Path: path,
//			URL:  url,
//			File: FormFile{
//				Filename: name,
//				OpenFunc: func() (io.ReadSeekCloser, error) {
//					return reader, nil
//				},
//			},
//		}
//	}

func (i *FileField) FormFiles(files []interfaces.File) error {
	if len(files) == 0 {
		return nil
	}
	if len(files) >= 1 {
		i.File = files[0]
	}
	return nil
}

// Save the file to the filer.
//
// This will also set the Path and URL fields.
func (i *FileField) Save(filer fs.Filer, media_url, pathInFiler string) error {
	var f, path, err = filer.Create(pathInFiler, i.File.Name())
	if err != nil {
		return err
	}
	defer f.Close()
	reader, err := i.File.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}
	i.Path = path
	path, err = filepath.Rel(filer.Base(), path)
	if err != nil {
		return err
	}
	i.URL = filepath.Join(media_url, path)
	return nil
}

func (i FileField) LabelHTML(r *request.Request, form_name, display_text string, tags tags.TagMap) interfaces.Element {
	var text = display_text
	if i.URL != "" {
		text = i.URL
	}
	var classes, ok = tags.GetOK("labelclass")
	if !ok {
		classes = []string{"file-upload-label"}
	} else {
		classes = append(classes, "file-upload-label")
	}
	tags["labelclass"] = classes
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, form_name, TagMapToElementAttributes(tags, AllTagsLabel...), text))
}

func (i FileField) InputHTML(r *request.Request, form_name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="file" name="%s" id="%s" %s>`, form_name, form_name, TagMapToElementAttributes(tags, AllTagsInput...)))
}

func (i *FileField) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var v []byte
	switch value.(type) {
	case []byte:
		v = value.([]byte)
	case string:
		v = []byte(value.(string))
	default:
		return fmt.Errorf("cannot scan value of type %T into FileField", value)
	}
	return json.Unmarshal(v, i)
}

func (i FileField) Value() (driver.Value, error) {
	var b bytes.Buffer
	var err = json.NewEncoder(&b).Encode(i)
	return b.String(), err
}
