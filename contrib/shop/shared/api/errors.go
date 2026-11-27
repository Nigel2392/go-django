package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/except"
)

type ErrorResponse struct {
	Response
	Message string `json:"message"`
}

func RespondError(w http.ResponseWriter, statusCode int, message string, args ...any) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	enc := json.NewEncoder(w)
	err := enc.Encode(ErrorResponse{
		StatusCode: statusCode,
		Message:    message,
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the error response: %v", err,
		)
	}
}
