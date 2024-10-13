package logger

import (
	"io"
	"os"
)

var (
	Writer    = func(level LogLevel) io.Writer { return os.Stdout }
	PWriter   = func(label string, level LogLevel) io.Writer { return os.Stdout }
	NameSpace = func(label string) Log { return nil }

	SetOutput func(level LogLevel, w io.Writer) = func(level LogLevel, w io.Writer) {}
	SetLevel  func(level LogLevel)              = func(level LogLevel) {}

	Debug  = func(args ...interface{}) {}
	Info   = func(args ...interface{}) {}
	Warn   = func(args ...interface{}) {}
	Error  = func(args ...interface{}) {}
	Fatal  = func(errorcode int, args ...interface{}) {}
	Debugf = func(format string, args ...interface{}) {}
	Infof  = func(format string, args ...interface{}) {}
	Warnf  = func(format string, args ...interface{}) {}
	Errorf = func(format string, args ...interface{}) {}
	Fatalf = func(errorcode int, format string, args ...interface{}) {}
	Logf   = func(level LogLevel, format string, args ...interface{}) {}
)

func Setup(logger Log) {
	Writer = logger.Writer
	PWriter = logger.PWriter
	NameSpace = logger.NameSpace

	SetOutput = logger.SetOutput
	SetLevel = logger.SetLevel

	Debug = logger.Debug
	Info = logger.Info
	Warn = logger.Warn
	Error = logger.Error
	Fatal = logger.Fatal

	Debugf = logger.Debugf
	Infof = logger.Infof
	Warnf = logger.Warnf
	Errorf = logger.Errorf
	Fatalf = logger.Fatalf
	Logf = logger.Logf
}
