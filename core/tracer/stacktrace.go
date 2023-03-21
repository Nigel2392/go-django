package tracer

import (
	"strconv"
	"strings"
)

type StackTrace []Caller

func (s StackTrace) String() string {
	var b strings.Builder
	for _, i := range s {
		b.WriteString("Error on line ")
		b.WriteString(strconv.Itoa(i.Line))
		b.WriteString(": ")
		b.WriteString(i.File)
		b.WriteString(i.FunctionName)
		b.WriteString("()\n")
	}
	return b.String()
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
