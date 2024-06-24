package filters

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type MultiFilter[T any] []EntryFilter[T]

func (f MultiFilter[T]) Form(prefix string, r *http.Request, initial map[string]interface{}) FilterForm[T] {
	var forms = orderedmap.NewOrderedMap[string, FilterForm[T]]()
	for formIndex, filter := range f {
		var subPrefix = fmt.Sprintf(
			"%s-%d", prefix, formIndex,
		)

		var (
			subInitial, ok = initial[subPrefix].(map[string]interface{})
		)

		assert.True(
			ok, "MultiFilter[%T].Form: initial[%s] must be a map[string]interface{}",
			f, subPrefix,
		)

		forms.Set(
			subPrefix,
			filter.Form(
				subPrefix, r, subInitial,
			),
		)

	}
	return boundFilterForms[T]{
		forms:  forms,
		filter: f,
	}
}

type boundFilterForms[T any] struct {
	forms  *orderedmap.OrderedMap[string, FilterForm[T]]
	filter EntryFilter[T]
}

func (f boundFilterForms[T]) AsP() template.HTML {
	var html = make([]string, 0, f.forms.Len())
	for head := f.forms.Front(); head != nil; head = head.Next() {
		html = append(html, string(head.Value.AsP()))
	}
	return template.HTML(strings.Join(html, "\n"))
}

func (f boundFilterForms[T]) AsUL() template.HTML {
	var html = make([]string, 0, f.forms.Len())
	for head := f.forms.Front(); head != nil; head = head.Next() {
		html = append(html, string(head.Value.AsUL()))
	}
	return template.HTML(strings.Join(html, "\n"))
}

func (f boundFilterForms[T]) Media() media.Media {
	var media media.Media = media.NewMedia()
	for head := f.forms.Front(); head != nil; head = head.Next() {
		media = media.Merge(
			head.Value.Media(),
		)
	}
	return media
}

func (f boundFilterForms[T]) IsValid() bool {
	var valid = true
	for head := f.forms.Front(); head != nil; head = head.Next() {
		valid = valid && head.Value.IsValid()
	}
	return valid
}

func (f boundFilterForms[T]) EntryFilter(data []T) []T {
	for head := f.forms.Front(); head != nil; head = head.Next() {
		if head.Value.IsValid() {
			data = head.Value.EntryFilter(data)
		}
	}
	return data
}
