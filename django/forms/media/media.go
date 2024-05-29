package media

import (
	"html/template"

	"github.com/elliotchance/orderedmap/v2"
)

type Media interface {
	Merge(other Media) Media
	JS() []template.HTML
	CSS() []template.HTML
}

type MediaDefiner interface {
	Media() Media
}

type MediaObject struct {
	Css *orderedmap.OrderedMap[string, template.HTML]
	Js  *orderedmap.OrderedMap[string, template.HTML]
}

func NewMedia() *MediaObject {
	return &MediaObject{
		Css: orderedmap.NewOrderedMap[string, template.HTML](),
		Js:  orderedmap.NewOrderedMap[string, template.HTML](),
	}
}

func (m *MediaObject) Merge(other Media) Media {
	if other == nil {
		return m
	}

	var (
		otherCss = other.CSS()
		otherJs  = other.JS()
	)

	for _, v := range otherCss {
		var strV = string(v)
		if _, ok := m.Css.Get(strV); !ok {
			m.Css.Set(strV, v)
		}
	}

	for _, v := range otherJs {
		var strV = string(v)
		if _, ok := m.Js.Get(strV); !ok {
			m.Js.Set(strV, v)
		}
	}

	return m
}

func (m *MediaObject) JS() []template.HTML {
	var ret = make([]template.HTML, 0, m.Js.Len())
	for head := m.Js.Front(); head != nil; head = head.Next() {
		ret = append(ret, head.Value)
	}
	return ret
}

func (m *MediaObject) CSS() []template.HTML {
	var ret = make([]template.HTML, 0, m.Css.Len())
	for head := m.Css.Front(); head != nil; head = head.Next() {
		ret = append(ret, head.Value)
	}
	return ret
}
