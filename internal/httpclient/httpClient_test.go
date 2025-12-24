package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return roundTripper(req)
}

type sequenceTransport struct {
	responses []*http.Response
	callCount int
}

func (transport *sequenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if transport.callCount >= len(transport.responses) {
		return nil, fmt.Errorf("no response configured for call %d", transport.callCount)
	}
	resp := transport.responses[transport.callCount]
	transport.callCount++
	return resp, nil
}

type trackingBody struct {
	reader *strings.Reader
	read   bool
	closed bool
}

func newTrackingBody(payload string) *trackingBody {
	return &trackingBody{
		reader: strings.NewReader(payload),
	}
}

func (body *trackingBody) Read(p []byte) (int, error) {
	n, err := body.reader.Read(p)
	if n > 0 {
		body.read = true
	}
	return n, err
}

func (body *trackingBody) Close() error {
	body.closed = true
	return nil
}

func closeResponseBody(testingContext *testing.T, response *http.Response) {
	testingContext.Helper()
	if response == nil || response.Body == nil {
		return
	}
	if err := response.Body.Close(); err != nil {
		testingContext.Fatalf("failed to close response body: %v", err)
	}
}

func newOKServer(testingContext *testing.T) *httptest.Server {
	testingContext.Helper()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			testingContext.Fatalf("failed to write response: %v", err)
		}
	}))
	testingContext.Cleanup(mockServer.Close)
	return mockServer
}

func newRequest(testingContext *testing.T, url string) *http.Request {
	testingContext.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		testingContext.Fatalf("failed to create request: %v", err)
	}
	return request
}

func assertRequestDurationAtLeast(
	testingContext *testing.T,
	client *RLHTTPClient,
	request *http.Request,
	requestCount int,
	minDuration time.Duration,
) {
	testingContext.Helper()
	start := time.Now()
	for index := 0; index < requestCount; index++ {
		response, err := client.Do(request)
		if err != nil {
			testingContext.Fatalf("Do failed: %v", err)
		}
		if response.StatusCode != http.StatusOK {
			testingContext.Fatalf("Expected status OK, got %v", response.StatusCode)
		}
		closeResponseBody(testingContext, response)
	}
	duration := time.Since(start)
	if duration < minDuration {
		testingContext.Fatalf("Rate limiter did not enforce delay, duration: %v", duration)
	}
}

func assertRequestDurationAtMost(
	testingContext *testing.T,
	client *RLHTTPClient,
	request *http.Request,
	requestCount int,
	maxDuration time.Duration,
) {
	testingContext.Helper()
	start := time.Now()
	for index := 0; index < requestCount; index++ {
		response, err := client.Do(request)
		if err != nil {
			testingContext.Fatalf("Do failed: %v", err)
		}
		if response.StatusCode != http.StatusOK {
			testingContext.Fatalf("Expected status OK, got %v", response.StatusCode)
		}
		closeResponseBody(testingContext, response)
	}
	duration := time.Since(start)
	if duration > maxDuration {
		testingContext.Fatalf("Requests took too long, duration: %v", duration)
	}
}

func TestRLHTTPClient_DoWithRateLimiting(t *testing.T) {
	mockServer := newOKServer(t)

	limiter := rate.NewLimiter(1, 1)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()

	request := newRequest(t, mockServer.URL)
	expectedDuration := 2 * time.Second // 3 requests with 1 request per second rate limit

	assertRequestDurationAtLeast(t, client, request, 3, expectedDuration)
}

func TestRLHTTPClient_DoWithRateLimiting_Assert(t *testing.T) {
	mockServer := newOKServer(t)

	limiter := rate.NewLimiter(1, 1)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()

	request := newRequest(t, mockServer.URL)
	expectedDuration := 2 * time.Second // 3 requests with 1 request per second rate limit

	start := time.Now()
	for index := 0; index < 3; index++ {
		response, err := client.Do(request)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		closeResponseBody(t, response)
	}
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, expectedDuration)
}

func TestRLHTTPClient_DoWithoutRateLimiting(t *testing.T) {
	mockServer := newOKServer(t)

	client := NewRLClient(rate.NewLimiter(rate.Inf, 0))
	client.RetryConfig = NoRetries()
	request := newRequest(t, mockServer.URL)

	assertRequestDurationAtMost(t, client, request, 5, time.Second)
}

func TestRLHTTPClient_DoWithoutRateLimiting_Assert(t *testing.T) {
	mockServer := newOKServer(t)

	client := NewRLClient(rate.NewLimiter(rate.Inf, 0))
	client.RetryConfig = NoRetries()
	request := newRequest(t, mockServer.URL)

	start := time.Now()
	for index := 0; index < 5; index++ {
		response, err := client.Do(request)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		closeResponseBody(t, response)
	}
	duration := time.Since(start)
	assert.Less(t, duration, time.Second)
}

func TestRLHTTPClient_DoWithRetriesOnServerError(t *testing.T) {
	attempts := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(mockServer.Close)

	limiter := rate.NewLimiter(1, 1)
	client := NewRLClient(limiter)
	client.RetryConfig = &RetryConfig{
		MaxRetries: 3,
		Interval:   1 * time.Second,
	}

	request := newRequest(t, mockServer.URL)

	start := time.Now()
	response, err := client.Do(request)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	closeResponseBody(t, response)
	duration := time.Since(start)
	expectedDuration := 2 * time.Second // 2 retries with 1 second interval
	assert.GreaterOrEqual(t, duration, expectedDuration)
}

