package change

import (
	"context"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestChangeModelStartChangeCmdMissingSender(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})

	msg := model.startChangeCmd()()
	typed := msg.(changeExecMessage)
	assert.Equal(t, changeStageDownloadFailed, typed.outcome.Stage)
}

func TestChangeExecSenderSendDispatches(t *testing.T) {
	called := false
	sender := changeExecSender{
		send: func(tea.Msg) {
			called = true
		},
	}

	sender.Send(changeSwitchingStartedMsg{})
	assert.True(t, called)
}

func TestChangeExecSenderSendNoopWhenNil(t *testing.T) {
	sender := changeExecSender{}
	assert.NotPanics(t, func() {
		sender.Send(changeSwitchingStartedMsg{})
	})
}

func TestChangeModelInitReturnsQuitWithoutRunner(t *testing.T) {
	model := &changeModel{}
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestChangeModelInitReturnsQuitWhenSenderMissing(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	msg := model.Init()()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestNewChangeModelUsesLineSpinnerWhenUnicodeDisabled(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	assert.Equal(t, spinner.Line, model.spinner.Spinner)
}

func TestNewChangeModelUsesDotSpinnerWhenUnicodeEnabled(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	assert.Equal(t, spinner.Dot, model.spinner.Spinner)
}

func TestChangeModelUpdateHandlesMessages(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.bindSender(func(tea.Msg) {})

	updated, _ := model.Update(changeCompatResultMsg{key: "modrinth:alpha", supported: true, resolvedName: "Alpha"})
	assert.Equal(t, changeCompatSupported, updated.(*changeModel).items[0].CompatStatus)
	assert.Equal(t, changeDownloadQueued, updated.(*changeModel).items[0].DownloadStatus)

	updated, _ = model.Update(changeCompatCheckingMsg{key: "modrinth:alpha"})
	assert.Equal(t, changeCompatChecking, updated.(*changeModel).items[0].CompatStatus)

	updated, _ = model.Update(changeDownloadProgressMsg{
		key:      "modrinth:alpha",
		progress: httpclient.DownloadProgressMsg{Downloaded: 10, Total: 20, Ratio: 0.5},
	})
	assert.Equal(t, changeDownloadInProgress, updated.(*changeModel).items[0].DownloadStatus)

	updated, _ = model.Update(changeDownloadProgressErrMsg{key: "modrinth:alpha", err: assert.AnError})
	assert.NotEmpty(t, updated.(*changeModel).items[0].ErrorReason)

	updated, _ = model.Update(changeDownloadFinishedMsg{key: "modrinth:alpha"})
	assert.Equal(t, changeDownloadSucceeded, updated.(*changeModel).items[0].DownloadStatus)

	updated, _ = model.Update(changeDownloadFailedMsg{key: "modrinth:alpha", err: assert.AnError})
	assert.Equal(t, changeDownloadFailed, updated.(*changeModel).items[0].DownloadStatus)

	updated, _ = model.Update(changeSwitchingStartedMsg{})
	assert.Equal(t, changeStageSwitching, updated.(*changeModel).stage)

	updated, _ = model.Update(changeSwitchStartedMsg{key: "modrinth:alpha"})
	assert.Equal(t, changeSwitchInProgress, updated.(*changeModel).items[0].SwitchStatus)

	updated, _ = model.Update(changeSwitchFinishedMsg{key: "modrinth:alpha"})
	assert.Equal(t, changeSwitchSucceeded, updated.(*changeModel).items[0].SwitchStatus)

	updated, _ = model.Update(changeSwitchFailedMsg{key: "modrinth:alpha", err: assert.AnError})
	assert.Equal(t, changeSwitchFailed, updated.(*changeModel).items[0].SwitchStatus)

	updated, _ = model.Update(changeSwitchSkippedMsg{key: "modrinth:alpha"})
	assert.Equal(t, changeSwitchSkipped, updated.(*changeModel).items[0].SwitchStatus)
}

func TestChangeModelApplyCompatResultUnsupported(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applyCompatResult(changeCompatResultMsg{key: "modrinth:alpha", supported: false, skipped: true, resolvedName: "Alpha"})
	item := model.items[0]
	assert.Equal(t, changeCompatUnsupported, item.CompatStatus)
	assert.True(t, item.Skipped)
	assert.Equal(t, "Alpha", item.DisplayName)
}

func TestChangeModelApplyCompatResultQueuesDownload(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applyCompatResult(changeCompatResultMsg{key: "modrinth:alpha", supported: true, resolvedName: "Alpha"})
	assert.Equal(t, changeDownloadQueued, model.items[0].DownloadStatus)
}

func TestChangeModelApplyDownloadErrorDoesNotOverride(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, ErrorReason: "existing"}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applyDownloadError("modrinth:alpha", assert.AnError)
	assert.Equal(t, "existing", model.items[0].ErrorReason)
}

func TestChangeModelApplyDownloadFinishedUpdatesName(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, DisplayName: "Old"}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applyDownloadFinished(changeDownloadFinishedMsg{key: "modrinth:alpha", resolvedName: "New"})
	assert.Equal(t, changeDownloadSucceeded, model.items[0].DownloadStatus)
	assert.Equal(t, "New", model.items[0].DisplayName)
}

