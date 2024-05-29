package widgets_test

import (
	"net/url"
	"testing"

	"github.com/Nigel2392/django/forms/widgets"
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
