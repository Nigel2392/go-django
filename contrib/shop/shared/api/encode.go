package api

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	StatusCode int `json:"status_code"`
}

type DataResponse[DATA any] struct {
	Response `json:",inline"`
	Data     DATA `json:"data,omitempty"`
}

func Respond[T any](w http.ResponseWriter, data T) error {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(DataResponse[T]{
		StatusCode: 200,
		Data:       data,
	})
}
