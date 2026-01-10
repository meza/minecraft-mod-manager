package install

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithInstallViewObserverReturnsOriginalContextWhenNil(t *testing.T) {
	ctx := context.Background()
	result := WithInstallViewObserver(ctx, nil)

	assert.True(t, ctx == result)
}

func TestNotifyInstallViewObserverNoObserver(t *testing.T) {
	assert.NotPanics(t, func() {
		NotifyInstallViewObserver(context.Background(), "ignored")
	})
}

func TestNotifyInstallViewObserverCallsObserver(t *testing.T) {
	called := false
	observedView := ""
	ctx := WithInstallViewObserver(context.Background(), func(view string) {
		called = true
		observedView = view
	})

	NotifyInstallViewObserver(ctx, "final view")

	assert.True(t, called)
	assert.Equal(t, "final view", observedView)
}
