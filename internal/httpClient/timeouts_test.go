package httpClient

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/stretchr/testify/assert"
)

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

type nonTimeoutNetError struct{}

func (nonTimeoutNetError) Error() string   { return "no timeout" }
func (nonTimeoutNetError) Timeout() bool   { return false }
func (nonTimeoutNetError) Temporary() bool { return false }

func TestTimeoutErrorMessage(t *testing.T) {
	err := &TimeoutError{Err: context.DeadlineExceeded}
	assert.Equal(t, i18n.T("error.network_timeout"), err.Error())
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestIsTimeoutError(t *testing.T) {
	assert.False(t, IsTimeoutError(nil))
	assert.True(t, IsTimeoutError(context.DeadlineExceeded))
	assert.True(t, IsTimeoutError(timeoutNetError{}))
	assert.False(t, IsTimeoutError(nonTimeoutNetError{}))
	assert.False(t, IsTimeoutError(errors.New("not a timeout")))
}

func TestWrapTimeoutError(t *testing.T) {
	wrapped := WrapTimeoutError(context.DeadlineExceeded)
	var timeoutErr *TimeoutError
	assert.ErrorAs(t, wrapped, &timeoutErr)

	notWrapped := WrapTimeoutError(errors.New("nope"))
	assert.False(t, errors.As(notWrapped, &timeoutErr))

	alreadyWrapped := &TimeoutError{Err: context.DeadlineExceeded}
	assert.Same(t, alreadyWrapped, WrapTimeoutError(alreadyWrapped))

	wrappedAgain := fmt.Errorf("download failed: %w", alreadyWrapped)
	assert.Same(t, alreadyWrapped, WrapTimeoutError(wrappedAgain))
}

func TestTimeoutHelpers(t *testing.T) {
	metaCtx, metaCancel := WithMetadataTimeout(context.Background())
	t.Cleanup(metaCancel)

	metaDeadline, ok := metaCtx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(DefaultMetadataTimeout), metaDeadline, time.Second)

	downloadCtx, downloadCancel := WithDownloadTimeout(context.Background())
	t.Cleanup(downloadCancel)

	downloadDeadline, ok := downloadCtx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(DefaultDownloadTimeout), downloadDeadline, time.Second)
}
