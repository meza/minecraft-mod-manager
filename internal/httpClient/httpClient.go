package httpClient

import (
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
	retryConfig := RetryConfig{
		MaxRetries: 3,
		Interval:   1 * time.Second,
	}

	if client.RetryConfig != nil {
		retryConfig = *client.RetryConfig
	}

	var response *http.Response
	var err error

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		attemptCtx, attemptSpan := perf.StartSpan(ctx, "net.http.request.attempt",
			perf.WithAttributes(
				attribute.Int("attempt", attempt),
				attribute.String("url", request.URL.String()),
			),
		)

		_, waitSpan := perf.StartSpan(attemptCtx, "net.http.ratelimit.wait",
			perf.WithAttributes(
				attribute.Int("attempt", attempt),
				attribute.String("url", request.URL.String()),
			),
		)
		err = client.Ratelimiter.Wait(attemptCtx) // This is a blocking call. Honors the rate limit
		waitSpan.End()
		if err != nil {
			attemptSpan.SetAttributes(
				attribute.Bool("success", false),
				attribute.String("error_type", fmt.Sprintf("%T", err)),
			)
			requestSpan.SetAttributes(
				attribute.Bool("success", false),
				attribute.String("error_type", fmt.Sprintf("%T", err)),
			)
			attemptSpan.End()
			if IsTimeoutError(err) {
				return nil, WrapTimeoutError(err)
			}
			return nil, fmt.Errorf("rate limit burst exceeded %w", err)
		}

		response, err = client.client.Do(request.WithContext(attemptCtx))
		if err != nil {
			attemptSpan.SetAttributes(
				attribute.Bool("success", false),
				attribute.String("error_type", fmt.Sprintf("%T", err)),
			)
			requestSpan.SetAttributes(
				attribute.Bool("success", false),
				attribute.String("error_type", fmt.Sprintf("%T", err)),
			)
			attemptSpan.End()
			return nil, WrapTimeoutError(err)
		}

		// Check if the response status is a server error (5xx)
		if response.StatusCode >= 500 && response.StatusCode < 600 {
			if attempt < retryConfig.MaxRetries {
				attemptSpan.SetAttributes(
					attribute.Bool("success", false),
					attribute.Int("status", response.StatusCode),
				)
				if drainErr := drainAndClose(response.Body); drainErr != nil {
					attemptSpan.SetAttributes(attribute.String("cleanup_error", drainErr.Error()))
				}
				time.Sleep(retryConfig.Interval)
				attemptSpan.End()
				continue
			}
		}

		attemptSpan.SetAttributes(
			attribute.Bool("success", true),
			attribute.Int("status", response.StatusCode),
		)
		// If the response is successful or a non-retryable error occurs, return the response or error
		attemptSpan.End()
		break
	}

	requestSpan.SetAttributes(attribute.Bool("success", err == nil))
	if response != nil {
		requestSpan.SetAttributes(attribute.Int("status", response.StatusCode))
	}
	return response, err
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
