package add

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestDownloadProgressModelStartDownloadCmdMissingDownload(t *testing.T) {
	model := &downloadProgressModel{}
	msg := model.startDownloadCmd()()
	typed, ok := msg.(downloadFailedMsg)
	assert.True(t, ok)
	assert.ErrorContains(t, typed.err, "missing download handler")
}

func TestDownloadProgressModelStartDownloadCmdMissingSender(t *testing.T) {
	model := &downloadProgressModel{
		download: func(httpclient.Sender) (modinstall.EnsureResult, error) {
			return modinstall.EnsureResult{}, nil
		},
	}
	msg := model.startDownloadCmd()()
	typed, ok := msg.(downloadFailedMsg)
	assert.True(t, ok)
	assert.ErrorContains(t, typed.err, "missing bubble tea sender")
}

type testSender struct{}

func (testSender) Send(tea.Msg) {}

func TestDownloadProgressModelStartDownloadCmdSuccess(t *testing.T) {
	model := &downloadProgressModel{
		download: func(httpclient.Sender) (modinstall.EnsureResult, error) {
			return modinstall.EnsureResult{Reason: modinstall.EnsureReasonMissing}, nil
		},
		sender: testSender{},
	}
	msg := model.startDownloadCmd()()
	typed, ok := msg.(downloadFinishedMsg)
	assert.True(t, ok)
	assert.Equal(t, modinstall.EnsureReasonMissing, typed.result.Reason)
}

func TestDownloadProgressModelStartDownloadCmdReturnsError(t *testing.T) {
	model := &downloadProgressModel{
		download: func(httpclient.Sender) (modinstall.EnsureResult, error) {
			return modinstall.EnsureResult{}, errors.New("boom")
		},
		sender: testSender{},
	}
	msg := model.startDownloadCmd()()
	typed, ok := msg.(downloadFailedMsg)
	assert.True(t, ok)
	assert.ErrorContains(t, typed.err, "boom")
}

func TestDownloadProgressModelUpdateProgressAndFinish(t *testing.T) {
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})

	updated, cmd := model.Update(httpclient.DownloadProgressMsg{Downloaded: 512, Total: 1024, Ratio: 0.5})
	assert.Nil(t, cmd)
	assert.Equal(t, progressStatusInProgress, updated.(*downloadProgressModel).status)

	updated, cmd = model.Update(downloadFinishedMsg{result: modinstall.EnsureResult{Reason: modinstall.EnsureReasonMissing}})
	assert.NotNil(t, cmd)
	assert.Equal(t, progressStatusCompleted, updated.(*downloadProgressModel).status)
}

func TestDownloadProgressModelUpdateProgressError(t *testing.T) {
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})
	updated, cmd := model.Update(httpclient.DownloadProgressErrMsg{Err: errors.New("write failed")})
	assert.Nil(t, cmd)
	assert.ErrorContains(t, updated.(*downloadProgressModel).err, "write failed")
}

func TestDownloadProgressModelUpdateFailure(t *testing.T) {
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, errors.New("boom")
	})

	updated, cmd := model.Update(downloadFailedMsg{err: errors.New("boom")})
	assert.NotNil(t, cmd)
	assert.Equal(t, progressStatusFailed, updated.(*downloadProgressModel).status)
}

func TestDownloadProgressModelViewCompletedReturnsEmpty(t *testing.T) {
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})
	model.status = progressStatusCompleted

	assert.Equal(t, "", model.View())
}

func TestDownloadProgressModelViewStarted(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})

	viewText := model.View()
	assert.Contains(t, viewText, "cmd.add.progress.start")
}

func TestDownloadProgressModelViewInProgress(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})
	model.status = progressStatusInProgress
	model.ratio = 0.5
	model.downloaded = 512
	model.total = 1024

	viewText := model.View()
	assert.Contains(t, viewText, "Example (abc) [modrinth]")
	assert.Contains(t, viewText, "50%")
}

func TestDownloadProgressModelViewFailedReturnsEmpty(t *testing.T) {
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})
	model.status = progressStatusFailed

	assert.Equal(t, "", model.View())
}

func TestProgressLineRendersDetails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})
	model.ratio = 0.5
	model.downloaded = 512
	model.total = 1024

	line := model.progressLine()
	assert.Contains(t, line, "Example (abc) [modrinth]")
	assert.Contains(t, line, "50%")
}

func TestProgressLineWrapsAsciiBar(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})
	model.ratio = 0.5
	model.downloaded = 512
	model.total = 1024

	line := model.progressLine()
	barLine := model.progress.ViewAs(model.ratio)
	assert.Contains(t, line, "["+barLine+"]")
}

func TestDownloadProgressModelUpdatePassesThroughUnknownMessage(t *testing.T) {
	model := newDownloadProgressModel("Example", "abc", models.MODRINTH, view.ColorDisabled, func(httpclient.Sender) (modinstall.EnsureResult, error) {
		return modinstall.EnsureResult{}, nil
	})

	updated, cmd := model.Update(tea.KeyMsg{})
	assert.Nil(t, cmd)
	assert.Equal(t, model.status, updated.(*downloadProgressModel).status)
}
