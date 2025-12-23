package httpClient

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

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type sequenceTransport struct {
	responses []*http.Response
	callCount int
}

func (t *sequenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.callCount >= len(t.responses) {
		return nil, fmt.Errorf("no response configured for call %d", t.callCount)
	}
	resp := t.responses[t.callCount]
	t.callCount++
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

func (b *trackingBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if n > 0 {
		b.read = true
	}
	return n, err
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

func TestRLHTTPClient_Fetch(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Create a rate limiter that allows 1 request per second
	limiter := rate.NewLimiter(1, 1)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()

	// Create a new HTTP request
	req, err := http.NewRequest("GET", mockServer.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Test the Do method
	t.Run("Do with rate limiting", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 3; i++ {
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Do failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected status OK, got %v", resp.StatusCode)
			}
		}
		duration := time.Since(start)
		expectedDuration := 2 * time.Second // 3 requests with 1 request per second rate limit
		if duration < expectedDuration {
			t.Fatalf("Rate limiter did not enforce delay, duration: %v", duration)
		}
	})

	// Test the Do method without rate limiting
	t.Run("Do without rate limiting", func(t *testing.T) {
		client.Ratelimiter = rate.NewLimiter(rate.Inf, 0) // Disable rate limiting
		client.RetryConfig = NoRetries()
		start := time.Now()
		for i := 0; i < 5; i++ {
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Do failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected status OK, got %v", resp.StatusCode)
			}
		}
		duration := time.Since(start)
		if duration > time.Second {
			t.Fatalf("Requests took too long, duration: %v", duration)
		}
	})

	t.Run("Do with rate limiting", func(t *testing.T) {
		// Create a mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("OK")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		// Create a rate limiter that allows 1 request per second
		limiter := rate.NewLimiter(1, 1)
		client := NewRLClient(limiter)
		client.RetryConfig = NoRetries()

		// Create a new HTTP request
		req, err := http.NewRequest("GET", mockServer.URL, nil)
		assert.NoError(t, err)

		// Test the Do method
		start := time.Now()
		for i := 0; i < 3; i++ {
			resp, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
		duration := time.Since(start)
		expectedDuration := 2 * time.Second // 3 requests with 1 request per second rate limit
		assert.GreaterOrEqual(t, duration, expectedDuration)
	})

	t.Run("Do without rate limiting", func(t *testing.T) {
		// Create a mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("OK")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		// Disable rate limiting
		client := NewRLClient(rate.NewLimiter(rate.Inf, 0))
		client.RetryConfig = NoRetries()

		// Create a new HTTP request
		req, err := http.NewRequest("GET", mockServer.URL, nil)
		assert.NoError(t, err)

		// Test the Do method
		start := time.Now()
		for i := 0; i < 5; i++ {
			resp, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
		duration := time.Since(start)
		assert.Less(t, duration, time.Second)
	})

	t.Run("Do with retries on server error", func(t *testing.T) {
		attempts := 0
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts <= 2 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("OK")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}
		}))
		defer mockServer.Close()

		// Create a rate limiter that allows 1 request per second
		limiter := rate.NewLimiter(1, 1)
		client := NewRLClient(limiter)
		client.RetryConfig = &RetryConfig{
			MaxRetries: 3,
			Interval:   1 * time.Second,
		}

		// Create a new HTTP request
		req, err := http.NewRequest("GET", mockServer.URL, nil)
		assert.NoError(t, err)

		// Test the Do method
		start := time.Now()
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		duration := time.Since(start)
		expectedDuration := 2 * time.Second // 2 retries with 1 second interval
		assert.GreaterOrEqual(t, duration, expectedDuration)
	})

	t.Run("Do with no retries on client error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer mockServer.Close()

		// Create a rate limiter that allows 1 request per second
		limiter := rate.NewLimiter(rate.Inf, 0)
		client := NewRLClient(limiter)
		client.RetryConfig = &RetryConfig{
			MaxRetries: 3,
			Interval:   1 * time.Second,
		}

		// Create a new HTTP request
		req, err := http.NewRequest("GET", mockServer.URL, nil)
		assert.NoError(t, err)

		// Test the Do method
		start := time.Now()
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		duration := time.Since(start)
		expectedDuration := 1 * time.Second // No retries for client error
		assert.LessOrEqual(t, duration, expectedDuration)
	})

	t.Run("Do with rate limit error", func(t *testing.T) {
		// Create a mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("OK")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		// Create a rate limiter with a burst of 0
		limiter := rate.NewLimiter(1, 0)
		client := NewRLClient(limiter)
		client.RetryConfig = NoRetries()

		// Create a new HTTP request
		req, err := http.NewRequest("GET", mockServer.URL, nil)
		assert.NoError(t, err)

		// Test the Do method
		resp, err := client.Do(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "rate limit burst exceeded")
	})

	t.Run("Do with HTTP client error", func(t *testing.T) {

		// Create a rate limiter that allows 1 request per second
		limiter := rate.NewLimiter(1, 1)
		client := NewRLClient(limiter)
		client.RetryConfig = &RetryConfig{
			MaxRetries: 3,
			Interval:   1 * time.Second,
		}

		// Use a custom HTTP client that always returns an error
		client.client = &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("round trip error")
			}),
		}

		// Create a new HTTP request (URL doesn't matter as the transport fails)
		req, err := http.NewRequest("GET", "https://example.com", nil)
		assert.NoError(t, err)

		// Test the Do method
		resp, err := client.Do(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
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

	req, err := http.NewRequest("GET", "https://example.com/retry", nil)
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

func (e *errorBody) Read(_ []byte) (int, error) {
	if e.readErr != nil {
		return 0, e.readErr
	}
	return 0, io.EOF
}

func (e *errorBody) Close() error {
	e.closed = true
	if e.closeErr != nil {
		return e.closeErr
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

type retryReadErrorBody struct {
	readErr error
	closed  bool
}

func (r *retryReadErrorBody) Read(_ []byte) (int, error) {
	return 0, r.readErr
}

func (r *retryReadErrorBody) Close() error {
	r.closed = true
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

	req, err := http.NewRequest("GET", "https://example.com/retry", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRLHTTPClient_ReturnsTimeoutErrorFromRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Inf, 0)
	client := NewRLClient(limiter)
	client.RetryConfig = NoRetries()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
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

	req, err := http.NewRequest("GET", "https://example.com", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.Nil(t, resp)
	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
}
