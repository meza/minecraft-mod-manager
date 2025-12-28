//go:build !windows

package writeerrors

import (
	"errors"
	"io"
	"syscall"
)

// IsBrokenPipe reports whether err represents a broken pipe on Unix-like systems.
func IsBrokenPipe(err error) bool {
	return errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE)
}
