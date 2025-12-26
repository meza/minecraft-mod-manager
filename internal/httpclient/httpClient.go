package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
	client         *http.Client
	Ratelimiter    *rate.Limiter
	RetryConfig    *RetryConfig
	rateLimitMutex sync.Mutex
	rateLimitUntil time.Time
}

func (client *RLHTTPClient) Do(request *http.Request) (*http.Response, error) {
	retryConfig := client.retryConfig()
	if validationErr := validateRequestForRetries(request, retryConfig); validationErr != nil {
		return nil, validationErr
	}

	ctx, requestSpan := perf.StartSpan(request.Context(), "net.http.request",
		perf.WithAttributes(
			attribute.String("url", request.URL.String()),
			attribute.String("method", request.Method),
			attribute.String("host", request.URL.Host),
		),
	)
	defer requestSpan.End()

	response, err := client.doWithRetries(ctx, request, retryConfig, requestSpan)

	requestSpan.SetAttributes(attribute.Bool("success", err == nil))
	if response != nil {
		requestSpan.SetAttributes(attribute.Int("status", response.StatusCode))
	}
	return response, err
}

func (client *RLHTTPClient) doWithRetries(ctx context.Context, request *http.Request, retryConfig RetryConfig, requestSpan *perf.Span) (*http.Response, error) {
	var response *http.Response
	var shouldRetry bool
	var err error

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		response, shouldRetry, err = client.doAttempt(ctx, request, attempt, retryConfig, requestSpan)
		if err != nil {
			return nil, err
		}
		if shouldRetry {
			continue
		}
		break
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

	if waitErr := client.waitForRateLimit(attemptCtx, attempt, request); waitErr != nil {
		return nil, false, handleRateLimitWaitError(attemptSpan, requestSpan, waitErr)
	}

	attemptRequest, err := buildAttemptRequest(attemptCtx, request, attempt)
	if err != nil {
		return nil, false, handleAttemptBuildError(attemptSpan, requestSpan, err)
	}

	response, err := client.client.Do(attemptRequest)
	if err != nil {
		shouldRetry, attemptErr := client.handleAttemptError(attemptCtx, attemptSpan, requestSpan, err, attempt, retryConfig)
		if attemptErr != nil {
			return nil, false, attemptErr
		}
		return nil, shouldRetry, nil
	}

	client.noteRateLimitDelay(response, attemptSpan)
	return client.handleAttemptResponse(attemptCtx, attemptSpan, requestSpan, response, attempt, retryConfig)
}

func handleRateLimitWaitError(attemptSpan *perf.Span, requestSpan *perf.Span, waitErr error) error {
	recordAttemptFailure(attemptSpan, requestSpan, waitErr)
	if IsTimeoutError(waitErr) {
		return WrapTimeoutError(waitErr)
	}
	return fmt.Errorf("rate limit burst exceeded %w", waitErr)
}

func handleAttemptBuildError(attemptSpan *perf.Span, requestSpan *perf.Span, err error) error {
	recordAttemptFailure(attemptSpan, requestSpan, err)
	return err
}

func (client *RLHTTPClient) handleAttemptError(
	ctx context.Context,
	attemptSpan *perf.Span,
	requestSpan *perf.Span,
	err error,
	attempt int,
	retryConfig RetryConfig,
) (bool, error) {
	recordAttemptFailure(attemptSpan, requestSpan, err)
	if shouldRetryError(err, attempt, retryConfig) {
		attemptSpan.SetAttributes(attribute.Bool("retry", true))
		waitErr := waitForRetry(ctx, retryDelay(attempt, retryConfig))
		if waitErr != nil {
			recordAttemptFailure(attemptSpan, requestSpan, waitErr)
			return false, WrapTimeoutError(waitErr)
		}
		return true, nil
	}
	return false, WrapTimeoutError(err)
}

