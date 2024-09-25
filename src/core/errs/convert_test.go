package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/src/core/errs"
)

func TestConvert(t *testing.T) {

	t.Run("nil", func(t *testing.T) {
		err := errs.Convert(nil, nil)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("nil default", func(t *testing.T) {
		err := errs.Convert(nil, nil, "foo")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		err := errors.New("foo")
		got := errs.Convert(err, nil)
		if got != err {
			t.Errorf("expected %v, got %v", err, got)
		}
	})

	t.Run("string", func(t *testing.T) {
		err := "foo"
		got := errs.Convert(err, nil)
		if got.Error() != err {
			t.Errorf("expected %v, got %v", err, got)
		}
	})

	t.Run("string with args", func(t *testing.T) {
		err := "foo %s"
		arg := "bar"
		got := errs.Convert(err, nil, arg)
		if got.Error() != fmt.Sprintf(err, arg) {
			t.Errorf("expected %v, got %v", fmt.Sprintf(err, arg), got)
		}
	})

	t.Run("default", func(t *testing.T) {
		defaultErr := errors.New("default")
		got := errs.Convert(nil, defaultErr)
		if got != defaultErr {
			t.Errorf("expected %v, got %v", defaultErr, got)
		}
	})

	t.Run("default string", func(t *testing.T) {
		defaultErr := "default"
		got := errs.Convert(nil, defaultErr)
		if got.Error() != defaultErr {
			t.Errorf("expected %v, got %v", defaultErr, got)
		}
	})

	t.Run("default string with args", func(t *testing.T) {
		defaultErr := "default %s"
		arg := "bar"
		got := errs.Convert(nil, defaultErr, arg)
		if got.Error() != fmt.Sprintf(defaultErr, arg) {
			t.Errorf("expected %v, got %v", fmt.Sprintf(defaultErr, arg), got)
		}
	})
}
