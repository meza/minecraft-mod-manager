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

func (l *Logger) Log(message string, forceShow bool) {
	if l.quiet && !forceShow && !l.debug {
		return
	}
	if _, err := fmt.Fprintln(l.out, message); err != nil {
		return
	}
}

func (l *Logger) Debug(message string) {
	if !l.debug {
		return
	}
	if _, err := fmt.Fprintln(l.out, message); err != nil {
		return
	}
}

func (l *Logger) Error(message string) {
	if _, err := fmt.Fprintln(l.err, message); err != nil {
		return
	}
}

func (l *Logger) Errorf(format string, args ...any) {
	if _, err := fmt.Fprintf(l.err, format, args...); err != nil {
		return
	}
}
