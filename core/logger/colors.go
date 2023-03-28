package logger

import (
	"regexp"
	"strings"
	"unsafe"
)

// ANSI color codes
const (
	Italics   string = "\x1b[3m"
	Underline string = "\x1b[4m"
	Blink     string = "\x1b[5m"
	Bold      string = "\x1b[1m"

	Reset string = "\033[0m"

	Red    string = "\033[31m"
	Green  string = "\033[32m"
	Yellow string = "\033[33m"
	Blue   string = "\033[34m"
	Purple string = "\033[35m"
	Cyan   string = "\033[36m"
	White  string = "\033[37m"
	Grey   string = "\033[90m"

	BrightRed    string = "\033[31;1m"
	BrightGreen  string = "\033[32;1m"
	BrightYellow string = "\033[33;1m"
	BrightBlue   string = "\033[34;1m"
	BrightPurple string = "\033[35;1m"
	BrightCyan   string = "\033[36;1m"
	BrightGrey   string = "\033[37;1m"

	DimRed    string = "\033[31;2m"
	DimGreen  string = "\033[32;2m"
	DimYellow string = "\033[33;2m"
	DimBlue   string = "\033[34;2m"
	DimPurple string = "\033[35;2m"
	DimCyan   string = "\033[36;2m"
	DimGrey   string = "\033[37;2m"
)

// Preset colors for use in the logger's Colorize function
var (
	// LogTest
	ColorLevelTest = Purple
	// LogDebug
	ColorLevelDebug = Green
	// LogInfo
	ColorLevelInfo = Blue
	// LogWarn
	ColorLevelWarning = Yellow
	// LogErr
	ColorLevelError = Red
	// No level, default switch case opt.
	ColorNoLevel = Green
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var remAnsiRex = regexp.MustCompile(ansi)

// Colorize a message.
func Colorize(msg string, colors ...string) string {
	var maxSize int = len(msg) + len(Reset)
	for _, c := range colors {
		maxSize += len(c)
	}
	var b = make([]byte, maxSize)
	var n int = 0
	for _, c := range colors {
		n += copy(b[n:], c)
	}
	n += copy(b[n:], msg)
	n += copy(b[n:], Reset)
	return unsafe.String(unsafe.SliceData(b), n)
}

// Remove all ANSI color codes from a string.
func DeColorize(str string) string {
	return remAnsiRex.ReplaceAllString(str, "")
}

// Helper function to write a string to a string builder, optionally colorizing it.
func writeIfColorized(b *strings.Builder, colorized bool, text string, color ...string) {
	if colorized {
		var colorStr = Colorize(text, color...)
		b.WriteString(colorStr)
		return
	}
	b.WriteString(text)
}
