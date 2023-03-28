package logger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/tracer"
)

type LogEntry struct {
	// The time the log entry was created.
	Time time.Time
	// The level of the log entry.
	Level Loglevel
	// The message of the log entry.
	Message string
	// The tracer of the log entry.
	Stacktrace tracer.StackTrace
}

func NewLogEntry(level Loglevel, message string, stackTraceLen, skip int) *LogEntry {
	return &LogEntry{
		Time:       time.Now(),
		Level:      level,
		Message:    message,
		Stacktrace: tracer.Trace(errors.New(message), stackTraceLen, skip+1).Trace(),
	}
}

func (e *LogEntry) AsString(prefix string, colorized bool) string {
	var charAfterNewLineOrMultiLine bool
	var multiLine bool
	for _, c := range e.Message {
		if c == '\n' {
			multiLine = true
		}
		if multiLine && c != '\n' {
			charAfterNewLineOrMultiLine = true
			break
		}
	}
	var b = &strings.Builder{}
	if charAfterNewLineOrMultiLine {
		b.WriteString("[ ")
		if prefix != "" {
			writeIfColorized(b, colorized, prefix, DimGrey)
		}
		writeIfColorized(b, colorized, e.Level.String(), getLogLevelColor(e.Level))
		b.WriteString(" ] - ")
		writeIfColorized(b, colorized, e.Time.Format("2006-01-02 15:04:05"), DimGrey)
	} else {
		writeIfColorized(b, colorized, e.Time.Format("2006-01-02 15:04:05"), DimGrey, Bold)
		b.WriteString(" [ ")
		if prefix != "" {
			writeIfColorized(b, colorized, prefix, DimGrey)
		}
		writeIfColorized(b, colorized, e.Level.String(), getLogLevelColor(e.Level))
		b.WriteString(" ] - ")
	}
	if e.Message != "" {
		if charAfterNewLineOrMultiLine {
			b.WriteString("\n\n")
		}
		b.WriteString(e.Message)
	}

	// Write the stacktrace of the message.
	if e.Level > ERROR || e.Stacktrace == nil {
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString("\n\n")
	writeIfColorized(b, colorized, "Stacktrace:\n", Red, Underline)

	var maxLenStart int
	var startSlice []string = make([]string, 0, len(e.Stacktrace))
	for _, caller := range e.Stacktrace {
		var start = fmt.Sprintf("Error on line %d:", caller.Line)
		startSlice = append(startSlice, start)
		if len(start) > maxLenStart {
			maxLenStart = len(start)
		}
	}
	var maxMiddleLen int
	var middleSlice []string = make([]string, 0, len(e.Stacktrace))
	for _, caller := range e.Stacktrace {
		var middle = fmt.Sprintf("%s()", httputils.CutStart((httputils.FilenameFromPath(caller.FunctionName)), 40, ".", false))
		middleSlice = append(middleSlice, middle)
		if len(middle) > maxMiddleLen {
			maxMiddleLen = len(middle)
		}
	}
	for i, caller := range e.Stacktrace {
		var start = startSlice[i]
		writeIfColorized(b, colorized, start, Italics, DimGrey)

		if len(start) < maxLenStart {
			b.Grow(maxLenStart - len(start) + 1)
			for i := 0; i < maxLenStart-len(start); i++ {
				b.WriteString(" ")
			}
		}
		b.WriteString(" ")

		var middle = middleSlice[i]

		writeIfColorized(b, colorized, middle, Italics, Red)

		if len(middle) < maxMiddleLen {
			b.Grow(maxMiddleLen - len(middle) + 1)
			for i := 0; i < maxMiddleLen-len(middle); i++ {
				b.WriteString(" ")
			}
		}
		b.WriteString(" ")

		writeIfColorized(b, colorized, httputils.CutFrontPath(caller.File, 40), Italics, DimGrey)
		b.WriteString("\n")
	}

	// max length of a line
	var maxLen int
	for _, line := range strings.Split(b.String(), "\n") {
		line = DeColorize(line)
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	// Add a line at the beginning and end of the message.
	b.Grow(maxLen + 1)
	for i := 0; i < maxLen; i++ {
		b.WriteString("-")
	}
	b.WriteString("\n")

	//	var str = b.String()
	//	b.Reset()
	//	b.Grow(maxLen + 1 + len(str) + 2)
	//	for i := 0; i < maxLen; i++ {
	//		b.WriteString("-")
	//	}
	//	b.WriteString("\n")
	//	b.WriteString(str)
	//	b.WriteString("\n")

	return b.String()
}
