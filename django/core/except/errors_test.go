package except_test

import (
	"errors"
	"testing"

	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/except"
)

func TestHttpError(t *testing.T) {

	t.Run("Is", func(t *testing.T) {

		t.Run("TopLevelIs", func(t *testing.T) {
			var (
				httpError = except.ServerError(404, "Not Found")
				newErr    = except.ServerError(404, "Not Found")
			)

			if !errors.Is(httpError, newErr) {
				t.Errorf("Expected %v, got %v", newErr, httpError)
			}
		})

		t.Run("NestedIs", func(t *testing.T) {
			var (
				httpError404 = except.ServerError(404, "Not Found")
				httpError500 = except.ServerError(500, "Internal Server Error")
				newErr400    = except.ServerError(404, "Not Found")
				newErr500    = except.ServerError(500, "Internal Server Error")
				multiErr     = errs.NewMultiError(
					errs.Wrap(
						httpError404,
						"nested error",
					),
					httpError500,
				)
			)

			if !errors.Is(multiErr, newErr400) {
				t.Errorf("Expected \"%v\", got \"%v\"", newErr400, multiErr)
			}

			if !errors.Is(multiErr, newErr500) {
				t.Errorf("Expected \"%v\", got \"%v\"", newErr500, multiErr)
			}
		})
	})

	t.Run("As", func(t *testing.T) {

		t.Run("TopLevelAs", func(t *testing.T) {
			var (
				httpError = except.ServerError(404, "Not Found")
				newErr    = &except.HttpError{}
			)

			if !errors.As(httpError, &newErr) {
				t.Errorf("Expected %v, got %v", newErr, httpError)
			}

			if newErr.Code != 404 {
				t.Errorf("Expected 404, got %v", newErr.Code)
			}

			if newErr.Message.Error() != "Not Found" {
				t.Errorf("Expected \"Not Found\", got \"%v\"", newErr.Message)
			}
		})

		t.Run("NestedAs", func(t *testing.T) {
			var (
				httpError404 = except.ServerError(404, "Not Found")
				httpError500 = except.ServerError(500, "Internal Server Error")
				newErr400    = &except.HttpError{}
				newErr500    = &except.HttpError{}
				multiErr     = errs.NewMultiError(
					errs.Wrap(
						httpError404,
						"nested error",
					),
					httpError500,
				)
			)

			if !errors.As(multiErr, &newErr400) {
				t.Errorf("Expected \"%v\", got \"%v\"", newErr400, multiErr)
			}

			if !errors.As(multiErr, &newErr500) {
				t.Errorf("Expected \"%v\", got \"%v\"", newErr500, multiErr)
			}

			if newErr400.Code != 404 {
				t.Errorf("Expected 404, got %v", newErr400.Code)
			}

			if newErr400.Message.Error() != "Not Found" {
				t.Errorf("Expected \"Not Found\", got \"%v\"", newErr400.Message)
			}
		})
	})
}
