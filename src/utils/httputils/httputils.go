package httputils

import (
	"encoding/json"
	"net/http"
)

// Returns a JSON error response with the following structure:
//
//	{
//	  "status": "error",
//	  "message": "Error message"
//	}
//
// It will also set the HTTP status code to the provided status code.
func JSONHttpError(w http.ResponseWriter, errStr string, status int) {
	var jsonResp = map[string]interface{}{
		"status":  "error",
		"message": errStr,
	}

	w.WriteHeader(status)

	var err = json.NewEncoder(w).Encode(jsonResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
