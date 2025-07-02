package logger

import (
	"fmt"
	"io"
	"os"
)

func printArgs(levelStr string, format string, args ...interface{}) {
	//	if format == "" {
	//		fmt.Printf("[%s] %s\n", levelStr, fmt.Sprint(args...))
	//		return
	//	}
	//	fmt.Printf("[%s] %s\n", levelStr, fmt.Sprintf(format, args...))
}

type prefixerwriter struct {
	label    string
	logLevel LogLevel
	out      io.Writer
}

func (p prefixerwriter) Write(b []byte) (n int, err error) {
	if p.label != "" {
		return fmt.Fprintf(p.out, "[%s] %s", p.label, string(b))
	}
	return fmt.Fprintf(p.out, "[%s] %s", p.logLevel.String(), string(b))
}

var (
	defaultLogLevel = INF
	Writer          = func(level LogLevel) io.Writer { return os.Stdout }
	PWriter         = func(label string, level LogLevel) io.Writer { return prefixerwriter{label, level, os.Stdout} }
	NameSpace       = func(label string) Log { return nil }

	SetOutput func(level LogLevel, w io.Writer) = func(level LogLevel, w io.Writer) {}
	SetLevel  func(level LogLevel)              = func(level LogLevel) { defaultLogLevel = level }
	GetLevel  func() (level LogLevel)           = func() (level LogLevel) { return defaultLogLevel }

	Debug  = func(args ...interface{}) { printArgs("DEBUG", "", args...) }
	Info   = func(args ...interface{}) { printArgs("INFO", "", args...) }
	Warn   = func(args ...interface{}) { printArgs("WARN", "", args...) }
	Error  = func(args ...interface{}) { printArgs("ERROR", "", args...) }
	Fatal  = func(errorcode int, args ...interface{}) { printArgs("FATAL", "", args...) }
	Debugf = func(format string, args ...interface{}) { printArgs("DEBUG", format, args...) }
	Infof  = func(format string, args ...interface{}) { printArgs("INFO", format, args...) }
	Warnf  = func(format string, args ...interface{}) { printArgs("WARN", format, args...) }
	Errorf = func(format string, args ...interface{}) { printArgs("ERROR", format, args...) }
	Fatalf = func(errorcode int, format string, args ...interface{}) { printArgs("FATAL", format, args...) }
	Logf   = func(level LogLevel, format string, args ...interface{}) {
		printArgs(level.String(), format, args...)
	}
)

func Setup(logger Log) {
	Writer = logger.Writer
	PWriter = logger.PWriter
	NameSpace = logger.NameSpace

	SetOutput = logger.SetOutput
	SetLevel = logger.SetLevel
	GetLevel = logger.GetLevel

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
