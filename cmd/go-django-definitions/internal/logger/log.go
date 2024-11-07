package logger

type Logger interface {
	Log(message string)
	Logf(format string, args ...interface{})
}

var DefaultLogger Logger

func Log(message string) {
	DefaultLogger.Log(message)
}

func Logf(format string, args ...interface{}) {
	DefaultLogger.Logf(format, args...)
}
