package output

import (
	"fmt"
	"io"

	"github.com/meza/minecraft-mod-manager/internal/writeerrors"
)

type Output struct {
	out   io.Writer
	err   io.Writer
	quiet bool
}

type LogVisibility int

const (
	LogQuiet LogVisibility = iota
	LogForce
)

func New(out io.Writer, err io.Writer, quiet bool) *Output {
	return &Output{
		out:   out,
		err:   err,
		quiet: quiet,
	}
}

func (output *Output) Log(message string, visibility LogVisibility) error {
	if output.quiet && visibility != LogForce {
		return nil
	}
	_, err := fmt.Fprintln(output.out, message)
	return normalizeWriteError(err)
}

func (output *Output) Error(message string) error {
	_, err := fmt.Fprintln(output.err, message)
	return normalizeWriteError(err)
}

func (output *Output) Errorf(format string, args ...any) error {
	_, err := fmt.Fprintf(output.err, format, args...)
	return normalizeWriteError(err)
}

func normalizeWriteError(err error) error {
	if err == nil || writeerrors.IsBrokenPipe(err) {
		return nil
	}
	return err
}
