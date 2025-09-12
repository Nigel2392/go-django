package text

import (
	gofmt "fmt"
	"strings"
)

func JoinFormat(sep string, fmt string, params ...[]any) string {
	switch len(params) {
	case 0:
		return ""
	case 1:
		return gofmt.Sprintf(fmt, params[0]...)
	}

	var b strings.Builder
	b.Grow(len(fmt) * len(params))
	b.Grow(len(sep) * (len(params) - 1))

	for i, elem := range params {
		if i > 0 {
			b.WriteString(sep)
		}
		gofmt.Fprintf(&b, fmt, elem...)
	}

	return b.String()
}
