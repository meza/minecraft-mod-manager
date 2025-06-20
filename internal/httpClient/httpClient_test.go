package httpClient

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRLHTTPClient_Fetch(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
			w.Write([]byte("OK"))
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
			w.Write([]byte("OK"))
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
				w.Write([]byte("OK"))
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
			w.Write([]byte("OK"))
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