func (client *RLHTTPClient) handleAttemptResponse(
	ctx context.Context,
	attemptSpan *perf.Span,
	requestSpan *perf.Span,
	response *http.Response,
	attempt int,
	retryConfig RetryConfig,
) (*http.Response, bool, error) {
	retryDelayValue, retryResponse := retryDelayForResponse(response, attempt, retryConfig)
	if retryResponse {
		recordAttemptRetry(attemptSpan, response)
		waitErr := waitForRetry(ctx, retryDelayValue)
		if waitErr != nil {
			recordAttemptFailure(attemptSpan, requestSpan, waitErr)
			return nil, false, WrapTimeoutError(waitErr)
		}
		return nil, true, nil
	}

	attemptSpan.SetAttributes(
		attribute.Bool("success", true),
		attribute.Int("status", response.StatusCode),
	)
	return response, false, nil
}

func recordAttemptFailure(attemptSpan *perf.Span, requestSpan *perf.Span, err error) {
	attemptSpan.SetAttributes(
		attribute.Bool("success", false),
		attribute.String("error_type", fmt.Sprintf("%T", err)),
	)
	requestSpan.SetAttributes(
		attribute.Bool("success", false),
		attribute.String("error_type", fmt.Sprintf("%T", err)),
	)
}

func recordAttemptRetry(attemptSpan *perf.Span, response *http.Response) {
	attemptSpan.SetAttributes(
		attribute.Bool("success", false),
		attribute.Int("status", response.StatusCode),
	)
	if drainErr := drainAndClose(response.Body); drainErr != nil {
		attemptSpan.SetAttributes(attribute.String("cleanup_error", drainErr.Error()))
	}
}

func (client *RLHTTPClient) waitForRateLimit(ctx context.Context, attempt int, request *http.Request) error {
	_, waitSpan := perf.StartSpan(ctx, "net.http.ratelimit.wait",
		perf.WithAttributes(
			attribute.Int("attempt", attempt),
			attribute.String("url", request.URL.String()),
		),
	)
	waitErr := client.waitForRateLimitDelay(ctx)
	if waitErr != nil {
		waitSpan.End()
		return waitErr
	}
	waitErr = client.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	waitSpan.End()
	return waitErr
}

