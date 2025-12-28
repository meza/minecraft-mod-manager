//go:build !windows

package output

import "syscall"

func brokenPipeErr() error {
	return syscall.EPIPE
}
