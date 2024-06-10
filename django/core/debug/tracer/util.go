package tracer

import (
	"regexp"
)

func contains(f []string, o string) bool {

	for _, v := range f {
		var reg, err = regexp.Compile(v)
		if err != nil {
			continue
		}
		if reg.MatchString(o) {
			return true
		}
	}

	for _, v := range f {
		if v == o {
			return true
		}
	}
	return false
}
