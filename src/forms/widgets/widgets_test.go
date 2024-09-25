package widgets_test

import (
	"net/url"
	"testing"

	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func TestValueFromDataDict(t *testing.T) {
	var (
		w           = widgets.NewTextInput(map[string]string{})
		data        = url.Values{"field": []string{"value"}}
		value, errs = w.ValueFromDataDict(data, nil, "field")
	)
	if len(errs) != 0 {
		t.Errorf("expected %d, got %d", 0, len(errs))
	}
	if value != "value" {
		t.Errorf("expected %q, got %q", "value", value)
	}
}

func TestValueOmittedFromData(t *testing.T) {
	var (
		d = url.Values{}
		w = widgets.NewTextInput(map[string]string{})
		v = w.ValueOmittedFromData(d, nil, "field")
	)
	t.Run("empty", func(t *testing.T) {
		if v != true {
			t.Errorf("expected %v, got %v", true, v)
		}
	})

	t.Run("value", func(t *testing.T) {
		d.Set("field", "value")
		v = w.ValueOmittedFromData(d, nil, "field")
		if v != false {
			t.Errorf("expected %v, got %v", false, v)
		}
	})
}
