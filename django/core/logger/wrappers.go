package logger

import (
	"fmt"
	"strings"
)

const (
	CMD_Green  = "\033[32m"
	CMD_Cyan   = "\033[36m"
	CMD_Yellow = "\033[33m"
	CMD_Red    = "\033[31m"
	CMD_Bold   = "\033[1m"
	CMD_Reset  = "\033[0m"
)

func Colorize(color, s any) string {
	return fmt.Sprintf("%s%v%s", color, s, CMD_Reset)
}

func wrapLog(colors ...string) func(l LogLevel, s string) string {
	var s strings.Builder
	for _, color := range colors {
		s.WriteString(color)
	}
	var prefix = s.String()
	return func(l LogLevel, s string) string {
		return Colorize(prefix, s)
	}
}

func ColoredLogWrapper(l LogLevel, s string) string {
	var fn, ok = logWrapperMap[l]
	if !ok {
		return s
	}
	return fn(l, s)
}

var logWrapperMap = map[LogLevel]func(l LogLevel, s string) string{
	DBG: wrapLog(CMD_Green),
	INF: wrapLog(CMD_Cyan),
	WRN: wrapLog(CMD_Yellow),
	ERR: wrapLog(CMD_Red, CMD_Bold),
}
