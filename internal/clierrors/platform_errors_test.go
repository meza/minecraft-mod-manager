package clierrors

import (
	"context"
	"errors"
	"net/url"
	"syscall"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSummarizePlatformErrorUsesResponseError(t *testing.T) {
	responseErr := &httpclient.ResponseError{
		Method:     "GET",
		URL:        "https://example.invalid",
		StatusCode: 403,
	}

	summary, ok := SummarizePlatformError(responseErr, models.CURSEFORGE)
	assert.True(t, ok)
	assert.Equal(t, i18n.T("cmd.platform.error.reason.auth", &i18n.Tvars{
		Data: &i18n.TData{
			"token":    "CURSEFORGE_API_KEY",
			"platform": string(models.CURSEFORGE),
		},
	}), summary.Reason)
	assert.Contains(t, summary.DebugDetails, "status=403")
}

func TestSummarizePlatformErrorTimeout(t *testing.T) {
	timeoutErr := httpclient.WrapTimeoutError(context.DeadlineExceeded)

	summary, ok := SummarizePlatformError(timeoutErr, models.MODRINTH)
	assert.True(t, ok)
	assert.Equal(t, i18n.T("cmd.platform.error.reason.timeout", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(models.MODRINTH)},
	}), summary.Reason)
	assert.Contains(t, summary.DebugDetails, "timed out")
}

func TestSummarizePlatformErrorConnection(t *testing.T) {
	connErr := &url.Error{Op: "Get", URL: "https://example.invalid", Err: syscall.ECONNREFUSED}

	summary, ok := SummarizePlatformError(connErr, models.MODRINTH)
	assert.True(t, ok)
	assert.Equal(t, i18n.T("cmd.platform.error.reason.connection", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(models.MODRINTH)},
	}), summary.Reason)
	assert.Contains(t, summary.DebugDetails, "connection refused")
}

func TestSummarizePlatformErrorNil(t *testing.T) {
	summary, ok := SummarizePlatformError(nil, models.CURSEFORGE)
	assert.False(t, ok)
	assert.Equal(t, PlatformErrorSummary{}, summary)
}

func TestSummarizePlatformErrorUnknown(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	summary, ok := SummarizePlatformError(errors.New("boom"), models.MODRINTH)
	assert.True(t, ok)
	assert.Contains(t, summary.Reason, "cmd.platform.error.reason.unknown")
	assert.Contains(t, summary.DebugDetails, "boom")
}

func TestApiTokenNameFallback(t *testing.T) {
	assert.Equal(t, "API_KEY", apiTokenName(models.Platform("other")))
}

func TestApiTokenNameModrinth(t *testing.T) {
	assert.Equal(t, "MODRINTH_API_KEY", apiTokenName(models.MODRINTH))
}

func TestReasonForStatusCodeVariants(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cases := []struct {
		name       string
		statusCode int
		expected   string
	}{
		{name: "auth", statusCode: 401, expected: "cmd.platform.error.reason.auth"},
		{name: "rate_limit", statusCode: 429, expected: "cmd.platform.error.reason.rate_limited"},
		{name: "not_found", statusCode: 404, expected: "cmd.platform.error.reason.not_found"},
		{name: "server", statusCode: 500, expected: "cmd.platform.error.reason.server_error"},
		{name: "bad_request", statusCode: 400, expected: "cmd.platform.error.reason.bad_request"},
		{name: "default", statusCode: 302, expected: "cmd.platform.error.reason.unknown"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := reasonForStatusCode(testCase.statusCode, models.CURSEFORGE)
			assert.Contains(t, result, testCase.expected)
		})
	}
}
