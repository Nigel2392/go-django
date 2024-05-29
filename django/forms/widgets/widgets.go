package widgets

import (
	"html/template"
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
	SetAttrs(attrs map[string]string)
	IdForLabel(id string) string
	GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context
	RenderWithErrors(id, name string, value interface{}, errors []error, attrs map[string]string) (template.HTML, error)
	Render(id, name string, value interface{}, attrs map[string]string) (template.HTML, error)

	FormValuer
	media.MediaDefiner
}
