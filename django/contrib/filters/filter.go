package filters

import (
	"html/template"
	"net/http"

	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
)

var _ Filter[any] = (*Sieve[any])(nil)

type BoundSieve[T any] struct {
	Form  forms.Form
	Sieve *Sieve[T]
}

func (b *BoundSieve[T]) AsP() template.HTML {
	return b.Form.AsP()
}

func (b *BoundSieve[T]) AsUL() template.HTML {
	return b.Form.AsUL()
}

func (b *BoundSieve[T]) Media() media.Media {
	return b.Form.Media()
}

func (b *BoundSieve[T]) IsValid() bool {
	return b.Form.IsValid()
}

func (b *BoundSieve[T]) Filter(data []T) []T {
	var result = make([]T, 0, len(data))
	for _, item := range data {
		if b.Sieve.Fn(item) {
			result = append(result, item)
		}
	}
	return result
}

type Sieve[T any] struct {
	Fields []fields.Field
	Fn     func(T) bool
}

func (s *Sieve[T]) Form(prefix string, r *http.Request, initial map[string]interface{}) FilterForm[T] {
	var form = forms.NewBaseForm(
		forms.WithPrefix(prefix),
		forms.WithInitial(initial),
		forms.WithRequestData("POST", r),
		forms.WithFields(s.Fields...),
	)

	return &BoundSieve[T]{
		Form:  form,
		Sieve: s,
	}
}