func NewRLClient(limiter *rate.Limiter) *RLHTTPClient {
	if limiter == nil {
		limiter = DefaultLimiter()
	}
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

func validateRequestForRetries(request *http.Request, retryConfig RetryConfig) error {
	if request == nil {
		return errors.New("request is nil")
	}
	if request.URL == nil {
		return errors.New("request url is nil")
	}
	if retryConfig.MaxRetries <= 0 {
		return nil
	}
	if request.Body == nil || request.Body == http.NoBody {
		return nil
	}
	if request.GetBody == nil {
		return errors.New("request body retries require GetBody")
	}
	return nil
}

func buildAttemptRequest(attemptCtx context.Context, request *http.Request, attemptIndex int) (*http.Request, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	attemptRequest := request.Clone(attemptCtx)
	if request.Body == nil || request.Body == http.NoBody {
		return attemptRequest, nil
	}
	if attemptIndex == 0 {
		attemptRequest.Body = request.Body
		return attemptRequest, nil
	}
	if request.GetBody == nil {
		return nil, errors.New("request body retries require GetBody")
	}
	body, err := request.GetBody()
	if err != nil {
		return nil, fmt.Errorf("failed to reset request body: %w", err)
	}
	attemptRequest.Body = body
	return attemptRequest, nil
}

func retryDelay(attempt int, retryConfig RetryConfig) time.Duration {
	if retryConfig.Interval <= 0 {
		return 0
	}
	return retryConfig.Interval * time.Duration(attempt+1)
}

func shouldRetryError(err error, attempt int, retryConfig RetryConfig) bool {
	if err == nil || attempt >= retryConfig.MaxRetries {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if IsTimeoutError(err) {
		return true
	}
	var tempErr temporaryError
	return errors.As(err, &tempErr) && tempErr.Temporary()
}

type temporaryError interface {
	Temporary() bool
}

func retryDelayForResponse(response *http.Response, attempt int, retryConfig RetryConfig) (time.Duration, bool) {
	if response == nil || attempt >= retryConfig.MaxRetries {
		return 0, false
	}
	if response.StatusCode == http.StatusTooManyRequests {
		delayFromHeaders, hasDelay := rateLimitDelayFromHeaders(response.Header, time.Now())
		delay := retryDelay(attempt, retryConfig)
		if hasDelay && delayFromHeaders > delay {
			delay = delayFromHeaders
		}
		return delay, true
	}
	if response.StatusCode >= http.StatusInternalServerError && response.StatusCode < http.StatusNetworkAuthenticationRequired {
		return retryDelay(attempt, retryConfig), true
	}
	return 0, false
}

func (client *RLHTTPClient) noteRateLimitDelay(response *http.Response, attemptSpan *perf.Span) {
	if response == nil {
		return
	}
	delay, hasDelay := rateLimitDelayFromHeaders(response.Header, time.Now())
	if !hasDelay {
		return
	}
	client.setRateLimitUntil(delay)
	if attemptSpan != nil {
		attemptSpan.SetAttributes(
			attribute.Bool("rate_limited", true),
			attribute.String("rate_limit_delay", delay.String()),
		)
	}
}

func rateLimitDelayFromHeaders(header http.Header, now time.Time) (time.Duration, bool) {
	if header == nil {
		return 0, false
	}
	retryAfterValue := header.Get("Retry-After")
	if retryDelayValue, ok := parseRetryAfterDelay(retryAfterValue, now); ok {
		return retryDelayValue, true
	}
	remainingValue := strings.TrimSpace(header.Get("X-Ratelimit-Remaining"))
	resetValue := strings.TrimSpace(header.Get("X-Ratelimit-Reset"))
	if remainingValue == "" || resetValue == "" {
		return 0, false
	}
	remainingCount, err := strconv.Atoi(remainingValue)
	if err != nil || remainingCount >= DefaultRateLimitRemainingThreshold {
		return 0, false
	}
	return parseResetDelay(resetValue, now)
}

func parseRetryAfterDelay(retryAfterValue string, now time.Time) (time.Duration, bool) {
	retryAfterValue = strings.TrimSpace(retryAfterValue)
	if retryAfterValue == "" {
		return 0, false
	}
	if secondsValue, err := strconv.Atoi(retryAfterValue); err == nil {
		if secondsValue <= 0 {
			return 0, false
		}
		return time.Duration(secondsValue) * time.Second, true
	}
	parsedTime, err := http.ParseTime(retryAfterValue)
	if err != nil {
		return 0, false
	}
	delay := parsedTime.Sub(now)
	if delay <= 0 {
		return 0, false
	}
	return delay, true
}

func parseResetDelay(resetValue string, now time.Time) (time.Duration, bool) {
	resetValue = strings.TrimSpace(resetValue)
	if resetValue == "" {
		return 0, false
	}
	resetSeconds, err := strconv.ParseInt(resetValue, 10, 64)
	if err != nil || resetSeconds <= 0 {
		return 0, false
	}
	if resetSeconds <= DefaultRateLimitResetMaxRelativeSeconds {
		return time.Duration(resetSeconds) * time.Second, true
	}

	resetTime := time.Unix(resetSeconds, 0)
	if !resetTime.After(now) {
		return 0, false
	}
	return resetTime.Sub(now), true
}

func (client *RLHTTPClient) setRateLimitUntil(delay time.Duration) {
	if delay <= 0 {
		return
	}
	targetTime := time.Now().Add(delay)
	client.rateLimitMutex.Lock()
	if targetTime.After(client.rateLimitUntil) {
		client.rateLimitUntil = targetTime
	}
	client.rateLimitMutex.Unlock()
}

func (client *RLHTTPClient) waitForRateLimitDelay(ctx context.Context) error {
	client.rateLimitMutex.Lock()
	waitUntil := client.rateLimitUntil
	client.rateLimitMutex.Unlock()
	if waitUntil.IsZero() {
		return nil
	}
	delay := time.Until(waitUntil)
	if delay <= 0 {
		return nil
	}
	return waitForRetry(ctx, delay)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
