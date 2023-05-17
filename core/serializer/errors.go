package serializer

import (
	"encoding/json"
	"errors"
	"io"
)

var ErrNotLister = errors.New("object does not implement interfaces.Lister")

type ErrorResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

func (s *ErrorResponse) AddError(err error) {
	s.Errors = append(s.Errors, err.Error())
}

func (s *ErrorResponse) WriteTo(w io.Writer) (int64, error) {
	var data, err = json.MarshalIndent(s, "", "  ")
	if err != nil {
		return 0, err
	}
	var n, err2 = w.Write(data)
	return int64(n), err2
}

func NewErrorResponse(code int, message string, errors ...error) *ErrorResponse {
	var errs = make([]string, len(errors))
	for i, err := range errors {
		errs[i] = err.Error()
	}
	return &ErrorResponse{
		Code:    code,
		Message: message,
		Errors:  errs,
	}
}
