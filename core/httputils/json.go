package httputils

import (
	"encoding/json"
	"strings"
)

// Jsonify converts a struct to json
func Jsonify(v interface{}, indent int) ([]byte, error) {
	if indent > 0 {
		return json.MarshalIndent(v, "", strings.Repeat(" ", indent))
	}
	return json.Marshal(v)
}

// UnJsonify converts json to a struct
func UnJsonify(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
