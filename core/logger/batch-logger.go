package logger

import (
	"fmt"
	"io"
	"time"

	"github.com/Nigel2392/batch/accumulator"
	"github.com/Nigel2392/router/v3/request"
)

// BatchLogger is a logger which logs messages in batches.
//
// This logger is useful when you want to log a large number of messages or need to pool log messages.
//
// This does not guarantee order of messages!
type BatchLogger struct {
	// The prefix of the logger.
	Prefix string

	// The log level of the logger.
	Loglevel Loglevel

	// Colorize is a flag which determines whether the log entry is written colorized.
	Colorize bool

	// Handler is a function which determines how the log entry is handled.
	Handler func(entry *LogEntry, stdout io.Writer)

	// File is the file to write to.
	File io.Writer

	// The batcher which is used to batch the log entries.
	batcher *accumulator.Accumulator[*LogEntry]
}

// NewBatchLogger creates a new Logger.
func NewBatchLogger(loglevel Loglevel, flushSize int, flushInterval time.Duration, file io.Writer, prefix ...string) *BatchLogger {
	var p string
	if len(prefix) > 0 {
		p = prefix[0]
	}
	var logger = &BatchLogger{
		Prefix:   p,
		Loglevel: loglevel,
		File:     file,
	}
	logger.batcher = accumulator.NewAccumulator(flushSize, flushInterval, logger.handle)
	return logger
}

// Loglevel returns the loglevel of the logger.
func (l *BatchLogger) LogLevel() request.LogLevel {
	return request.LogLevel(l.Loglevel)
}

// Critical logs a critical message.
func (l *BatchLogger) Critical(e error) {
	l.log(CRITICAL, e.Error())
}

// Criticalf logs a critical message with a format.
func (l *BatchLogger) Criticalf(format string, args ...any) {
	l.log(CRITICAL, fmt.Sprintf(format, args...))
}

// Write an error message, loglevel error
func (l *BatchLogger) Error(args ...any) {
	l.log(ERROR, fmt.Sprint(args...))
}

// Write an error message, loglevel error
//
// Format the message in the fmt package format.
func (l *BatchLogger) Errorf(format string, args ...any) {
	l.log(ERROR, fmt.Sprintf(format, args...))
}

// Write a warning message, loglevel warning
func (l *BatchLogger) Warning(args ...any) {
	l.log(WARNING, fmt.Sprint(args...))
}

// Write a warning message, loglevel warning
//
// Format the message in the fmt package format.
func (l *BatchLogger) Warningf(format string, args ...any) {
	l.log(WARNING, fmt.Sprintf(format, args...))
}

// Write an info message, loglevel info
func (l *BatchLogger) Info(args ...any) {
	l.log(INFO, fmt.Sprint(args...))
}

// Write an info message, loglevel info
//
// Format the message in the fmt package format.
func (l *BatchLogger) Infof(format string, args ...any) {
	l.log(INFO, fmt.Sprintf(format, args...))
}

// Write a debug message, loglevel debug
func (l *BatchLogger) Debug(args ...any) {
	l.log(DEBUG, fmt.Sprint(args...))
}

// Write a debug message, loglevel debug
//
// Format the message in the fmt package format.
func (l *BatchLogger) Debugf(format string, args ...any) {
	l.log(DEBUG, fmt.Sprintf(format, args...))
}

// Write a test message, loglevel test
func (l *BatchLogger) Test(args ...any) {
	l.log(TEST, fmt.Sprint(args...))
}

// Write a test message, loglevel test
//
// Format the message in the fmt package format.
func (l *BatchLogger) Testf(format string, args ...any) {
	l.log(TEST, fmt.Sprintf(format, args...))
}

// Write a message instantly with the given loglevel.
func (l *BatchLogger) Now(loglevel Loglevel, format string, args ...any) {
	if l.Loglevel < loglevel {
		return
	}
	var entry = NewLogEntry(loglevel, fmt.Sprintf(format, args...), 8, 1)
	l.handle(entry)
}

func (l *BatchLogger) log(loglevel Loglevel, message string) {
	if l.Loglevel < loglevel {
		return
	}

	var entry = NewLogEntry(loglevel, message, 8, 1)

	l.batcher.Push(entry)
}

func (l *BatchLogger) handle(entry *LogEntry) {
	if l.Handler != nil {
		l.Handler(entry, l.File)
		return
	}
	l.write(entry)
}

// log logs a log entry.
func (l *BatchLogger) write(entry *LogEntry) error {
	// Write to file.
	if l.File != nil {
		var _, err = l.File.Write([]byte(entry.AsString(l.Prefix, l.Colorize)))
		if err != nil {
			return err
		}
	}
	return nil
}
