package httpClient

import (
	"golang.org/x/time/rate"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
}
