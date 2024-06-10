package tracer

import (
	"fmt"
	"strings"
)

// Error is an error with stack trace.
// It consists of callers and the error supplied.
type ErrorType interface {
	Error() string
	Trace() StackTrace
	Unwrap() error
}

type Callers struct {
	Callers StackTrace
	err     error
}

func (c *Callers) String() string {
	var b strings.Builder
	for _, i := range c.Callers {
		b.WriteString(fmt.Sprintf("%s:%d %s\n", i.File, i.Line, i.FunctionName))
	}
	return b.String()
}

func (c *Callers) Error() string {
	return c.err.Error()
}

func (c *Callers) Unwrap() error {
	return c.err
}

func (c *Callers) Trace() StackTrace {
	return c.Callers
}

//	// Sort the stacktrace in reverse.
//	// Sorts the slice in place.
//	func (c *Callers) Reverse() Error {
//		var r = make([]Caller, 0, len(c.Callers))
//		for i := len(c.Callers) - 1; i >= 0; i-- {
//			r = append(r, c.Callers[i])
//		}
//		return r
//	}
//
//	func (c Callers) Filter(filter func(Caller) bool) []Caller {
//		var r = make([]Caller, 0, len(c.Callers))
//		for _, i := range c.Callers {
//			if filter(i) {
//				r = append(r, i)
//			}
//		}
//		return r
//	}
//
