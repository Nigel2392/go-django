package tracer

import (
	"fmt"
	"strings"
)

type StackTrace []Caller // A stacktrace is a slice of callers.

func (s StackTrace) String() string {
	var b strings.Builder
	var maxLenStart int
	var startSlice []string = make([]string, 0, len(s))
	for _, caller := range s {
		var start = fmt.Sprintf("Error on line %d:", caller.Line)
		startSlice = append(startSlice, start)
		if len(start) > maxLenStart {
			maxLenStart = len(start)
		}
	}
	var maxMiddleLen int
	var middleSlice []string = make([]string, 0, len(s))
	for _, caller := range s {
		var middle = fmt.Sprintf("%s()", filenameFromPath(caller.FunctionName))
		middleSlice = append(middleSlice, middle)
		if len(middle) > maxMiddleLen {
			maxMiddleLen = len(middle)
		}
	}

	for i, caller := range s {
		var start = startSlice[i]
		b.WriteString(start)
		if len(start) < maxLenStart {
			b.Grow(maxLenStart - len(start) + 3)
			for i := 0; i < maxLenStart-len(start); i++ {
				b.WriteString(" ")
			}
		}
		b.WriteString(" ")
		var middle = middleSlice[i]
		b.WriteString(middle)
		if len(middle) < maxMiddleLen {
			b.Grow(maxMiddleLen - len(middle) + 3)
			for i := 0; i < maxMiddleLen-len(middle); i++ {
				b.WriteString(" ")
			}
		}
		b.WriteString(" ")
		b.WriteString(cutFrontPath(caller.File, 40))
		b.WriteString("\n")
	}
	return b.String()
}

// Cut the front of a string, and add "..." if it was cut.
func cutFrontPath(s string, length int) string {
	if len(s) > length {
		var cut = len(s) - length
		s = s[cut:]
		var parts = strings.Split(s, "/")
		if len(parts) > 1 {
			return ".../" + strings.Join(parts[1:], "/")
		}
		return "..." + s
	}
	return s
}

// Get the filename from a path
func filenameFromPath(s string) string {
	s = strings.Replace(s, "\\", "/", -1)
	var sSplit = strings.Split(s, "/")
	return sSplit[len(sSplit)-1]
}

// Reverse the stacktrace.
func (s StackTrace) reverse() StackTrace {
	var r = make([]Caller, 0, len(s))
	for i := len(s) - 1; i >= 0; i-- {
		r = append(r, s[i])
	}
	return r
}

// Filter the stacktrace by caller.
func (s StackTrace) Filter(filter func(Caller) bool) []Caller {
	var r = make([]Caller, 0, len(s))
	for _, i := range s {
		if filter(i) {
			r = append(r, i)
		}
	}
	return r
}
