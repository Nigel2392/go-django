package multiselect

import (
	"context"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
)

type MultiSelectWidget struct {
	*options.OptionsWidget
}

func NewMultiSelectInput(attrs map[string]string, choices func(ctx context.Context) []widgets.Option) widgets.Widget {
	return &MultiSelectWidget{
		OptionsWidget: options.NewOptionWidget("select", "forms/widgets/multi-select.html", attrs, choices),
	}
}

func (m *MultiSelectWidget) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var values, ok = data[name]
	if !ok {
		return nil, nil
	}
	return values, nil
}

func (m *MultiSelectWidget) Media() media.Media {
	var formMedia = media.NewMedia()
	formMedia.AddCSS(media.CSS(django.Static(
		"forms/css/multiple-select.css",
	)))
	formMedia.AddJS(&media.JSAsset{
		URL: django.Static(
			"forms/js/index.js",
		),
	})
	return formMedia
}
