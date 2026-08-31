package install

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestInstallModelUpdateWindowSizeIgnoresZeroValues(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowW = 80
	model.windowH = 24

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	result := updated.(*installModel)

	assert.Equal(t, 80, result.windowW)
	assert.Equal(t, 24, result.windowH)
}

func TestInstallModelUpdateWindowSizeUpdatesDimensions(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 33})
	result := updated.(*installModel)

	assert.Equal(t, 120, result.windowW)
	assert.Equal(t, 33, result.windowH)
}

func TestInstallModelViewRendersStates(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	model := newInstallModel(context.Background(), view.ColorDisabled, items, indexByKey, nil, nil, nil)

	model.state = installViewRunning
	assert.Contains(t, model.View(), "cmd.install.header.success")

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

func TestInstallModelUpdateKeyScrollsViewport(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.viewport.Height = 1
	model.viewport.Width = 10
	model.viewport.SetContent("one\ntwo\nthree")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := updated.(*installModel)
	assert.Equal(t, 1, result.viewport.YOffset)
	assert.True(t, result.userScroll)
}

func TestInstallModelUpdateMouseScrollsViewport(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.viewport.MouseWheelEnabled = true
	model.viewport.Height = 1
	model.viewport.Width = 10
	model.viewport.SetContent("one\ntwo\nthree")

	updated, _ := model.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	result := updated.(*installModel)
	assert.Greater(t, result.viewport.YOffset, 0)
	assert.True(t, result.userScroll)
}

func TestInstallModelUpdateViewportKeepsOffsetWhenUserScrolled(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowW = 10
	model.viewport.Height = 1
	model.viewport.Width = 10
	model.viewport.SetContent("one\ntwo\nthree")
	model.viewport.YOffset = 1
	model.userScroll = true

	model.updateViewport("one\ntwo\nthree", 1)

	assert.Equal(t, 1, model.viewport.YOffset)
}

func TestInstallModelUpdateViewportUsesContentHeight(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)

	model.updateViewport("one", 1)

	assert.Equal(t, 1, model.viewport.Height)
}

func TestInstallModelUpdateViewportUsesContentWidthWhenWindowUnknown(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.viewport.Width = 0

	model.updateViewport("alpha\nbeta", 2)

	assert.Equal(t, lipgloss.Width("alpha\nbeta"), model.viewport.Width)
}

func TestInstallModelFooterLinesWithHeaderWhenOffscreen(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 3

	layout := installViewLayout{
		header:      "Header",
		listLines:   []string{"one", "two", "three"},
		footerLines: []string{"Footer"},
	}

	footerLines := model.footerLinesWithHeaderIfNeeded(layout, 0, 3)

	assert.Equal(t, []string{"Header", "Footer"}, footerLines)
}

func TestInstallModelFooterLinesWithHeaderWhenSpaceFits(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 10

	layout := installViewLayout{
		header:      "Header",
		listLines:   []string{"one"},
		footerLines: []string{"Footer"},
	}

	footerLines := model.footerLinesWithHeaderIfNeeded(layout, 0, 1)

	assert.Equal(t, []string{"Footer"}, footerLines)
}

func TestInstallModelFooterLinesWithHeaderSkipsWhenUnavailable(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 0

	layout := installViewLayout{
		header:      "Header",
		listLines:   []string{"one"},
		footerLines: []string{"Footer"},
	}

	footerLines := model.footerLinesWithHeaderIfNeeded(layout, 0, 1)

	assert.Equal(t, []string{"Header", "Footer"}, footerLines)
}

func TestInstallModelFooterLinesWithHeaderSkipsWhenFooterEmptyAndWindowUnknown(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 0

	layout := installViewLayout{
		header:      "Header",
		listLines:   []string{"one"},
		footerLines: nil,
	}

	footerLines := model.footerLinesWithHeaderIfNeeded(layout, 0, 1)

	assert.Empty(t, footerLines)
}

func TestInstallModelFooterLinesWithHeaderSkipsWhenHeaderBlank(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 3

	layout := installViewLayout{
		header:      " ",
		listLines:   []string{"one"},
		footerLines: []string{"Footer"},
	}

	footerLines := model.footerLinesWithHeaderIfNeeded(layout, 0, 1)

	assert.Equal(t, []string{"Footer"}, footerLines)
}

func TestInstallModelRenderInstallViewWithStickyHeaderSkipsWhenHeaderBlank(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 10

	output := "one\ntwo"
	assert.Equal(t, output, model.renderInstallViewWithStickyHeader(output, ""))
}

func TestInstallModelRenderInstallViewWithStickyHeaderSkipsWhenFits(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 10

	output := "Header\none"
	assert.Equal(t, output, model.renderInstallViewWithStickyHeader(output, "Header"))
}

func TestInstallModelRenderInstallViewWithStickyHeaderEchoesWhenOverflow(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, nil, nil, nil, nil, nil)
	model.windowH = 2

	output := "Header\none\ntwo"
	expected := view.RenderViewSections([]string{output, "Header"}, view.SectionSeparatorParagraph)
	assert.Equal(t, expected, model.renderInstallViewWithStickyHeader(output, "Header"))
}

func TestRenderInstallViewWithoutViewportIncludesFooter(t *testing.T) {
	layout := installViewLayout{
		header:      "Header",
		listLines:   []string{"one"},
		footerLines: []string{"Footer"},
		separator:   installSeparatorParagraph,
	}
	content := "Header\none"
	expected := view.RenderViewSections([]string{content, "Footer"}, view.SectionSeparatorParagraph)
	assert.Equal(t, expected, renderInstallViewWithoutViewport(layout, view.SectionSeparatorParagraph))
}

func TestRenderInstallViewWithoutViewportSkipsFooterWhenEmpty(t *testing.T) {
	layout := installViewLayout{
		header:    "Header",
		listLines: []string{"one"},
		separator: installSeparatorParagraph,
	}
	expected := "Header\none"
	assert.Equal(t, expected, renderInstallViewWithoutViewport(layout, view.SectionSeparatorParagraph))
}

func TestIsViewportScrollKeyDetectsNavigationKeys(t *testing.T) {
	assert.True(t, isViewportScrollKey(tea.KeyMsg{Type: tea.KeyPgDown}))
	assert.True(t, isViewportScrollKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}))
	assert.False(t, isViewportScrollKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))
}

func TestIsViewportScrollMouseDetectsWheelPress(t *testing.T) {
	assert.True(t, isViewportScrollMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp}))
	assert.False(t, isViewportScrollMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}))
	assert.False(t, isViewportScrollMouse(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonWheelDown}))
}
