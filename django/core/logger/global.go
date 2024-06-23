package logger

import "io"

var (
	globalLogger Log
	Writer       func(level LogLevel) io.Writer
	PWriter      func(label string, level LogLevel) io.Writer
	NameSpace    func(label string) Log

	SetOutput func(level LogLevel, w io.Writer)
	SetLevel  func(level LogLevel)

	Debug  func(args ...interface{})
	Info   func(args ...interface{})
	Warn   func(args ...interface{})
	Error  func(args ...interface{})
	Fatal  func(errorcode int, args ...interface{})
	Debugf func(format string, args ...interface{})
	Infof  func(format string, args ...interface{})
	Warnf  func(format string, args ...interface{})
	Errorf func(format string, args ...interface{})
	Fatalf func(errorcode int, format string, args ...interface{})
	Logf   func(level LogLevel, format string, args ...interface{})
)

func Setup(logger Log) {
	globalLogger = logger

	Writer = globalLogger.Writer
	PWriter = globalLogger.PWriter
	NameSpace = globalLogger.NameSpace

	SetOutput = globalLogger.SetOutput
	SetLevel = globalLogger.SetLevel

	Debug = globalLogger.Debug
	Info = globalLogger.Info
	Warn = globalLogger.Warn
	Error = globalLogger.Error
	Fatal = globalLogger.Fatal

	Debugf = globalLogger.Debugf
	Infof = globalLogger.Infof
	Warnf = globalLogger.Warnf
	Errorf = globalLogger.Errorf
	Fatalf = globalLogger.Fatalf
	Logf = globalLogger.Logf
}
