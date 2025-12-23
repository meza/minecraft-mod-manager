package httpclient

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

const (
	DefaultMetadataTimeout = 15 * time.Second
	DefaultDownloadTimeout = 5 * time.Minute
)

type TimeoutError struct {
	Err error
}

func (e *TimeoutError) Error() string {
	return i18n.T("error.network_timeout")
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func WrapTimeoutError(err error) error {
	if !IsTimeoutError(err) {
		return err
	}
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr
	}
	return &TimeoutError{Err: err}
}

func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

func WithMetadataTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, DefaultMetadataTimeout)
}

func WithDownloadTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, DefaultDownloadTimeout)
}
