package strich

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/src/core/trans"
)

type Check[CHECK ~string] interface {
	Is(haystack CHECK, needle ...CHECK) bool
	IsAny(haystack CHECK, needle ...CHECK) bool
	StringOf(ctx context.Context, haystack CHECK) string
}

type CheckerDisplay[CHECK ~string] struct {
	Delimiter     string
	NotFoundLabel func(context.Context, CHECK) string
	Labels        map[CHECK]func(context.Context) string
}

type Checker[CHECK ~string] struct {
	// if delimiter is set, it is possible to use multiple flags.
	Delimiter rune

	Display CheckerDisplay[CHECK]
}

func notFoundLabel[CHECK ~string](ctx context.Context, _ CHECK) string {
	return trans.T(ctx, "Unknown")
}

func (s Checker[CHECK]) StringOf(ctx context.Context, haystack CHECK) string {
	var notFound func(ctx context.Context, _ CHECK) string
	if s.Display.NotFoundLabel == nil {
		notFound = notFoundLabel
	} else {
		notFound = s.Display.NotFoundLabel
	}

	if s.Delimiter == 0 {
		t, ok := s.Display.Labels[haystack]
		if !ok {
			return notFound(ctx, haystack)
		}

		return t(ctx)
	}

	t, ok := s.Display.Labels[haystack]
	if ok {
		return t(ctx)
	}

	var (
		secStart int
		delims   int
		sb       strings.Builder
	)
	for idx, char := range haystack {
		if char == s.Delimiter && secStart < len(haystack) {
			delims++

			var text string
			var section = haystack[secStart:idx]
			var match, ok = s.Display.Labels[section]
			if !ok {
				text = notFound(ctx, section)
			} else {
				text = match(ctx)
			}

			if secStart > 0 {
				sb.WriteString(s.Display.Delimiter)
			}

			sb.WriteString(text)

			secStart = idx + 1
		}
	}

	if delims > 0 && secStart < len(haystack) {
		var text string
		var section = haystack[secStart:]
		var match, ok = s.Display.Labels[section]
		if !ok {
			text = notFound(ctx, section)
		} else {
			text = match(ctx)
		}

		sb.WriteString(s.Display.Delimiter)
		sb.WriteString(text)
	}

	if secStart == 0 {
		sb.WriteString(notFound(
			ctx, haystack,
		))
	}

	return sb.String()
}

func (s Checker[CHECK]) Is(haystack CHECK, needle ...CHECK) bool {
	f, _ := s.contains(haystack, needle)
	return f == len(needle)
}

func (s Checker[CHECK]) IsAny(haystack CHECK, needle ...CHECK) bool {
	f, _ := s.contains(haystack, needle)
	return f > 0 || len(needle) == 0
}

func (s Checker[CHECK]) contains(haystack CHECK, needle []CHECK) (found, delims int) {

	var needles = make(map[CHECK]struct{})
	for _, n := range needle {
		needles[n] = struct{}{}
	}

	if _, ok := needles[haystack]; ok {
		return 1, 0
	}

	if s.Delimiter == 0 {
		return 0, 0
	}

	var secStart int
	for idx, char := range haystack {
		if char == s.Delimiter && secStart < len(haystack) {
			delims++

			_, ok := needles[haystack[secStart:idx]]
			if ok {
				found++
			}

			secStart = idx + 1
		}
	}

	if delims > 0 && secStart < len(haystack) {
		_, ok := needles[haystack[secStart:]]
		if ok {
			found++
		}
	}

	return found, delims
}
