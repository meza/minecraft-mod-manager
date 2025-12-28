// Package logger provides diagnostic logging helpers.
package logger

import (
	"fmt"
	"io"

	"github.com/meza/minecraft-mod-manager/internal/writeerrors"
)

type Logger struct {
	out   io.Writer
	err   io.Writer
	quiet bool
	debug bool
}

type LogVisibility int

const (
	LogQuiet LogVisibility = iota
	LogForce
)

func New(out io.Writer, err io.Writer, quiet bool, debug bool) *Logger {
	return &Logger{
		out:   out,
		err:   err,
		quiet: quiet,
		debug: debug,
	}
}

func (logger *Logger) Log(message string, visibility LogVisibility) error {
	if logger.quiet && visibility != LogForce && !logger.debug {
		return nil
	}
	_, err := fmt.Fprintln(logger.out, message)
	return normalizeWriteError(err)
}

func (logger *Logger) Debug(message string) error {
	if !logger.debug {
		return nil
	}
	_, err := fmt.Fprintln(logger.out, message)
	return normalizeWriteError(err)
}

func (logger *Logger) Error(message string) error {
	_, err := fmt.Fprintln(logger.err, message)
	return normalizeWriteError(err)
}

func (logger *Logger) Errorf(format string, args ...any) error {
	_, err := fmt.Fprintf(logger.err, format, args...)
	return normalizeWriteError(err)
}

func normalizeWriteError(err error) error {
	if err == nil || writeerrors.IsBrokenPipe(err) {
		return nil
	}
	return err
}
