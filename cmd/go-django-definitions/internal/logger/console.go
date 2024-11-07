//go:build debug
// +build debug

package logger

import (
	"fmt"
	"os"
	"runtime"
)

var stdout *os.File

func init() {
	var err error
	switch runtime.GOOS {
	case "windows":
		stdout, err = os.OpenFile("CONOUT$", os.O_RDWR, 0)
	case "linux", "darwin":
		stdout, err = os.OpenFile("/dev/stdout", os.O_RDWR, 0)
	}

	if err != nil {
		panic(err)
	}

	DefaultLogger = &ConsoleLogger{}
}

type ConsoleLogger struct{}

func (l *ConsoleLogger) Log(message string) {
	fmt.Fprintln(stdout, message)
}

func (l *ConsoleLogger) Logf(format string, args ...interface{}) {
	fmt.Fprintf(stdout, format, args...)
}
