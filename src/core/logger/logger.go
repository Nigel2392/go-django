package logger

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type LogLevel int8

func (l LogLevel) String() string {
	return levelMap[l]
}

var (
	ErrOutputInvalid = errors.New("output is invalid")

	levelMap = map[LogLevel]string{
		DBG: "DEBUG",
		INF: "INFO",
		WRN: "WARN",
		ERR: "ERROR",
	}

	revMap = map[string]LogLevel{
		levelMap[DBG]: DBG,
		levelMap[INF]: INF,
		levelMap[WRN]: WRN,
		levelMap[ERR]: ERR,
	}
)

const (
	// DBG is the lowest log level.
	DBG LogLevel = iota

	// INF is the default log level.
	INF

	// WRN is used for warnings.
	WRN

	// ERR is used for errors.
	ERR

	// OutputAll is used to output all log levels in the SetOutput function.
	OutputAll LogLevel = -1
)

func ParseLogLevel(s string) (LogLevel, error) {
	s = strings.ToUpper(s)
	var l, ok = revMap[s]
	if !ok {
		return INF, fmt.Errorf("unknown log level %q", s)
	}
	return l, nil
}

type LogWriter struct {
	Logger *Logger
	Level  LogLevel
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	if lw.Level >= lw.Logger.Level {
		var out = lw.Logger.Output(lw.Level)
		lw.Logger.writePrefix(lw.Level, out)
		n, err = out.Write(p)
		lw.Logger.writeSuffix(out)
	}
	return
}

type Log interface {
	// Getters

	// Writer returns a new io.Writer that writes to the log output for the given log level.
	Writer(level LogLevel) io.Writer

	// PWriter returns a new io.Writer that writes to the log output for the given log level with a prefix.
	PWriter(label string, level LogLevel) io.Writer

	// NameSpace returns a new Log with the given label as the prefix.
	NameSpace(label string) Log

	// Setters

	// SetOutput sets the output for the given log level.
	SetOutput(level LogLevel, w io.Writer)

	// SetLevel sets the current log level.
	//
	// Log messages with a log level lower than the current log level will not be written.
	SetLevel(level LogLevel)

	// Logging

	// Log a debug message.
	Debug(args ...interface{})

	// Log an info message.
	Info(args ...interface{})

	// Log a warning message.
	Warn(args ...interface{})

	// Log an error message.
	Error(args ...interface{})

	// Log a message and exit the program with the given error code.
	Fatal(errorcode int, args ...interface{})

	// Log a format- and args-based debug message.
	Debugf(format string, args ...interface{})

	// Log a format- and args-based info message.
	Infof(format string, args ...interface{})

	// Log a format- and args-based warning message.
	Warnf(format string, args ...interface{})

	// Log a format- and args-based error message.
	Errorf(format string, args ...interface{})

	// Log a format- and args-based message and exit the program with the given error code.
	Fatalf(errorcode int, format string, args ...interface{})

	// Log a message at the given log level.
	Log(level LogLevel, args ...interface{})

	// Log a format string at the given log level.
	Logf(level LogLevel, format string, args ...interface{})

	// WriteString writes a string to the log output.
	WriteString(s string) (n int, err error)
}

// The Logger type is used to log messages at different log levels.
//
// It is possible to set the log level, prefix, suffix, and outputs for the log messages.
//
// When a log message is written, the prefix, log level, and suffix are written to the output
// provided for that specific loglevel.
type Logger struct {
	// Level is the log level.
	Level LogLevel

	// Prefix is the prefix for each log message.
	Prefix string

	// Suffix is the suffix for each log message.
	Suffix string

	// Display a timestamp alongside the log message.
	OutputTime bool

	// Outputs for the log messages.
	OutputDebug io.Writer
	OutputInfo  io.Writer
	OutputWarn  io.Writer
	OutputError io.Writer

	// WrapPrefix determines how the prefix should be wrapped
	// based on the LogLevel.
	WrapPrefix func(LogLevel, string) string
}

// SetOutput sets the output for the given log level.
func (l *Logger) SetOutput(level LogLevel, w io.Writer) {
	switch level {
	case DBG:
		l.OutputDebug = w
	case INF:
		l.OutputInfo = w
	case WRN:
		l.OutputWarn = w
	case ERR:
		l.OutputError = w
	case OutputAll:
		l.OutputDebug = w
		l.OutputInfo = w
		l.OutputWarn = w
		l.OutputError = w
	}
}

// validateOutputs ensures that the outputs are not nil.
//
// If an output is nil before a lower importance output, it will be set to io.Discard.
//
// If an output is nil after a higher importance output, the output will be set to the higher importance output.
func (l *Logger) validateOutputs() {
	if l.OutputDebug == nil {
		l.OutputDebug = io.Discard
	}
	if l.OutputInfo == nil {
		l.OutputInfo = l.OutputDebug
	}
	if l.OutputWarn == nil {
		l.OutputWarn = l.OutputInfo
	}
	if l.OutputError == nil {
		l.OutputError = l.OutputWarn
	}
}

// Returns the output for the given log level.
func (l *Logger) Output(level LogLevel) io.Writer {
	l.validateOutputs()
	switch level {
	case DBG:
		return l.OutputDebug
	case INF:
		return l.OutputInfo
	case WRN:
		return l.OutputWarn
	case ERR:
		return l.OutputError
	}
	if l.Level < DBG {
		return l.OutputDebug
	}
	if l.Level > ERR {
		return l.OutputError
	}
	return nil
}

