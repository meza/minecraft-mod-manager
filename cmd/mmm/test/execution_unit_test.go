package test

import (
	"errors"
	"io"
	"net/http"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestLogEventsSkipsWhenLoggerNil(t *testing.T) {
	err := logEvents(testDeps{}, []logEvent{{Kind: logEventKindDebug, Message: "noop"}})
	require.NoError(t, err)
}

func TestTestExecSenderSendNoopsWhenNil(t *testing.T) {
	sender := testExecSender{}

	require.NotPanics(t, func() {
		sender.Send(testStartMsg{})
	})
}

func TestTestExecSenderSendDispatchesMessage(t *testing.T) {
	var received tea.Msg
	sender := testExecSender{
		send: func(msg tea.Msg) {
			received = msg
		},
	}

	sender.Send(testStartMsg{})

	assert.IsType(t, testStartMsg{}, received)
}

func TestLogEventsSkipsUnknownKind(t *testing.T) {
	deps := testDeps{
		logger: logger.New(io.Discard, io.Discard, false, true),
	}
	err := logEvents(deps, []logEvent{{Kind: logEventKind(99), Message: "noop"}})
	require.NoError(t, err)
}

func TestFetchFailureReasonUsesSummaryReason(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fetchErr := &httpclient.ResponseError{StatusCode: http.StatusTooManyRequests}

	reason := fetchFailureReason(fetchErr, models.MODRINTH)
	assert.Contains(t, reason, i18n.T("cmd.platform.error.reason.rate_limited", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(models.MODRINTH)},
	}))
}

func TestFetchFailureReasonFallbacksToUnknown(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	reason := fetchFailureReason(errors.New("boom"), models.CURSEFORGE)
	assert.Contains(t, reason, i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(models.CURSEFORGE)},
	}))
}

func TestFetchFailureReasonHandlesNilError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	reason := fetchFailureReason(nil, models.MODRINTH)
	assert.Contains(t, reason, i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(models.MODRINTH)},
	}))
}
