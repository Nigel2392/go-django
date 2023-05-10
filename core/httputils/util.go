package httputils

import (
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var TitleCaser = cases.Title(language.English)

func MustInt(s string, min, max, dflt int) int {
	var i, err = strconv.Atoi(s)
	if err != nil {
		return dflt
	}
	if i < min {
		return min
	}
	if i > max {
		return max
	}
	return i
}

func FormatLabel(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var last rune
	for _, r := range s {
		if s == "_" {
			b.WriteRune(' ')
			last = ' '
			continue
		}
		if r >= 'A' && r <= 'Z' {
			if last >= 'a' && last <= 'z' {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
		last = r
	}
	return TitleCaser.String(b.String())
}