func TestChangeModelApplySwitchFinishedSkipsSkippedItems(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, SwitchStatus: changeSwitchSkipped}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applySwitchFinished(changeSwitchFinishedMsg{key: "modrinth:alpha"})
	assert.Equal(t, changeSwitchSkipped, model.items[0].SwitchStatus)
}

func TestChangeModelUpdateExecMessageQuits(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.bindSender(func(tea.Msg) {})

	updated, cmd := model.Update(changeExecMessage{outcome: changeOutcome{Stage: changeStageSuccess}})
	assert.Equal(t, changeStageSuccess, updated.(*changeModel).stage)
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(changeFinalizeMsg)
	assert.True(t, ok)
}

func TestChangeModelFinalizeMessageQuits(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})

	_, cmd := model.Update(changeFinalizeMsg{})
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestChangeModelUpdateExecMessageUpdatesItems(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.bindSender(func(tea.Msg) {})

	updatedItems := []changeItem{{
		Mod:          models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName:  "Alpha",
		CompatStatus: changeCompatSupported,
	}}

	updated, _ := model.Update(changeExecMessage{outcome: changeOutcome{
		Stage: changeStageCompatibilityFailed,
		Items: updatedItems,
	}})

	assert.Equal(t, updatedItems, updated.(*changeModel).items)
}

func TestChangeModelUpdateIgnoresUnknownMessage(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})
	model.bindSender(func(tea.Msg) {})

	updated, cmd := model.Update(tea.WindowSizeMsg{})
	assert.Equal(t, changeStageRunning, updated.(*changeModel).stage)
	assert.Nil(t, cmd)
}

func TestChangeModelUpdateCompatFailureMessageSetsStage(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})
	model.bindSender(func(tea.Msg) {})

	updated, _ := model.Update(changeCompatFailureMsg{})
	assert.Equal(t, changeStageCompatibilityFailureDetected, updated.(*changeModel).stage)
}

func TestChangeModelUpdateHandlesSpinnerTick(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	updated, cmd := model.Update(spinner.TickMsg{})
	assert.NotNil(t, updated)
	assert.NotNil(t, cmd)
}

