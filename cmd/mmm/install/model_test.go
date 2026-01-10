package install

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestInstallExecSenderSendHandlesNil(t *testing.T) {
	sender := installExecSender{}
	sender.Send(installItemSuccessMsg{})
}

func TestInstallExecSenderSendCallsHandler(t *testing.T) {
	called := false
	sender := installExecSender{send: func(tea.Msg) { called = true }}
	sender.Send(installItemSuccessMsg{})
	assert.True(t, called)
}

func TestInstallModelInitQuitsWhenMissingSender(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, func(context.Context, httpclient.Sender) installExecutionOutcome {
		return installExecutionOutcome{}
	}, nil)

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestInstallModelInitQuitsWhenMissingRunner(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestInstallModelStartInstallCmdReturnsFailureWhenRunnerMissing(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	msg := model.startInstallCmd()()
	typed, ok := msg.(installExecutionFinishedMsg)
	assert.True(t, ok)
	assert.ErrorContains(t, typed.outcome.err, "missing execution runner")
	assert.Equal(t, installExecutionErrorUnknown, typed.outcome.errType)
}

func TestInstallModelUpdateAppliesItemMessages(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	model := newInstallModel(context.Background(), view.ColorDisabled, items, indexByKey, nil, nil, nil)
	key := installModKey(cfg.Mods[0])

	model.Update(installItemProgressMsg{key: key, progress: httpclient.DownloadProgressMsg{Downloaded: 10, Total: 20, Ratio: 0.5}})
	item := model.items[indexByKey[key]]
	assert.Equal(t, installItemDownloading, item.Status)
	assert.Equal(t, int64(10), item.Progress.downloaded)

	model.Update(installItemProgressErrMsg{key: key, err: errors.New("boom")})
	item = model.items[indexByKey[key]]
	assert.Contains(t, item.FailureReason, "boom")

	model.Update(installItemProgressErrMsg{key: key, err: nil})
	item = model.items[indexByKey[key]]
	assert.Contains(t, item.FailureReason, "boom")

	model.Update(installItemFailureMsg{key: key, reason: "failed"})
	model.Update(installItemProgressErrMsg{key: key, err: errors.New("ignored")})
	item = model.items[indexByKey[key]]
	assert.Equal(t, "failed", item.FailureReason)

	model.Update(installItemSuccessMsg{key: key, displayName: "Alpha Updated"})
	item = model.items[indexByKey[key]]
	assert.Equal(t, installItemSuccess, item.Status)
	assert.Equal(t, "Alpha Updated", item.DisplayName)

	model.Update(installItemAbortedMsg{key: key})
	item = model.items[indexByKey[key]]
	assert.Equal(t, installItemAborted, item.Status)
}

func TestInstallModelUpdateSetsFinalState(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)

	_, cmd := model.Update(installExecutionFinishedMsg{outcome: installExecutionOutcome{errType: installExecutionErrorDownload}})
	assert.NotNil(t, cmd)
	assert.Equal(t, installViewDownloadFailed, model.state)
}

func TestInstallModelViewRendersStates(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	model := newInstallModel(context.Background(), view.ColorDisabled, items, indexByKey, nil, nil, nil)

	model.state = installViewRunning
	assert.Contains(t, model.View(), "cmd.install.header.running")

	model.state = installViewSuccess
	assert.Contains(t, model.View(), "cmd.install.summary.success")

	model.state = installViewDownloadFailed
	assert.Contains(t, model.View(), "cmd.install.summary.download_failed")

	model.state = installViewWriteLockFailed
	model.outcome.lockPath = "/lock"
	assert.Contains(t, model.View(), "cmd.install.summary.write_failed_abort")

	model.state = installViewWriteConfigFailed
	model.outcome.configPath = "/config"
	assert.Contains(t, model.View(), "cmd.install.summary.write_failed_abort")

	model.state = installViewCanceled
	assert.Contains(t, model.View(), "cmd.install.summary.canceled")

	model.state = installViewFailed
	model.outcome.err = errors.New("boom")
	assert.Contains(t, model.View(), "cmd.install.error.failed")
}

func TestViewStateFromOutcome(t *testing.T) {
	assert.Equal(t, installViewDownloadFailed, viewStateFromOutcome(installExecutionErrorDownload))
	assert.Equal(t, installViewWriteLockFailed, viewStateFromOutcome(installExecutionErrorWriteLock))
	assert.Equal(t, installViewWriteConfigFailed, viewStateFromOutcome(installExecutionErrorWriteConfig))
	assert.Equal(t, installViewCanceled, viewStateFromOutcome(installExecutionErrorCanceled))
	assert.Equal(t, installViewFailed, viewStateFromOutcome(installExecutionErrorUnknown))
	assert.Equal(t, installViewSuccess, viewStateFromOutcome(installExecutionErrorNone))
}

func TestInstallModelUpdateIgnoresUnknownMessage(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	updated, cmd := model.Update(tea.KeyMsg{})
	assert.Nil(t, cmd)
	assert.Equal(t, model, updated)
}

func TestInstallModelUpdateSkipsUnknownKey(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	model := newInstallModel(context.Background(), view.ColorDisabled, items, indexByKey, nil, nil, nil)

	model.Update(installItemSuccessMsg{key: "missing"})
	item := model.items[indexByKey[installModKey(cfg.Mods[0])]]
	assert.Equal(t, installItemPending, item.Status)
}

func TestInstallModelUpdateCancelsOnCtrlC(t *testing.T) {
	canceled := false
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, func() {
		canceled = true
	}, nil, nil)

	model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, canceled)
}

func TestInstallModelUpdateCancelsOnQ(t *testing.T) {
	canceled := false
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, func() {
		canceled = true
	}, nil, nil)

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.True(t, canceled)
}

func TestInstallModelUpdateCancelsOnEsc(t *testing.T) {
	canceled := false
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, func() {
		canceled = true
	}, nil, nil)

	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, canceled)
}

func TestInstallModelUpdateIgnoresUnknownMessageType(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Equal(t, model, updated)
}
