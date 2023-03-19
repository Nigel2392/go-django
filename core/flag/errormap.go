package flag

import "fmt"

type ErrorMap map[string]error

func (e ErrorMap) Error() string {
	var errStr = ""
	for k, v := range e {
		errStr += fmt.Sprintf("%s: %s\n", k, v.Error())
	}
	return errStr
}

func (e ErrorMap) Has(s string) bool {
	_, ok := e[s]
	return ok
}