func TestChangeModelApplyMethodsIgnoreUnknownKey(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applyCompatResult(changeCompatResultMsg{key: "missing", supported: true})
	model.applyCompatChecking(changeCompatCheckingMsg{key: "missing"})
	model.applyDownloadProgress(changeDownloadProgressMsg{key: "missing"})
	model.applyDownloadError("missing", assert.AnError)
	model.applyDownloadFinished(changeDownloadFinishedMsg{key: "missing"})
	model.applyDownloadFailed(changeDownloadFailedMsg{key: "missing"})
	model.applySwitchStarted(changeSwitchStartedMsg{key: "missing"})
	model.applySwitchFinished(changeSwitchFinishedMsg{key: "missing"})
	model.applySwitchFailed(changeSwitchFailedMsg{key: "missing"})
	model.applySwitchSkipped(changeSwitchSkippedMsg{key: "missing"})

	assert.Equal(t, changeCompatPending, model.items[0].CompatStatus)
	assert.Equal(t, changeDownloadPending, model.items[0].DownloadStatus)
	assert.Equal(t, changeSwitchPending, model.items[0].SwitchStatus)
}

func TestChangeModelApplyDownloadFailedNilError(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})

	model.applyDownloadFailed(changeDownloadFailedMsg{key: "modrinth:alpha", err: nil})
	assert.Equal(t, changeDownloadFailed, model.items[0].DownloadStatus)
	assert.Empty(t, model.items[0].ErrorReason)
}

func TestRenderCompatibilityLineVariants(t *testing.T) {
	supported := renderCompatibilityLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, DisplayName: "Alpha", CompatStatus: changeCompatSupported})
	assert.Contains(t, supported, "Alpha (alpha) [modrinth]")

	queued := renderCompatibilityLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:            models.Mod{ID: "beta", Type: models.MODRINTH},
		DisplayName:    "Beta",
		CompatStatus:   changeCompatSupported,
		DownloadStatus: changeDownloadQueued,
	})
	assert.Contains(t, queued, view.PendingIcon(view.ColorDisabled))

	pending := renderCompatibilityLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:          models.Mod{ID: "theta", Type: models.MODRINTH},
		DisplayName:  "Theta",
		CompatStatus: changeCompatPending,
	})
	assert.Contains(t, pending, view.PendingIcon(view.ColorDisabled))

	checking := renderCompatibilityLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:          models.Mod{ID: "iota", Type: models.MODRINTH},
		DisplayName:  "Iota",
		CompatStatus: changeCompatChecking,
	})
	assert.Contains(t, checking, ".")

	unsupported := renderCompatibilityLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, DisplayName: "Beta", CompatStatus: changeCompatUnsupported})
	assert.Contains(t, unsupported, "unsupported for 1.20.1")
}

func TestRenderDownloadLineCompatUnsupported(t *testing.T) {
	skipped := renderDownloadLine(changeViewInput{
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	}, changeItem{
		Mod:          models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName:  "Alpha",
		CompatStatus: changeCompatUnsupported,
		Skipped:      true,
	})
	assert.Contains(t, skipped, "unsupported for 1.20.1")

	unsupported := renderDownloadLine(changeViewInput{
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	}, changeItem{
		Mod:          models.Mod{ID: "beta", Type: models.MODRINTH},
		DisplayName:  "Beta",
		CompatStatus: changeCompatUnsupported,
	})
	assert.Contains(t, unsupported, "unsupported for 1.20.1")
}

func TestRenderDownloadLineProgressAndFailure(t *testing.T) {
	inProgress := renderDownloadLine(changeViewInput{
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	}, changeItem{
		Mod:            models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName:    "Alpha",
		DownloadStatus: changeDownloadInProgress,
		Download:       changeDownloadProgress{ratio: 0.5, downloaded: 512, total: 1024},
	})
	assert.Contains(t, inProgress, "50%")

	failed := renderDownloadLine(changeViewInput{
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	}, changeItem{
		Mod:            models.Mod{ID: "beta", Type: models.MODRINTH},
		DisplayName:    "Beta",
		DownloadStatus: changeDownloadFailed,
		ErrorReason:    "boom",
	})
	assert.Contains(t, failed, "download failed: boom")
}

func TestRenderDownloadLineQueued(t *testing.T) {
	line := renderDownloadLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:            models.Mod{ID: "beta", Type: models.MODRINTH},
		DisplayName:    "Beta",
		DownloadStatus: changeDownloadQueued,
	})
	assert.Contains(t, line, view.PendingIcon(view.ColorDisabled))
}

