package widgets_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
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

func TestOptionsWidgetValidate(t *testing.T) {
	var opts = []widgets.Option{
		widgets.NewOption("name1", "label1", "value1"),
		widgets.NewOption("name2", "label2", "value2"),
		widgets.NewOption("name3", "label3", "value3"),
	}

	var w = options.NewSelectInput(map[string]string{}, func() []widgets.Option {
		return opts
	})

	t.Run("valid", func(t *testing.T) {
		var errorList = w.Validate("value1")
		if len(errorList) != 0 {
			t.Errorf("expected %d, got %d", 0, len(errorList))
		}
	})

	t.Run("invalid", func(t *testing.T) {
		var errorList = w.Validate("invalid")

		if len(errorList) != 1 {
			t.Errorf("expected %d, got %d", 1, len(errorList))
		}

		if !errors.Is(errorList[0], errs.ErrInvalidValue) {
			t.Errorf("expected %v, got %v", errs.ErrInvalidValue, errorList[0])
		}
	})
}
