package httpClient

import (
	"context"
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"golang.org/x/time/rate"
	"io"
	"net/http"
	"time"
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
	ctx := context.WithValue(context.Background(), "url", request.URL)
	details := perf.PerformanceDetails{
		"url":    request.URL.String(),
		"method": request.Method,
		"host":   request.URL.Host,
	}
	region := perf.StartRegionWithDetails("net.http.request", &details)
	defer region.End()
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
		attemptDetails := perf.PerformanceDetails{
			"attempt": attempt,
			"url":     request.URL.String(),
		}
		attemptRegion := perf.StartRegionWithDetails("net.http.request.attempt", &attemptDetails)

		waitRegion := perf.StartRegionWithDetails("net.http.ratelimit.wait", &attemptDetails)
		err = client.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
		waitRegion.End()
		if err != nil {
			attemptDetails["success"] = false
			details["success"] = false
			attemptRegion.End()
			return nil, fmt.Errorf("rate limit burst exceeded %w", err)
		}

		response, err = client.client.Do(request)
		if err != nil {
			attemptDetails["success"] = false
			details["success"] = false
			attemptRegion.End()
			return nil, err
		}

		// Check if the response status is a server error (5xx)
		if response.StatusCode >= 500 && response.StatusCode < 600 {
			if attempt < retryConfig.MaxRetries {
				attemptDetails["success"] = false
				drainAndClose(response.Body)
				time.Sleep(retryConfig.Interval)
				attemptRegion.End()
				continue
			}
		}

		attemptDetails["success"] = true
		attemptDetails["status"] = response.StatusCode
		// If the response is successful or a non-retryable error occurs, return the response or error
		attemptRegion.End()
		break
	}

	details["success"] = err == nil
	if response != nil {
		details["status"] = response.StatusCode
	}
	return response, err
}

func NewRLClient(limiter *rate.Limiter) *RLHTTPClient {
	client := &RLHTTPClient{
		client:      http.DefaultClient,
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

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
