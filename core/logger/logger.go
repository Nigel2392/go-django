package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Nigel2392/go-django/core/tracer"
)

func NewLogFile(filename string) (*os.File, error) {
	var file *os.File
	var err error
	if _, err = os.Stat(filename); os.IsNotExist(err) {
		var dir = filepath.Dir(filename)
		os.MkdirAll(dir, os.ModePerm)
		file, err = os.Create(filename)
	} else if err == nil {
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	return file, err
}

type Logger struct {
	Loglevel Loglevel
	prefix   string
	File     io.Writer
}

func NewLogger(loglevel Loglevel, w io.Writer, prefix ...string) Logger {
	var l = Logger{
		Loglevel: loglevel,
		File:     w,
	}
	if len(prefix) > 0 {
		l.prefix = prefix[0]
	}
	return l
}

func (l *Logger) Critical(err error) {
	var t = tracer.TraceSafe(err, 6, 1)
	l.logLine(CRITICAL, err.Error())
	for _, i := range t.Trace() {
		l.logLine(CRITICAL, fmt.Sprintf("%s:%d", i.File, i.Line))
	}
}

// Write an error message, loglevel error
func (l *Logger) Error(args ...any) {
	l.logLine(ERROR, fmt.Sprint(args...))
}

// Write an error message, loglevel error
func (l *Logger) Errorf(format string, args ...any) {
	l.logLine(ERROR, fmt.Sprintf(format, args...))
}

// Write a warning message, loglevel warning
func (l *Logger) Warning(args ...any) {
	l.logLine(WARNING, fmt.Sprint(args...))
}

// Write a warning message, loglevel warning
func (l *Logger) Warningf(format string, args ...any) {
	l.logLine(WARNING, fmt.Sprintf(format, args...))
}

// Write an info message, loglevel info
func (l *Logger) Info(args ...any) {
	l.logLine(INFO, fmt.Sprint(args...))
}

// Write an info message, loglevel info
func (l *Logger) Infof(format string, args ...any) {
	l.logLine(INFO, fmt.Sprintf(format, args...))
}

// Write a debug message, loglevel debug
func (l *Logger) Debug(args ...any) {
	l.logLine(DEBUG, fmt.Sprint(args...))
}

// Write a debug message, loglevel debug
func (l *Logger) Debugf(format string, args ...any) {
	l.logLine(DEBUG, fmt.Sprintf(format, args...))
}

// Write a test message, loglevel test
func (l *Logger) Test(args ...any) {
	l.logLine(TEST, fmt.Sprint(args...))
}

// Write a test message, loglevel test
func (l *Logger) Testf(format string, args ...any) {
	l.logLine(TEST, fmt.Sprintf(format, args...))
}

func (l *Logger) logLine(level Loglevel, msg string) {
	l.log(level, msg+"\n")
}

func (l *Logger) log(msgType Loglevel, msg string) {
	if l.Loglevel >= Loglevel(msgType) {
		fmt.Fprintf(l.File, "%s%s", generatePrefix(true, l.prefix, msgType), msg)
	}
}

func generatePrefix(colorized bool, prefix string, level Loglevel) string {
	var msg string
	msg = "[%s%s] "
	msg = fmt.Sprintf(msg, prefix, level.String())
	msg = timestamp(msg)
	if colorized {
		var color = getLogLevelColor(level)
		msg = Colorize(msg, color)
	}
	return msg
}

func timestamp(msg string) string {
	return fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), msg)
}
