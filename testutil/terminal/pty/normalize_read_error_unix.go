//go:build !windows

package pty

import (
	"errors"
	"os"
	"syscall"
)

func normalizeReadError(err error) error {
	if err == nil || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EIO) {
		return nil
	}
	return err
}