func TestRenderDownloadLineFailedWithoutReason(t *testing.T) {
	line := renderDownloadLine(changeViewInput{
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	}, changeItem{
		Mod:            models.Mod{ID: "beta", Type: models.MODRINTH},
		DisplayName:    "Beta",
		DownloadStatus: changeDownloadFailed,
	})
	assert.Contains(t, line, "Beta (beta) [modrinth]")
}

func TestRenderDownloadLineSuccess(t *testing.T) {
	line := renderDownloadLine(changeViewInput{
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	}, changeItem{
		Mod:            models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName:    "Alpha",
		DownloadStatus: changeDownloadSucceeded,
	})
	assert.Contains(t, line, "Alpha (alpha) [modrinth]")
}

func TestRenderSwitchingLineVariants(t *testing.T) {
	inProgress := renderSwitchingLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:          models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName:  "Alpha",
		SwitchStatus: changeSwitchInProgress,
	})
	assert.Contains(t, inProgress, ". Alpha")

	failed := renderSwitchingLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:          models.Mod{ID: "beta", Type: models.MODRINTH},
		DisplayName:  "Beta",
		SwitchStatus: changeSwitchFailed,
		ErrorReason:  "boom",
	})
	assert.Contains(t, failed, "switch failed: boom")

	succeeded := renderSwitchingLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:          models.Mod{ID: "delta", Type: models.MODRINTH},
		DisplayName:  "Delta",
		SwitchStatus: changeSwitchSucceeded,
	})
	assert.Contains(t, succeeded, "Delta (delta) [modrinth]")

	skipped := renderSwitchingLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:          models.Mod{ID: "gamma", Type: models.MODRINTH},
		DisplayName:  "Gamma",
		SwitchStatus: changeSwitchSkipped,
	})
	assert.Contains(t, skipped, "unsupported for 1.20.1")
}

func TestRenderSwitchingLinePending(t *testing.T) {
	line := renderSwitchingLine(changeViewInput{
		target:       "1.20.1",
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}, changeItem{
		Mod:         models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName: "Alpha",
	})
	assert.Contains(t, line, "[~] Alpha")
}

func TestBuildChangeSectionsSwitchingStage(t *testing.T) {
	items := []changeItem{{
		Mod:            models.Mod{ID: "alpha", Type: models.MODRINTH},
		DisplayName:    "Alpha",
		CompatStatus:   changeCompatSupported,
		DownloadStatus: changeDownloadSucceeded,
	}}
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageSwitching,
		target:    "1.20.1",
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Switching:")
}

func TestBuildChangeSectionsNoopStage(t *testing.T) {
	sections := buildChangeSections(changeViewInput{
		stage:     changeStageNoop,
		target:    "1.20.1",
		colorMode: view.ColorDisabled,
	})
	output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	assert.Contains(t, output, "Already targeting 1.20.1")
}

func TestChangeModelViewRendersHeader(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.bindSender(func(tea.Msg) {})

	viewText := model.View()
	assert.Contains(t, viewText, "Change Minecraft version to 1.19.4")
}

func TestChangeModelViewRendersFullContentOnFinalRender(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:       context.Background(),
		target:    "1.19.4",
		colorMode: view.ColorDisabled,
		items: []changeItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha"},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH}, DisplayName: "Beta"},
		},
		indexByKey: map[string]int{"modrinth:alpha": 0, "modrinth:beta": 1},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})
	model.windowH = 1
	model.finalRender = true

	viewText := model.View()
	assert.Contains(t, viewText, "Compatibility:")
	assert.Contains(t, viewText, "Alpha (alpha) [modrinth]")
	assert.Contains(t, viewText, "Beta (beta) [modrinth]")
}

