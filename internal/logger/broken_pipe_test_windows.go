//go:build windows

package logger

import "syscall"

func brokenPipeErr() error {
	return syscall.ERROR_BROKEN_PIPE
}
