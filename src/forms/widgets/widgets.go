package widgets

import (
	"io"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
)

type FormValueConverter interface {
	// Convert the forms' string value to the appropriate GO type.
	ValueToGo(value interface{}) (interface{}, error)

	// Convert the GO type to the forms' string value.
	ValueToForm(value interface{}) interface{}
}

type FormValueOmitter interface {
	// Check if the value is omitted from the data provided.
	ValueOmittedFromData(data url.Values, files map[string][]filesystem.FileHeader, name string) bool
}

type FormValueGetter interface {
	// Get the value from the provided data.
	ValueFromDataDict(data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error)
}

type FormValuer interface {
	FormValueConverter
	FormValueOmitter
	FormValueGetter
}

type Widget interface {
	IsHidden() bool
	Hide(hidden bool)
	FormType() string
	SetAttrs(attrs map[string]string)
	IdForLabel(id string) string
	GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context
	RenderWithErrors(w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error
	Render(w io.Writer, id, name string, value interface{}, attrs map[string]string) error
	Validate(value interface{}) []error

	FormValuer
	media.MediaDefiner
}
