//go:build windows

package writeerrors

import "syscall"

func brokenPipeErr() error {
	return syscall.ERROR_BROKEN_PIPE
}
