package serializer

import (
	"html/template"

	"github.com/Nigel2392/go-django/core/views/interfaces"
)

type ObjectGetterFunc[T any] func(page, itemsPerPage int) (items []T, totalCount int64, err error)

type Model[T any] interface {
	interfaces.Saver
	interfaces.StringGetter[T]
	interfaces.Lister[T]
}

type UIContext struct {
	Paginator    *Paginator
	Object       interface{}
	Static       string
	Content      template.HTML
	Extra        template.HTML
	LimitOptions []int
	Index        string
	EscapeHTML   bool
}

type CSRFResponse struct {
	CSRFToken string `json:"csrf_token"`
	Detail    any    `json:"detail"`
}

type Paginator struct {
	Count        int           `json:"count,omitempty"`
	NumPages     int           `json:"num_pages"`
	Page         int           `json:"current_page"`
	ItemsPerPage int           `json:"limit"`
	Next         string        `json:"next"`
	Previous     string        `json:"previous"`
	Results      []interface{} `json:"results"`
}