// SetLevel sets the current log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.Level = level
}

// Copy returns a copy of the logger.
func (l *Logger) Copy() *Logger {
	return &Logger{
		Level:       l.Level,
		Prefix:      l.Prefix,
		Suffix:      l.Suffix,
		OutputTime:  l.OutputTime,
		WrapPrefix:  l.WrapPrefix,
		OutputDebug: l.OutputDebug,
		OutputInfo:  l.OutputInfo,
		OutputWarn:  l.OutputWarn,
		OutputError: l.OutputError,
	}
}

// NameSpace returns a new Log with the given label as the prefix.
func (l *Logger) NameSpace(label string) Log {
	var logger = l.Copy()
	if l.Prefix != "" {
		label = fmt.Sprintf("%s / %s", l.Prefix, label)
	}
	logger.Prefix = label
	return logger
}

// Writer returns a new io.Writer that writes to the log output for the given log level.
func (l *Logger) Writer(level LogLevel) io.Writer {
	return &LogWriter{
		Logger: l.Copy(),
		Level:  level,
	}
}

// PWriter returns a new io.Writer that writes to the log output for the given log level with a prefix.
func (l *Logger) PWriter(label string, level LogLevel) io.Writer {

	if l.Prefix != "" {
		label = fmt.Sprintf("%s / %s", l.Prefix, label)
	}

	var lw = &LogWriter{
		Logger: l.Copy(),
		Level:  level,
	}
	lw.Logger.Prefix = label
	return lw
}

func (l *Logger) Debug(args ...interface{}) {
	if l.Level <= DBG {
		l.log(DBG, args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	if l.Level <= INF {
		l.log(INF, args...)
	}
}

func (l *Logger) Warn(args ...interface{}) {
	if l.Level <= WRN {
		l.log(WRN, args...)
	}
}

func (l *Logger) Error(args ...interface{}) {
	if l.Level <= ERR {
		l.log(ERR, args...)
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.Level <= DBG {
		l.logf(DBG, format, args...)
	}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	if l.Level <= INF {
		l.logf(INF, format, args...)
	}
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.Level <= WRN {
		l.logf(WRN, format, args...)
	}
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.Level <= ERR {
		l.logf(ERR, format, args...)
	}
}

// Fatal is a convenience function for logging an error and exiting the program.
func (l *Logger) Fatal(errorcode int, args ...interface{}) {
	l.Error(args...)
	os.Exit(errorcode)
}

// Fatalf is a convenience function for logging an error and exiting the program.
func (l *Logger) Fatalf(errorcode int, format string, args ...interface{}) {
	l.Errorf(format, args...)
	os.Exit(errorcode)
}

func (l *Logger) Log(level LogLevel, args ...interface{}) {
	l.log(level, args...)
}

func (l *Logger) Logf(level LogLevel, format string, args ...interface{}) {
	l.logf(level, format, args...)
}

func (l *Logger) WriteString(s string) (n int, err error) {
	if l.Level <= DBG {
		return l.log(DBG, s)
	}
	if l.Level <= INF {
		return l.log(INF, s)
	}
	if l.Level <= WRN {
		return l.log(WRN, s)
	}
	if l.Level <= ERR {
		return l.log(ERR, s)
	}
	return l.log(INF, s)
}

func (l *Logger) writePrefix(level LogLevel, w io.Writer) {
	var b = new(bytes.Buffer)

	_, _ = b.Write([]byte("["))
	if l.Prefix != "" {
		_, _ = b.Write([]byte(l.Prefix))
		_, _ = b.Write([]byte(" / "))
	}

	_, _ = b.Write([]byte(level.String()))

	if l.OutputTime {
		_, _ = b.Write([]byte(" / "))
		var t = time.Now().Format("2006-01-02 15:04:05")
		_, _ = b.Write([]byte(t))
	}

	_, _ = b.Write([]byte("]: "))

	var prefix = b.String()
	if l.WrapPrefix != nil {
		prefix = l.WrapPrefix(level, prefix)
	}

	_, _ = w.Write([]byte(prefix))
}

func (l *Logger) writeSuffix(w io.Writer) {
	if l.Suffix != "" {
		_, _ = w.Write([]byte(" "))
		_, _ = w.Write([]byte(l.Suffix))
	}
}

var mu = new(sync.Mutex)

func (l *Logger) log(level LogLevel, args ...interface{}) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	var out = l.Output(level)
	if out == nil {
		return 0, ErrOutputInvalid
	}

	var b = new(bytes.Buffer)
	l.writePrefix(level, b)
	fmt.Fprint(b, args...)
	l.writeSuffix(b)

	var message = b.String()
	if l.WrapPrefix != nil {
		message = l.WrapPrefix(level, message)
	}

	var (
		n, i int
		err  error
	)
	i, err = out.Write(
		[]byte(message),
	)
	n += i
	if err != nil {
		return n, err
	}

	i, err = out.Write([]byte("\n"))
	return n + i, err
}

func (l *Logger) logf(level LogLevel, format string, args ...interface{}) {
	l.log(level, fmt.Sprintf(format, args...))
}
