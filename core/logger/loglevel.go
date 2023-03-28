package logger

type Loglevel int

const (
	CRITICAL Loglevel = iota + 1
	ERROR
	WARNING
	INFO
	DEBUG
	TEST
)

func (l Loglevel) String() string {
	switch l {
	case CRITICAL:
		return "CRITICAL"
	case ERROR:
		return "ERROR"
	case WARNING:
		return "WARNING"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	case TEST:
		return "TEST"
	default:
		return "UNKNOWN"
	}
}

// getLogLevelColor returns the color for a loglevel.
func getLogLevelColor(level Loglevel) string {
	switch level {
	case CRITICAL:
		return ColorLevelError + Underline
	case ERROR:
		return ColorLevelError
	case WARNING:
		return ColorLevelWarning
	case INFO:
		return ColorLevelInfo
	case DEBUG:
		return ColorLevelDebug
	case TEST:
		return ColorLevelTest
	}
	return ColorLevelInfo
}
