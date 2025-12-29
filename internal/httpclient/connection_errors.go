package httpclient

import (
	"errors"
	"net"
	"net/url"
	"os"
	"syscall"
)

// IsConnectionError reports whether the error looks like a transient connectivity failure.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		err = urlErr.Err
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) && isTransientSyscallError(opErr.Err) {
		return true
	}

	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) && isTransientSyscallError(syscallErr.Err) {
		return true
	}

	return isTransientSyscallError(err)
}

func isTransientSyscallError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.ETIMEDOUT)
}
