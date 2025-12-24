// Package logger provides structured logging helpers.
package logger

import (
	"fmt"
	"io"
)

type Logger struct {
	out   io.Writer
	err   io.Writer
	quiet bool
	debug bool
}

func New(out io.Writer, err io.Writer, quiet bool, debug bool) *Logger {
	return &Logger{
		out:   out,
		err:   err,
		quiet: quiet,
		debug: debug,
	}
}

func (logger *Logger) Log(message string, forceShow bool) {
	if logger.quiet && !forceShow && !logger.debug {
		return
	}
	if _, err := fmt.Fprintln(logger.out, message); err != nil {
		return
	}
}

func (logger *Logger) Debug(message string) {
	if !logger.debug {
		return
	}
	if _, err := fmt.Fprintln(logger.out, message); err != nil {
		return
	}
}

func (logger *Logger) Error(message string) {
	if _, err := fmt.Fprintln(logger.err, message); err != nil {
		return
	}
}

func (logger *Logger) Errorf(format string, args ...any) {
	if _, err := fmt.Fprintf(logger.err, format, args...); err != nil {
		return
	}
}
