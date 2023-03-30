package flag

import (
	"fmt"
	"io"
	"strings"

	"github.com/Nigel2392/go-django/core/logger"
	"golang.org/x/exp/slices"
)

// PrettyPrintUsage returns a function that prints the usage of the flag set
// to the given writer.
func PrettyPrintUsage(w io.Writer, f *Flags) func() {
	return func() {
		if f.Info == "" {
			colorizer(w, logger.Blue, "Usage of ")
		}
		colorizer(w, logger.Blue, "%s:\n", f.FlagSet.Name())
		if f.Info != "" {
			indentColorizer(w, 2, logger.White, "%s\n", newLineReplacer(f.Info, 2))
		}
		if f.Info != "" && len(f.Commands) > 0 {
			colorizer(w, logger.Blue, "Command-line flags:\n")
		}
		slices.SortFunc(f.Commands, func(a, b *Command) bool {
			return a.Name < b.Name
		})
		for _, cmd := range f.Commands {
			var hasDefault = isBool(cmd.Default) || !equalsNew(cmd.Default)
			var hasDescription = cmd.Description != ""
			if hasDefault || hasDescription {
				indentColorizer(w, 4, logger.BrightBlue, "-%s", cmd.Name)
				colorizer(w, logger.White, ":")
				fmt.Fprintf(w, "\n")
			} else {
				indentColorizer(w, 4, logger.BrightCyan, "-%s", cmd.Name)
			}
			if hasDefault {
				indentColorizer(w, 7, logger.BrightCyan, "Default: ")
				colorizer(w, logger.White, "%v %s\n", quoteString(cmd.Default), colorizeType(cmd.Default, "[", "]"))
			}
			if hasDescription {
				indentColorizer(w, 10, logger.BrightCyan, "Description: ")
				colorizer(w, logger.White, "%s\n", newLineReplacer(cmd.Description, 12))
			}
			fmt.Fprintln(w)
		}
	}
}

// If the value is a string, it will be quoted.
func quoteString(v any) string {
	if isString(v) {
		return fmt.Sprintf("\"%v\"", v)
	}
	return fmt.Sprintf("%v", v)
}

// Colorize the given string with the given color.
func colorizer(w io.Writer, color string, format string, args ...any) (err error) {
	if len(args) == 0 {
		_, err = fmt.Fprintf(w, "%s%s%s", color, format, logger.Reset)
	} else {
		_, err = fmt.Fprintf(w, "%s%s%s", color, fmt.Sprintf(format, args...), logger.Reset)
	}
	return err
}

// Colorize the given string with the given color and indent.
func indentColorizer(w io.Writer, indent int, color string, format string, args ...any) (err error) {
	if len(args) == 0 {
		_, err = fmt.Fprintf(w, "%s%s%s%s", strings.Repeat(" ", indent), color, format, logger.Reset)
	} else {
		_, err = fmt.Fprintf(w, "%s%s%s%s", strings.Repeat(" ", indent), color, fmt.Sprintf(format, args...), logger.Reset)
	}
	return err
}

// Replace all newlines with a newline and the given indent.
func newLineReplacer(s string, indent int) string {
	return strings.Replace(s, "\n", fmt.Sprintf("\n%s", strings.Repeat(" ", indent)), -1)
}

// Colorize the type of the given value.
func colorizeType(v any, pref, suff string) string {
	var b = &strings.Builder{}
	switch v.(type) {
	case string:
		b.WriteString(logger.Yellow)
		b.WriteString(pref)
		b.WriteString("string")
	case int, int8, int16, int32, int64:
		b.WriteString(logger.Green)
		b.WriteString(pref)
		b.WriteString("int")
	case uint, uint8, uint16, uint32, uint64:
		b.WriteString(logger.Green)
		b.WriteString(pref)
		b.WriteString("uint")
	case float32, float64:
		b.WriteString(logger.Green)
		b.WriteString(pref)
		b.WriteString("float")
	case bool:
		b.WriteString(logger.Purple)
		b.WriteString(pref)
		b.WriteString("bool")
	default:
		b.WriteString(logger.Blue)
		b.WriteString(pref)
		b.WriteString(strings.ToLower(fmt.Sprintf("%T", v)))
	}
	b.WriteString(suff)
	b.WriteString(logger.Reset)
	return b.String()
}