func TestChangeModelViewportUsesContentHeightWhenWindowHeightZero(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.bindSender(func(tea.Msg) {})

	content := view.RenderViewSections(buildChangeSections(changeViewInput{
		stage:        changeStageRunning,
		target:       "1.19.4",
		items:        model.items,
		colorMode:    view.ColorDisabled,
		spinnerFrame: ".",
	}), view.SectionSeparatorParagraph)

	_ = model.View()
	assert.Equal(t, lipgloss.Height(content), model.viewport.Height)
}

func TestChangeModelViewportClampsToWindowHeight(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.bindSender(func(tea.Msg) {})

	_, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 1})
	_ = model.View()
	assert.Equal(t, 1, model.viewport.Height)
	assert.Equal(t, 80, model.viewport.Width)
}

func TestChangeModelUpdateViewportMessageKeyMovesOffset(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})

	model.windowH = 1
	model.updateViewport("one\ntwo")

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, model.viewport.YOffset)
}

func TestChangeModelUpdateViewportMessageMouseMovesOffset(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})

	model.windowH = 1
	model.updateViewport("one\ntwo")

	_, _ = model.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, model.viewport.YOffset)
}

func TestChangeModelUpdateViewportMessageDefaultReturnsFalse(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})

	_, handled := model.updateViewportMessage(struct{}{})
	assert.False(t, handled)
}

func TestChangeModelUpdateUnknownMessageKeepsState(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})
	model.bindSender(func(tea.Msg) {})

	updated, cmd := model.Update(struct{}{})
	assert.Equal(t, changeStageRunning, updated.(*changeModel).stage)
	assert.Nil(t, cmd)
}

func TestChangeModelUpdateViewportNoWindowClamp(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})

	model.windowH = 10
	model.updateViewport("one\ntwo")
	assert.Equal(t, 2, model.viewport.Height)
}

func TestChangeModelSpinnerFrameHandlesEmptyFrames(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{}
		},
	})
	model.spinner.Spinner.Frames = []string{}
	assert.Equal(t, "", model.spinnerFrame())
}

func TestRenderFinalErrorLineColorEnabled(t *testing.T) {
	line := renderFinalErrorLine(view.ColorEnabled, "Failed")
	assert.Contains(t, line, "Failed")
}

func TestPartitionChangeItems(t *testing.T) {
	items := []changeItem{
		{Mod: models.Mod{ID: "pending"}, CompatStatus: changeCompatPending},
		{Mod: models.Mod{ID: "unsupported"}, CompatStatus: changeCompatUnsupported},
		{Mod: models.Mod{ID: "supported_pending"}, CompatStatus: changeCompatSupported, DownloadStatus: changeDownloadQueued},
		{Mod: models.Mod{ID: "supported_downloading"}, CompatStatus: changeCompatSupported, DownloadStatus: changeDownloadInProgress},
		{Mod: models.Mod{ID: "supported_failed"}, CompatStatus: changeCompatSupported, DownloadStatus: changeDownloadFailed},
		{Mod: models.Mod{ID: "supported_done"}, CompatStatus: changeCompatSupported, DownloadStatus: changeDownloadSucceeded},
		{Mod: models.Mod{ID: "switching"}, CompatStatus: changeCompatSupported, DownloadStatus: changeDownloadQueued, SwitchStatus: changeSwitchInProgress},
	}

	partition := partitionChangeItems(items)
	require.Len(t, partition.compatibility, 2)
	assert.Equal(t, "pending", partition.compatibility[0].Mod.ID)
	assert.Equal(t, "unsupported", partition.compatibility[1].Mod.ID)

	require.Len(t, partition.downloading, 3)
	assert.Equal(t, "supported_pending", partition.downloading[0].Mod.ID)
	assert.Equal(t, "supported_downloading", partition.downloading[1].Mod.ID)
	assert.Equal(t, "supported_failed", partition.downloading[2].Mod.ID)

	require.Len(t, partition.switching, 2)
	assert.Equal(t, "supported_done", partition.switching[0].Mod.ID)
	assert.Equal(t, "switching", partition.switching[1].Mod.ID)
}
