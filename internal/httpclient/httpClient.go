package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/perf"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
)

type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

type RetryConfig struct {
	MaxRetries int
	Interval   time.Duration
}

type RLHTTPClient struct {
	client      *http.Client
	Ratelimiter *rate.Limiter
	RetryConfig *RetryConfig
}

func (client *RLHTTPClient) Do(request *http.Request) (*http.Response, error) {
	ctx, requestSpan := perf.StartSpan(request.Context(), "net.http.request",
		perf.WithAttributes(
			attribute.String("url", request.URL.String()),
			attribute.String("method", request.Method),
			attribute.String("host", request.URL.Host),
		),
	)
	defer requestSpan.End()
	retryConfig := client.retryConfig()

	var response *http.Response
	var err error

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		shouldRetry := false
		response, shouldRetry, err = client.doAttempt(ctx, request, attempt, retryConfig, requestSpan)
		if err != nil {
			return nil, err
		}
		if shouldRetry {
			continue
		}
		break
	}

	requestSpan.SetAttributes(attribute.Bool("success", err == nil))
	if response != nil {
		requestSpan.SetAttributes(attribute.Int("status", response.StatusCode))
	}
	return response, err
}

func (client *RLHTTPClient) retryConfig() RetryConfig {
	if client.RetryConfig != nil {
		return *client.RetryConfig
	}
	return RetryConfig{
		MaxRetries: 3,
		Interval:   1 * time.Second,
	}
}

func (client *RLHTTPClient) doAttempt(
	ctx context.Context,
	request *http.Request,
	attempt int,
	retryConfig RetryConfig,
	requestSpan *perf.Span,
) (*http.Response, bool, error) {
	attemptCtx, attemptSpan := perf.StartSpan(ctx, "net.http.request.attempt",
		perf.WithAttributes(
			attribute.Int("attempt", attempt),
			attribute.String("url", request.URL.String()),
		),
	)
	defer attemptSpan.End()

	waitErr := client.waitForRateLimit(attemptCtx, attempt, request)
	if waitErr != nil {
		attemptSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error_type", fmt.Sprintf("%T", waitErr)),
		)
		requestSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error_type", fmt.Sprintf("%T", waitErr)),
		)
		if IsTimeoutError(waitErr) {
			return nil, false, WrapTimeoutError(waitErr)
		}
		return nil, false, fmt.Errorf("rate limit burst exceeded %w", waitErr)
	}

	response, err := client.client.Do(request.WithContext(attemptCtx))
	if err != nil {
		attemptSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error_type", fmt.Sprintf("%T", err)),
		)
		requestSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error_type", fmt.Sprintf("%T", err)),
		)
		return nil, false, WrapTimeoutError(err)
	}

	if shouldRetry(response, attempt, retryConfig) {
		attemptSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.Int("status", response.StatusCode),
		)
		if drainErr := drainAndClose(response.Body); drainErr != nil {
			attemptSpan.SetAttributes(attribute.String("cleanup_error", drainErr.Error()))
		}
		time.Sleep(retryConfig.Interval)
		return nil, true, nil
	}

	attemptSpan.SetAttributes(
		attribute.Bool("success", true),
		attribute.Int("status", response.StatusCode),
	)
	return response, false, nil
}

func (client *RLHTTPClient) waitForRateLimit(ctx context.Context, attempt int, request *http.Request) error {
	_, waitSpan := perf.StartSpan(ctx, "net.http.ratelimit.wait",
		perf.WithAttributes(
			attribute.Int("attempt", attempt),
			attribute.String("url", request.URL.String()),
		),
	)
	waitErr := client.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	waitSpan.End()
	return waitErr
}

func shouldRetry(response *http.Response, attempt int, retryConfig RetryConfig) bool {
	return response.StatusCode >= 500 && response.StatusCode < 600 && attempt < retryConfig.MaxRetries
}

func NewRLClient(limiter *rate.Limiter) *RLHTTPClient {
	client := &RLHTTPClient{
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		Ratelimiter: limiter,
	}
	return client
}

func NoRetries() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 0,
		Interval:   0,
	}
}

func drainAndClose(body io.ReadCloser) error {
	if body == nil {
		return nil
	}

	readErr := drainBody(body)
	closeErr := body.Close()
	if readErr != nil && closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func drainBody(body io.Reader) error {
	_, err := io.Copy(io.Discard, body)
	return err
}
