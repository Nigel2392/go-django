package interfaces

import (
	"html/template"
	"io"

	"github.com/Nigel2392/go-django/core/httputils/tags"
	"github.com/Nigel2392/router/v3/request"
)

type Element interface {
	String() string
	HTML() template.HTML
}

type FormField interface {
	LabelHTML(r *request.Request, form_name string, tagmap tags.TagMap /* struct tags for the HTML input elemnt */) Element
	InputHTML(r *request.Request, form_name string, tagmap tags.TagMap /* struct tags for the HTML input elemnt */) Element
}

// This is a full implementation of the interfaces.
//
// These are all the possible interfaces that can be implemented on a field.
type EXAMPLE_FullFieldImplementation interface {
	// FormValues for the field will be passed into this. (Can not be used together with FileField)
	Field
	// FormFiles for the field will be passed into this. (Can not be used together with Field)
	FileField
	// This will be called before the field is rendered.
	//
	// This is to generate HTML input and label elements.
	FormField
	// This will be called right after FormValues or FormFiles is called,
	// and before saving the form.
	Validator
}

type Field interface {
	FormValues([]string) error // Formvalues for the field will be passed into this.
}

type File interface {
	Name() string
	Open() (io.ReadSeekCloser, error)
}

type FileField interface {
	FormFiles([]File) error // Formvalues for the field will be passed into this.
}

type FileSaver interface {
	Save(MediaWriter) error
}

type MediaWriter interface {
	WriteToMedia(path string, r io.Reader) (string, error)
	MediaPathToURL(path string) (string, error)
}

type Validator interface {
	Validate() error
}

type Lister[T any] interface {
	List(page, itemsPerPage int) (items []T, totalCount int64, err error)
}

// Fields of the creator model must adhere to the Field or FileField interface!
//
// It is also optional to implement the Validator interface, or the Initializer interface on the fields.
type Saver interface {
	Save() error
}

type Deleter interface {
	Delete() error
}
