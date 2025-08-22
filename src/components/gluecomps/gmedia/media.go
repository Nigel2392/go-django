package gmedia

import (
	"context"
	"io"

	"github.com/Nigel2392/go-django/src/components"
	"github.com/Nigel2392/go-django/src/components/gluecomps"
	"github.com/Nigel2392/go-django/src/forms/media"
)

type Media struct {
	media.Media
}

func NewMedia(source media.Media) *Media {
	if source == nil {
		source = media.NewMedia()
	}
	return &Media{
		Media: source,
	}
}

func (m *Media) Merge(other media.Media) {
	if m.Media == nil {
		m.Media = media.NewMedia()
	}
	m.Media = m.Media.Merge(other)
}

func (m *Media) Render(ctx context.Context, w io.Writer) error {
	for _, css := range m.Media.CSS() {
		w.Write([]byte(css))
	}
	for _, js := range m.Media.JS() {
		w.Write([]byte(js))
	}
	return nil
}

func (m *Media) CSS() components.Component {
	return gluecomps.Func(func(ctx context.Context, w io.Writer) error {
		for _, css := range m.Media.CSS() {
			w.Write([]byte(css))
		}
		return nil
	})
}

func (m *Media) JS() components.Component {
	return gluecomps.Func(func(ctx context.Context, w io.Writer) error {
		for _, js := range m.Media.JS() {
			w.Write([]byte(js))
		}
		return nil
	})
}
