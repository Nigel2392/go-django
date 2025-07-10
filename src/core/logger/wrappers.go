package logger

import (
	"context"
	"fmt"
	"io"
	"strings"
)

var SUPPORTS_COLORS_DEFAULT = true

const (
	CMD_Green  = "\033[32m"
	CMD_Cyan   = "\033[36m"
	CMD_Yellow = "\033[33m"
	CMD_Red    = "\033[31m"
	CMD_Bold   = "\033[1m"
	CMD_Reset  = "\033[0m"
)

type supportsColorsKey struct{}

func SupportsColors(ctx context.Context) bool {
	var supports, ok = ctx.Value(supportsColorsKey{}).(bool)
	if !ok {
		return SUPPORTS_COLORS_DEFAULT
	}
	return supports
}

func WithSupportsColors(ctx context.Context, supports bool) context.Context {
	return context.WithValue(ctx, supportsColorsKey{}, supports)
}

func Colorize(ctx context.Context, color, s any) string {
	var c string
	switch color := color.(type) {
	case string:
		c = color
	case []string:
		c = strings.Join(color, "")
	default:
		panic(fmt.Sprintf("invalid color type: %T", color))
	}
	if !SupportsColors(ctx) {
		return fmt.Sprintf("%v", s)
	}
	return fmt.Sprintf("%s%v%s", c, s, CMD_Reset)
}

func FColorize(ctx context.Context, w io.Writer, color, s any) (n int, err error) {
	var str = Colorize(ctx, color, s)
	return fmt.Fprint(w, str)
}

func wrapLog(colors ...string) func(ctx context.Context, l LogLevel, s string) string {
	var s strings.Builder
	for _, color := range colors {
		s.WriteString(color)
	}
	var prefix = s.String()
	return func(ctx context.Context, l LogLevel, s string) string {
		return Colorize(ctx, prefix, s)
	}
}

func ColoredLogWrapper(ctx context.Context, l LogLevel, s string) string {
	var fn, ok = logWrapperMap[l]
	if !ok {
		return s
	}
	return fn(ctx, l, s)
}

var logWrapperMap = map[LogLevel]func(ctx context.Context, l LogLevel, s string) string{
	DBG:  wrapLog(CMD_Green),
	INF:  wrapLog(CMD_Cyan),
	WRN:  wrapLog(CMD_Yellow),
	ERR:  wrapLog(CMD_Red, CMD_Bold),
	CRIT: wrapLog(CMD_Red, CMD_Bold),
}
