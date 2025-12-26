package httpclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResponseErrorCapturesDetails(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/resource", nil)
	assert.NoError(t, err)

	body := strings.NewReader("  {\"error\":\"nope\"}\n")
	response := &http.Response{
		StatusCode: http.StatusForbidden,
		Header: http.Header{
			"X-Ratelimit-Remaining": []string{"7"},
			"X-Ratelimit-Reset":     []string{"1234"},
			"Retry-After":           []string{"60"},
		},
		Body:    io.NopCloser(body),
		Request: request,
	}

	responseError := NewResponseError(response)
	assert.Equal(t, http.MethodGet, responseError.Method)
	assert.Equal(t, "https://example.invalid/resource", responseError.URL)
	assert.Equal(t, http.StatusForbidden, responseError.StatusCode)
	assert.Equal(t, "{\"error\":\"nope\"}", responseError.BodySnippet)
	assert.Equal(t, "7", responseError.RateLimit.Remaining)
	assert.Equal(t, "1234", responseError.RateLimit.Reset)
	assert.Equal(t, "60", responseError.RateLimit.RetryAfter)
}

func TestResponseErrorMessageIncludesRateLimit(t *testing.T) {
	responseError := &ResponseError{
		Method:      http.MethodPost,
		URL:         "https://example.invalid/endpoint",
		StatusCode:  http.StatusTooManyRequests,
		BodySnippet: "rate limited",
		RateLimit: RateLimitInfo{
			Remaining:  "0",
			Reset:      "10",
			RetryAfter: "5",
		},
	}

	message := responseError.Error()
	assert.Contains(t, message, "POST https://example.invalid/endpoint")
	assert.Contains(t, message, "status=429")
	assert.Contains(t, message, "body=")
	assert.Contains(t, message, "ratelimit=remaining:0 reset:10 retry_after:5")
}

func TestExtractResponseError(t *testing.T) {
	responseErr := &ResponseError{StatusCode: http.StatusBadRequest}

	parsed, ok := ExtractResponseError(responseErr)
	assert.True(t, ok)
	assert.Equal(t, responseErr, parsed)
}

func TestExtractResponseErrorReturnsFalse(t *testing.T) {
	parsed, ok := ExtractResponseError(errors.New("no response"))
	assert.False(t, ok)
	assert.Nil(t, parsed)
}

func TestNewResponseErrorNilResponse(t *testing.T) {
	responseErr := NewResponseError(nil)
	assert.Equal(t, 0, responseErr.StatusCode)
	assert.Equal(t, "", responseErr.Method)
	assert.Equal(t, "", responseErr.URL)
}

func TestNewResponseErrorHandlesNilBody(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       nil,
	}

	responseErr := NewResponseError(response)
	assert.Equal(t, http.StatusBadGateway, responseErr.StatusCode)
	assert.Equal(t, "", responseErr.BodySnippet)
}

func TestNewResponseErrorHandlesReadError(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/read", nil)
	assert.NoError(t, err)

	response := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       readErrorCloser{err: errors.New("read failed")},
		Request:    request,
	}

	responseErr := NewResponseError(response)
	assert.Contains(t, responseErr.BodySnippet, "failed to read response body")
}

func TestResponseErrorEmptyFallback(t *testing.T) {
	responseErr := &ResponseError{
		Method:     http.MethodGet,
		URL:        "https://example.invalid/ratelimit",
		StatusCode: http.StatusTooManyRequests,
		RateLimit: RateLimitInfo{
			Remaining: "",
			Reset:     "10",
		},
	}

	message := responseErr.Error()
	assert.Contains(t, message, "ratelimit=remaining:- reset:10 retry_after:-")
}

func TestReadBodySnippetTruncates(t *testing.T) {
	longBody := strings.Repeat("a", responseBodySnippetLimit+10)
	reader := io.NopCloser(strings.NewReader(longBody))

	snippet, err := readBodySnippet(reader)
	assert.NoError(t, err)
	assert.Len(t, snippet, responseBodySnippetLimit)
}

func TestReadBodySnippetReturnsError(t *testing.T) {
	reader := readErrorCloser{err: errors.New("read failed")}

	_, err := readBodySnippet(reader)
	assert.Error(t, err)
}

func TestReadBodySnippetReturnsDrainError(t *testing.T) {
	reader := &drainErrorCloser{
		remaining: responseBodySnippetLimit + 1,
		err:       errors.New("drain failed"),
	}

	_, err := readBodySnippet(reader)
	assert.Error(t, err)
}

type readErrorCloser struct {
	err error
}

func (closer readErrorCloser) Read([]byte) (int, error) {
	return 0, closer.err
}

func (closer readErrorCloser) Close() error {
	return nil
}

type drainErrorCloser struct {
	remaining int
	err       error
}

func (closer *drainErrorCloser) Read(buffer []byte) (int, error) {
	if closer.remaining == 0 {
		return 0, closer.err
	}
	toCopy := len(buffer)
	if closer.remaining < toCopy {
		toCopy = closer.remaining
	}
	for i := 0; i < toCopy; i++ {
		buffer[i] = 'a'
	}
	closer.remaining -= toCopy
	return toCopy, nil
}

func (closer *drainErrorCloser) Close() error {
	return nil
}
