package httpclient

import (
	"errors"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

type timeoutNetErrorForConnection struct{}

func (timeoutNetErrorForConnection) Error() string   { return "timeout network error" }
func (timeoutNetErrorForConnection) Timeout() bool   { return true }
func (timeoutNetErrorForConnection) Temporary() bool { return false }

type nonTimeoutNetErrorForConnection struct{}

func (nonTimeoutNetErrorForConnection) Error() string   { return "non-timeout network error" }
func (nonTimeoutNetErrorForConnection) Timeout() bool   { return false }
func (nonTimeoutNetErrorForConnection) Temporary() bool { return false }

type syscallMatchError struct{}

func (syscallMatchError) Error() string { return "syscall match error" }
func (syscallMatchError) Is(target error) bool {
	return target == syscall.ECONNREFUSED
}

func TestIsConnectionErrorHandlesTimeoutNetError(t *testing.T) {
	assert.True(t, IsConnectionError(timeoutNetErrorForConnection{}))
}

func TestIsConnectionErrorIgnoresNonTimeoutNetError(t *testing.T) {
	assert.False(t, IsConnectionError(nonTimeoutNetErrorForConnection{}))
}

func TestIsConnectionErrorHandlesDNSError(t *testing.T) {
	assert.True(t, IsConnectionError(&net.DNSError{Err: "no such host"}))
}

func TestIsConnectionErrorHandlesOpErrorSyscall(t *testing.T) {
	opErr := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNRESET}
	assert.True(t, IsConnectionError(opErr))
}

func TestIsConnectionErrorHandlesOpErrorNonTransient(t *testing.T) {
	opErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("boom")}
	assert.False(t, IsConnectionError(opErr))
}

func TestIsConnectionErrorHandlesURLErrorSyscall(t *testing.T) {
	urlErr := &url.Error{Op: "Get", URL: "https://example.invalid", Err: syscall.ECONNREFUSED}
	assert.True(t, IsConnectionError(urlErr))
}

func TestIsConnectionErrorIgnoresURLErrorWithoutInnerError(t *testing.T) {
	urlErr := &url.Error{Op: "Get", URL: "https://example.invalid"}
	assert.False(t, IsConnectionError(urlErr))
}

func TestIsConnectionErrorHandlesSyscallError(t *testing.T) {
	syscallErr := &os.SyscallError{Syscall: "dial", Err: syscall.ETIMEDOUT}
	var matched *os.SyscallError
	assert.True(t, errors.As(syscallErr, &matched))
	assert.Equal(t, syscallErr, matched)
	assert.True(t, IsConnectionError(syscallErr))
}

func TestIsConnectionErrorHandlesSyscallErrorWithCustomMatcher(t *testing.T) {
	syscallErr := &os.SyscallError{Syscall: "dial", Err: syscallMatchError{}}
	assert.True(t, IsConnectionError(syscallErr))
}

func TestIsConnectionErrorHandlesWrappedSyscallError(t *testing.T) {
	wrapped := &url.Error{
		Op:  "Get",
		URL: "https://example.invalid",
		Err: &os.SyscallError{Syscall: "dial", Err: syscall.ETIMEDOUT},
	}
	assert.True(t, IsConnectionError(wrapped))
}

func TestIsConnectionErrorHandlesDirectSyscallError(t *testing.T) {
	assert.True(t, IsConnectionError(syscall.EHOSTUNREACH))
}

func TestIsConnectionErrorHandlesDirectNetworkUnreachable(t *testing.T) {
	assert.True(t, IsConnectionError(syscall.ENETUNREACH))
}

func TestIsConnectionErrorReturnsFalseForNil(t *testing.T) {
	assert.False(t, IsConnectionError(nil))
}

func TestIsConnectionErrorReturnsFalseForUnknown(t *testing.T) {
	assert.False(t, IsConnectionError(errors.New("boom")))
}

func TestIsTransientSyscallErrorHandlesNil(t *testing.T) {
	assert.False(t, isTransientSyscallError(nil))
}

func TestIsTransientSyscallErrorHandlesNonTransient(t *testing.T) {
	assert.False(t, isTransientSyscallError(errors.New("boom")))
}

func TestIsTransientSyscallErrorHandlesTimeout(t *testing.T) {
	assert.True(t, isTransientSyscallError(syscall.ETIMEDOUT))
}