func TestRLHTTPClient_NoRetriesOnClientError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(mockServer.Close)

	limiter := rate.NewLimiter(rate.Inf, 0)
	client := NewRLClient(limiter)
	client.RetryConfig = &RetryConfig{
		MaxRetries: 3,
		Interval:   1 * time.Second,
	}

	request := newRequest(t, mockServer.URL)

	start := time.Now()
	response, err := client.Do(request)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	closeResponseBody(t, response)
	duration := time.Since(start)
	expectedDuration := 1 * time.Second // No retries for client error
	assert.LessOrEqual(t, duration, expectedDuration)
}

func TestRLHTTPClient_DoWithRateLimitError(t *testing.T) {
	mockServer := newOKServer(t)

	limiter := rate.NewLimiter(1, 0)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()

	request := newRequest(t, mockServer.URL)

	response, err := client.Do(request)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "rate limit burst exceeded")
}

func TestRLHTTPClient_DoWithHTTPClientError(t *testing.T) {
	client := NewRLClient(rate.NewLimiter(1, 1))
	client.RetryConfig = &RetryConfig{
		MaxRetries: 3,
		Interval:   1 * time.Second,
	}
	client.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("round trip error")
		}),
	}

	request := newRequest(t, "https://example.com")
	response, err := client.Do(request)
	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestRLHTTPClient_ClosesResponseBodiesBeforeRetry(t *testing.T) {
	firstFailureBody := newTrackingBody("first failure body")
	secondFailureBody := newTrackingBody("second failure body")
	successBody := newTrackingBody("success body")

	transport := &sequenceTransport{
		responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       firstFailureBody,
				Header:     make(http.Header),
			},
			{
				StatusCode: http.StatusBadGateway,
				Body:       secondFailureBody,
				Header:     make(http.Header),
			},
			{
				StatusCode: http.StatusOK,
				Body:       successBody,
				Header:     make(http.Header),
			},
		},
	}

	client := NewRLClient(rate.NewLimiter(rate.Inf, 0))
	client.RetryConfig = &RetryConfig{MaxRetries: 2, Interval: 0}
	client.client = &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com/retry", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Cleanup(func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	})

	assert.Equal(t, 3, transport.callCount)
	assert.True(t, firstFailureBody.closed, "first failure body must be closed before retry")
	assert.True(t, secondFailureBody.closed, "second failure body must be closed before retry")
	assert.True(t, firstFailureBody.read, "first failure body should be drained to allow connection reuse")
	assert.True(t, secondFailureBody.read, "second failure body should be drained to allow connection reuse")
	assert.False(t, successBody.closed, "successful response body should remain open for the caller")
}

func TestDrainAndCloseHandlesNilBody(t *testing.T) {
	assert.NotPanics(t, func() {
		assert.NoError(t, drainAndClose(nil))
	})
}

type errorBody struct {
	readErr  error
	closeErr error
	closed   bool
}

func (body *errorBody) Read(_ []byte) (int, error) {
	if body.readErr != nil {
		return 0, body.readErr
	}
	return 0, io.EOF
}

func (body *errorBody) Close() error {
	body.closed = true
	if body.closeErr != nil {
		return body.closeErr
	}
	return nil
}

func TestDrainAndCloseReturnsReadError(t *testing.T) {
	body := &errorBody{readErr: errors.New("read failed")}
	assert.ErrorContains(t, drainAndClose(body), "read failed")
	assert.True(t, body.closed)
}

func TestDrainAndCloseReturnsCloseError(t *testing.T) {
	body := &errorBody{closeErr: errors.New("close failed")}
	assert.ErrorContains(t, drainAndClose(body), "close failed")
	assert.True(t, body.closed)
}

func TestDrainAndCloseJoinsReadAndCloseErrors(t *testing.T) {
	readErr := errors.New("read failed")
	closeErr := errors.New("close failed")
	body := &errorBody{readErr: readErr, closeErr: closeErr}

	err := drainAndClose(body)
	assert.ErrorContains(t, err, "read failed")
	assert.ErrorContains(t, err, "close failed")
	assert.ErrorIs(t, err, readErr)
	assert.ErrorIs(t, err, closeErr)
	assert.True(t, body.closed)
}

func TestRLHTTPClientRetryConfigDefaultsAndOverrides(t *testing.T) {
	client := &RLHTTPClient{}

	defaultConfig := client.retryConfig()
	assert.Equal(t, 3, defaultConfig.MaxRetries)
	assert.Equal(t, time.Second, defaultConfig.Interval)

	overridden := RetryConfig{MaxRetries: 5, Interval: 2 * time.Second}
	client.RetryConfig = &overridden
	overrideConfig := client.retryConfig()
	assert.Equal(t, overridden, overrideConfig)
}

type retryReadErrorBody struct {
	readErr error
	closed  bool
}

func (body *retryReadErrorBody) Read(_ []byte) (int, error) {
	return 0, body.readErr
}

func (body *retryReadErrorBody) Close() error {
	body.closed = true
	return nil
}

func TestRLHTTPClient_RetriesWhenDrainFails(t *testing.T) {
	firstBody := &retryReadErrorBody{readErr: errors.New("read failed")}
	successBody := newTrackingBody("ok")

	transport := &sequenceTransport{
		responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       firstBody,
				Header:     make(http.Header),
			},
			{
				StatusCode: http.StatusOK,
				Body:       successBody,
				Header:     make(http.Header),
			},
		},
	}

	client := NewRLClient(rate.NewLimiter(rate.Inf, 0))
	client.RetryConfig = &RetryConfig{MaxRetries: 1, Interval: 0}
	client.client = &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, "https://example.com/retry", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	closeResponseBody(t, resp)
}

func TestRLHTTPClient_ReturnsTimeoutErrorFromRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Inf, 0)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.Nil(t, resp)
	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
}

func TestRLHTTPClient_WrapsTimeoutErrorFromTransport(t *testing.T) {
	limiter := rate.NewLimiter(rate.Inf, 0)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()
	client.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.Nil(t, resp)
	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
}
