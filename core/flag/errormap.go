package flag

import "fmt"

// ErrorMap is a map of errors.
// Errors for commands can be stored in this map.
type ErrorMap map[string]error

// Error returns the error string.
func (e ErrorMap) Error() string {
	var errStr = ""
	for k, v := range e {
		errStr += fmt.Sprintf("%s: %s\n", k, v.Error())
	}
	return errStr
}

// Has checks if the error map has an error for the given key.
func (e ErrorMap) Has(s string) bool {
	_, ok := e[s]
	return ok
}
