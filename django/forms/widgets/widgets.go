package widgets

import (
	"io"
	"net/url"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/media"
)

type FormValueConverter interface {
	ValueToGo(value interface{}) (interface{}, error)
	ValueToForm(value interface{}) interface{}
}

type FormValueOmitter interface {
	ValueOmittedFromData(data url.Values, files map[string][]io.ReadCloser, name string) bool
}

type FormValueGetter interface {
	ValueFromDataDict(data url.Values, files map[string][]io.ReadCloser, name string) (interface{}, []error)
}

type FormValuer interface {
	FormValueConverter
	FormValueOmitter
	FormValueGetter
}

type Widget interface {
	IsHidden() bool
	Hide(hidden bool)
	Type() string
	SetAttrs(attrs map[string]string)
	IdForLabel(id string) string
	GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context
	RenderWithErrors(w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error
	Render(w io.Writer, id, name string, value interface{}, attrs map[string]string) error

	FormValuer
	media.MediaDefiner
}
