package tracer

import (
	"runtime"
)

type callerFunc struct {
	Name string
	Line int
}

var STACKLOGGER_ALLOWED_FILES map[string][]*callerFunc = make(map[string][]*callerFunc)
var STACKLOGGER_DISALLOWED_FILES map[string][]*callerFunc = make(map[string][]*callerFunc)
var STACKLOGGER_UNSAFE bool = true

func Trace(err error, stackLen, skip int) ErrorType {
	if stackLen <= 0 || skip < 0 {
		return nil
	}
	if stackLen > 32 {
		stackLen = 32
	}
	if skip > 30 {
		skip = 30
	}
	skip += 2 // skip this function and the function that called this function
	pcs := make([]uintptr, stackLen+skip)
	n := runtime.Callers(0, pcs)
	pcs = pcs[:n]

	frames := runtime.CallersFrames(pcs)
	var callFrames = make(StackTrace, 0, n)
	var frame runtime.Frame
	var more bool
	for {
		frame, more = frames.Next()
		callFrames = append(callFrames, Caller{
			File:         frame.File,
			Line:         frame.Line,
			FunctionName: frame.Func.Name(),
		})
		if !more {
			break
		}
	}

	if skip > len(callFrames) {
		skip = len(callFrames)
	}

	return &Callers{
		Callers: callFrames[skip:].reverse(),
		err:     err,
	}
}

// Get a stacktrace of the current goroutine.
//
// stackLen is the number of stack frames to get.
//
// skip is the number of stack frames to skip.
//
// It will be default skip the first two frames,
// this means it will return out the function that called this function last.
func TraceSafe(err error, stackLen, skip int) ErrorType {
	var frames = Trace(err, stackLen, skip+1)
	var newFrames = make([]Caller, 0, len(frames.Trace()))
	for _, frame := range frames.Trace() {
		if frame.isAllowed(STACKLOGGER_ALLOWED_FILES) {
			newFrames = append(newFrames, frame)
		} else if !frame.isDisallowed(STACKLOGGER_DISALLOWED_FILES) {
			newFrames = append(newFrames, frame)
		} else {
			if STACKLOGGER_UNSAFE {
				newFrames = append(newFrames, frame)
			}
		}
	}
	return &Callers{
		Callers: newFrames,
		err:     err,
	}
}
