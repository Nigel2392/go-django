package interfaces

import (
	"html/template"
	"io"

	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
)

type Element interface {
	String() string
	HTML() template.HTML
}

type FormField interface {
	LabelHTML(r *request.Request, form_name string, display_text string, tagmap tags.TagMap /* struct tags for the HTML input elemnt */) Element
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

type Validator interface {
	Validate() error
}

type ValidatorTagged interface {
	ValidateWithTags(t tags.TagMap) error
}

type Lister[T any] interface {
	List(page, itemsPerPage int) (items []T, totalCount int64, err error)
}

// Fields of the creator model must adhere to the Field or FileField interface!
//
// It is also optional to implement the Validator interface, or the Initializer interface on the fields.
type Saver interface {
	Save(isNew bool) error
}

type Deleter interface {
	Delete() error
}

// Fields of the creator model must adhere to the Field or FileField interface!
//
// It is also optional to implement the Validator interface, or the Initializer interface on the fields.

type Updater interface {
	Update() error
}

type StringGetter[T any] interface {
	StringID() string
	GetFromStringID(id string) (item T, err error) // Returns the type of the model it was called on.
}

type Option interface {
	OptionLabel() string
	OptionValue() string
	OptionSelected() bool
}

// This method is called on FormFields, and not models.
type Initializer interface {
	Initial(r *request.Request, model any, fieldname string)
}

// A function to import JS into the form.
type Scripter interface {
	Script() (key string, value template.HTML)
}

// Options getter for fields defined in fields/slices.go
type OptionsGetter interface {
	// XXX is the fieldname, ___1 should be omitted.
	GetXXXOptions___1(r *request.Request) []string

	// XXX is the fieldname, ___2 should be omitted.
	GetXXXOptions___2(r *request.Request, model any) []string

	// XXX is the fieldname, ___3 should be omitted.
	GetXXXOptions___3(r *request.Request, model any, fieldName string) []string
}

// GenericOptionsGetter is used for fields that have a generic type.
type GenericOptionsGetter[T any] interface {
	// XXX is the fieldname.
	GetXXXOptions() []Option

	// XXX is the fieldname, ___1 should be omitted.
	GetXXXOptions___1(r *request.Request) []Option

	// XXX is the fieldname, ___2 should be omitted.
	GetXXXOptions___2(r *request.Request, model any) []Option

	// XXX is the fieldname, ___3 should be omitted.
	GetXXXOptions___3(r *request.Request, model any, fieldName string) []Option
}

// Options getter for fields defined in fields/multi_select.go
type MultiOptionsGetter interface {
	// XXX is the fieldname, ___1 should be omitted.
	GetXXXOptions___1() ([]Option, []Option)

	// XXX is the fieldname, ___2 should be omitted.
	GetXXXOptions___2(r *request.Request) ([]Option, []Option)

	// XXX is the fieldname, ___3 should be omitted.
	GetXXXOptions___3(r *request.Request, model any) ([]Option, []Option)

	// XXX is the fieldname, ___4 should be omitted.
	GetXXXOptions___4(r *request.Request, model any, fieldName string) ([]Option, []Option)
}

type DisplayGetter interface {
	// XXX is the fieldname, ___1 should be omitted.
	GetXXXDisplay___1() string
}
