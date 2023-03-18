package logger

import (
	"fmt"
	"runtime"
	"time"
)

type Loglevel = int

const (
	CRITICAL Loglevel = iota
	ERROR
	WARNING
	INFO
	DEBUG
	TEST
)

type msgType int

const (
	msgTypeCritical msgType = iota
	msgTypeError
	msgTypeWarning
	msgTypeInfo
	msgTypeDebug
	msgTypeTest
)

type Logger struct {
	Loglevel Loglevel
	prefix   string
}

func NewLogger(loglevel Loglevel, prefix ...string) Logger {
	var l = Logger{
		Loglevel: loglevel,
	}
	if len(prefix) > 0 {
		l.prefix = prefix[0]
	}
	return l
}

func (l Logger) Critical(err error) {
	var info = GetCaller(1)
	l.logLine(msgTypeCritical, err.Error())
	for _, i := range info {
		l.logLine(msgTypeCritical, fmt.Sprintf("%s:%d", i.File, i.Line))
	}
}

func (l Logger) Error(args ...any) {
	l.logLine(msgTypeError, fmt.Sprint(args...))
}

func (l Logger) Errorf(format string, args ...any) {
	l.log(msgTypeError, fmt.Sprintf(format, args...))
}

func (l Logger) Warning(args ...any) {
	l.logLine(msgTypeWarning, fmt.Sprint(args...))
}

func (l Logger) Warningf(format string, args ...any) {
	l.log(msgTypeWarning, fmt.Sprintf(format, args...))
}

func (l Logger) Info(args ...any) {
	l.logLine(msgTypeInfo, fmt.Sprint(args...))
}

func (l Logger) Infof(format string, args ...any) {
	l.log(msgTypeInfo, fmt.Sprintf(format, args...))
}

func (l Logger) Debug(args ...any) {
	l.logLine(msgTypeDebug, fmt.Sprint(args...))
}

func (l Logger) Debugf(format string, args ...any) {
	l.log(msgTypeDebug, fmt.Sprintf(format, args...))
}

func (l Logger) Test(args ...any) {
	l.logLine(msgTypeTest, fmt.Sprint(args...))
}

func (l Logger) Testf(format string, args ...any) {
	l.log(msgTypeTest, fmt.Sprintf(format, args...))
}

func (l Logger) logLine(msgType msgType, msg string) {
	l.log(msgType, msg+"\n")
}

func (l Logger) log(msgType msgType, msg string) {
	if l.Loglevel >= int(msgType) {
		var prefix string = generatePrefix(l.prefix, msgType)
		fmt.Print(prefix + msg)
	}
}

func generatePrefix(prefix string, msgType msgType) string {
	var msg string
	switch msgType {
	case msgTypeCritical:
		msg = "[%sCRITICAL] "
	case msgTypeError:
		msg = "[%sERROR] "
	case msgTypeWarning:
		msg = "[%sWARNING] "
	case msgTypeInfo:
		msg = "[%sINFO] "
	case msgTypeDebug:
		msg = "[%sDEBUG] "
	case msgTypeTest:
		msg = "[%sTEST] "
	}
	msg = fmt.Sprintf(msg, prefix)
	msg = timestamp(msg)
	var color = getColor(msgType)
	msg = Colorize(color, msg)
	return msg
}

func timestamp(msg string) string {
	return fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func getColor(msgType msgType) string {
	var color string
	switch msgType {
	case msgTypeCritical:
		color = ColorLevelError
	case msgTypeError:
		color = ColorLevelError
	case msgTypeWarning:
		color = ColorLevelWarning
	case msgTypeInfo:
		color = ColorLevelInfo
	case msgTypeDebug:
		color = ColorLevelDebug
	case msgTypeTest:
		color = ColorLevelTest
	}
	return color
}

//	    // This message should be handled differently
//	    // than the other ways of reporting.
//	    Critical(err error)
//	    // Write an error message, loglevel error
//	    Error(args ...any)
//	    Errorf(format string, args ...any)
//	    // Write a warning message, loglevel warning
//	    Warning(args ...any)
//	    Warningf(format string, args ...any)
//	    // Write an info message, loglevel info
//	    Info(args ...any)
//	    Infof(format string, args ...any)
//	    // Write a debug message, loglevel debug
//	    Debug(args ...any)
//	    Debugf(format string, args ...any)
//	    // Write a test message, loglevel test
//	    Test(args ...any)
//	    Testf(format string, args ...any)

type CallerInfo struct {
	File string
	Line int
}

func GetCaller(skip int) []CallerInfo {
	skip += 2 // skip this function and the function that called this function
	pcs := make([]uintptr, 6)
	n := runtime.Callers(0, pcs)
	pcs = pcs[:n]

	frames := runtime.CallersFrames(pcs)
	var callFrames = make([]CallerInfo, 0, n)
	for {
		frame, more := frames.Next()
		callFrames = append(callFrames, CallerInfo{frame.File, frame.Line})
		if !more {
			break
		}
	}

	if skip > len(callFrames) {
		skip = len(callFrames)
	}
	return callFrames[skip:]
}
