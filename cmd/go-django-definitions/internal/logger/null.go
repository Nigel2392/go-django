//go:build !debug
// +build !debug

package logger

func init() {
	DefaultLogger = &NullLogger{}
}

type NullLogger struct{}

func (l *NullLogger) Log(message string) {}

func (l *NullLogger) Logf(format string, args ...interface{}) {}
