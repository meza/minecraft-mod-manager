//go:build windows

package pty

import (
	"errors"
	"os"
)

func normalizeReadError(err error) error {
	if err == nil || errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}
