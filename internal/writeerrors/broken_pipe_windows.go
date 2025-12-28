//go:build windows

package writeerrors

import (
	"errors"
	"io"
	"syscall"
)

// IsBrokenPipe reports whether err represents a broken pipe on Windows.
func IsBrokenPipe(err error) bool {
	return errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.ERROR_BROKEN_PIPE)
}
