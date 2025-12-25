package httpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const responseBodySnippetLimit = 2048

type RateLimitInfo struct {
	Remaining  string
	Reset      string
	RetryAfter string
}

func (info RateLimitInfo) IsZero() bool {
	return info.Remaining == "" && info.Reset == "" && info.RetryAfter == ""
}

type ResponseError struct {
	Method      string
	URL         string
	StatusCode  int
	BodySnippet string
	RateLimit   RateLimitInfo
}

func (responseError *ResponseError) Error() string {
	message := fmt.Sprintf("request failed: %s %s status=%d", responseError.Method, responseError.URL, responseError.StatusCode)
	if strings.TrimSpace(responseError.BodySnippet) != "" {
		message = fmt.Sprintf("%s body=%q", message, responseError.BodySnippet)
	}
	if !responseError.RateLimit.IsZero() {
		message = fmt.Sprintf("%s ratelimit=remaining:%s reset:%s retry_after:%s",
			message,
			emptyFallback(responseError.RateLimit.Remaining),
			emptyFallback(responseError.RateLimit.Reset),
			emptyFallback(responseError.RateLimit.RetryAfter),
		)
	}
	return message
}

func NewResponseError(response *http.Response) *ResponseError {
	if response == nil {
		return &ResponseError{Method: "", URL: "", StatusCode: 0}
	}

	snippet, err := readBodySnippet(response.Body, responseBodySnippetLimit)
	if err != nil {
		snippet = fmt.Sprintf("failed to read response body: %v", err)
	}

	request := response.Request
	method := ""
	urlValue := ""
	if request != nil && request.URL != nil {
		method = request.Method
		urlValue = request.URL.String()
	}

	return &ResponseError{
		Method:      method,
		URL:         urlValue,
		StatusCode:  response.StatusCode,
		BodySnippet: snippet,
		RateLimit:   rateLimitInfo(response.Header),
	}
}

func ExtractResponseError(err error) (*ResponseError, bool) {
	var responseErr *ResponseError
	if errors.As(err, &responseErr) {
		return responseErr, true
	}
	return nil, false
}

func rateLimitInfo(header http.Header) RateLimitInfo {
	return RateLimitInfo{
		Remaining:  header.Get("X-Ratelimit-Remaining"),
		Reset:      header.Get("X-Ratelimit-Reset"),
		RetryAfter: header.Get("Retry-After"),
	}
}

func readBodySnippet(body io.ReadCloser, limit int) (string, error) {
	if body == nil {
		return "", nil
	}

	limited := io.LimitReader(body, int64(limit+1))
	content, readErr := io.ReadAll(limited)
	if readErr != nil {
		return "", readErr
	}

	if _, drainErr := io.Copy(io.Discard, body); drainErr != nil {
		return "", drainErr
	}

	if len(content) > limit {
		content = content[:limit]
	}

	snippet := strings.TrimSpace(string(content))
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.ReplaceAll(snippet, "\r", " ")
	snippet = strings.Join(strings.Fields(snippet), " ")
	return snippet, nil
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
