package text

import (
	"strings"
)

func Trunc(s string, length int) string {
	return NewTruncator(length).Trunc(s)
}

type Truncator struct {
	Length int
}

func NewTruncator(length int) *Truncator {
	return &Truncator{Length: length}
}

func (t *Truncator) Trunc(s string) string {
	if len(s) > t.Length {
		var b strings.Builder
		b.WriteString(s[:t.Length-3])
		b.WriteString("...")
		return b.String()
	}
	return s
}
