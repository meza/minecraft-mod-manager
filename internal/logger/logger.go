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
	_, _ = fmt.Fprintln(l.out, message)
}

func (l *Logger) Debug(message string) {
	if !l.debug {
		return
	}
	_, _ = fmt.Fprintln(l.out, message)
}

func (l *Logger) Error(message string) {
	_, _ = fmt.Fprintln(l.err, message)
}

func (l *Logger) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(l.err, format, args...)
}
