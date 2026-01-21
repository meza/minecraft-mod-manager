package httpclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithRequestStartHookReturnsOriginalContextWhenNil(t *testing.T) {
	ctx := context.Background()

	hooked := WithRequestStartHook(ctx, nil)

	assert.Equal(t, ctx, hooked)
	assert.Nil(t, RequestStartHookFromContext(hooked))
}

func TestRequestStartHookFromContextReturnsHook(t *testing.T) {
	ctx := context.Background()
	hook := RequestStartHook(func(*http.Request) {})

	hooked := WithRequestStartHook(ctx, hook)

	assert.NotNil(t, RequestStartHookFromContext(hooked))
}
