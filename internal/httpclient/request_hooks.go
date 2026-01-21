package httpclient

import (
	"context"
	"net/http"
)

// RequestStartHook is invoked after rate limiting succeeds and just before the HTTP request is sent.
type RequestStartHook func(*http.Request)

type requestStartHookKey struct{}

// WithRequestStartHook stores a hook in the context for RLHTTPClient to invoke before sending requests.
// It returns the original context when the hook is nil.
func WithRequestStartHook(ctx context.Context, hook RequestStartHook) context.Context {
	if hook == nil {
		return ctx
	}
	return context.WithValue(ctx, requestStartHookKey{}, hook)
}

// RequestStartHookFromContext retrieves the request-start hook from the context, if one is set.
func RequestStartHookFromContext(ctx context.Context) RequestStartHook {
	hook, ok := ctx.Value(requestStartHookKey{}).(RequestStartHook)
	if !ok {
		return nil
	}
	return hook
}
