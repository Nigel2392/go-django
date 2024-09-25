package widgets_test

import (
	"testing"

	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func TestIntNumberValueToGo(t *testing.T) {
	var (
		w          = widgets.NewNumberInput[int](nil)
		value, err = w.ValueToGo("100")
	)
	if err != nil {
		t.Errorf("expected %v, got %v", nil, err)
	}
	if value != 100 {
		t.Errorf("expected %d, got %d", 1, value)
	}
}

func TestIntNumberValueToForm(t *testing.T) {
	var (
		w     = widgets.NewNumberInput[int](nil)
		value = w.ValueToForm(100)
	)
	if value != "100" {
		t.Errorf("expected %d, got %d", 1, value)
	}
}

func TestFloatNumberValueToGo(t *testing.T) {
	var (
		w          = widgets.NewNumberInput[float64](nil)
		value, err = w.ValueToGo("100.1")
	)
	if err != nil {
		t.Errorf("expected %v, got %v", nil, err)
	}
	if value != 100.1 {
		t.Errorf("expected %f, got %f", 100.1, value)
	}
}

func TestFloatNumberValueToForm(t *testing.T) {
	var (
		w     = widgets.NewNumberInput[float64](nil)
		value = w.ValueToForm(100.1)
	)
	if value != "100.1" {
		t.Errorf("expected %f, got %f", 100.1, value)
	}
}

func TestFloatNumberValueToFormFromInt(t *testing.T) {
	var (
		w     = widgets.NewNumberInput[float64](nil)
		value = w.ValueToForm(100)
	)
	if value != "100" {
		t.Errorf("expected %v, got %v", 100, value)
	}
}

func TestFloatNumberValueToGoFail(t *testing.T) {
	var (
		w      = widgets.NewNumberInput[int](nil)
		_, err = w.ValueToGo("100.1")
	)

	if err == nil {
		t.Errorf("expected %v, got %v", "error", err)
	}
}

func TestFloatNumberValueToFormFail(t *testing.T) {
	var (
		w     = widgets.NewNumberInput[int](nil)
		value = w.ValueToForm(100.1)
	)
	if value != "100" {
		t.Errorf("expected %d, got %v", 100, value)
	}
}
