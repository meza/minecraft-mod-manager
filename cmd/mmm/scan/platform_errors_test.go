package scan

import (
	"context"
	"net/url"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
)

func TestAllowPlatformFallbackAllowsNil(t *testing.T) {
	assert.True(t, allowPlatformFallback(nil))
}

func TestAllowPlatformFallbackBlocksTimeout(t *testing.T) {
	assert.False(t, allowPlatformFallback(httpclient.WrapTimeoutError(context.DeadlineExceeded)))
}

func TestAllowPlatformFallbackBlocksConnectionError(t *testing.T) {
	connErr := &url.Error{Op: "Get", URL: "https://example.invalid", Err: syscall.ECONNRESET}
	assert.False(t, allowPlatformFallback(connErr))
}

func TestAllowPlatformFallbackAllowsOtherErrors(t *testing.T) {
	assert.True(t, allowPlatformFallback(assert.AnError))
}
