package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/src/core/errs"
)

type TestError struct {
	Name    string
	Message string
}

func (e TestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

func TestMultiError(t *testing.T) {

	var (
		Err1 = errs.Error("error 1")
		Err2 = errs.Error("error 2")
		Err3 = errs.Error("error 3")

		Err4 = TestError{Name: "test", Message: "error 4"}
		Err5 = TestError{Name: "test", Message: "error 5"}
		Err6 = TestError{Name: "test", Message: "error 6"}
	)

	t.Run("Is", func(t *testing.T) {
		var multiError = errs.NewMultiError(
			Err1, Err2, Err3,
		)

		if !errors.Is(multiError, Err1) {
			t.Errorf("Expected %v, got %v", Err1, multiError)
		}

		if !errors.Is(multiError, Err2) {
			t.Errorf("Expected %v, got %v", Err2, multiError)
		}

		if !errors.Is(multiError, Err3) {
			t.Errorf("Expected %v, got %v", Err3, multiError)
		}
	})

	t.Run("As", func(t *testing.T) {
		var multiError = errs.NewMultiError(
			Err4, Err5, Err6,
		)

		var target TestError
		if !errors.As(multiError, &target) {
			t.Errorf("Expected %v, got %v", target, multiError)
		}
	})

	t.Run("Error", func(t *testing.T) {
		var multiError = errs.NewMultiError(
			Err1, Err2, Err3,
		)

		var expected = "error 1: error 2: error 3"
		if multiError.Error() != expected {
			t.Errorf("Expected %v, got %v", expected, multiError.Error())
		}
	})
}
