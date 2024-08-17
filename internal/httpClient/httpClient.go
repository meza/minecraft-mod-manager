package httpClient

import (
	"context"
	"golang.org/x/time/rate"
	"net/http"
)

type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

type RLHTTPClient struct {
	client      *http.Client
	Ratelimiter *rate.Limiter
}

func (client *RLHTTPClient) Do(request *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := client.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	response, err := client.client.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func NewRLClient(limiter *rate.Limiter) *RLHTTPClient {
	client := &RLHTTPClient{
		client:      http.DefaultClient,
		Ratelimiter: limiter,
	}
	return client
}
