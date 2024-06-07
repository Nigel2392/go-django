package utils

import "strings"

func Trunc(s string, length int) string {
	if len(s) > length {
		var b strings.Builder
		b.WriteString(s[:length-3])
		b.WriteString("...")
		return b.String()
	}
	return s
}
