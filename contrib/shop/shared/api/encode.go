package api

import (
	"encoding/json"
	"net/http"
)

func Respond(w http.ResponseWriter, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
