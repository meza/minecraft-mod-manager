package update

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

func TestNewUpdateModelUsesLineSpinnerWhenUnicodeUnsupported(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})
	assert.Equal(t, spinner.Line.Frames[0], model.spinner.StaticFrame())
}

func TestUpdateModelInitQuitsWithoutRunner(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
	})

	msg := model.Init()()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestUpdateModelInitNonTestModeReturnsBatch(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})
	model.bindSender(func(tea.Msg) {})

	msg := model.Init()()
	_, ok := msg.(tea.BatchMsg)
	assert.True(t, ok)
}

func TestStartUpdateCmdHandlesNilRunner(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
	})

	msg := model.startUpdateCmd()()
	typed, ok := msg.(updateExecutionFinishedMsg)
	assert.True(t, ok)
	assert.Equal(t, updateExecutionErrorCanceled, typed.outcome.errType)
}

func TestUpdateModelInitStartsUpdateInTestMode(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})
	model.bindSender(func(tea.Msg) {})

	msg := model.Init()()
	_, ok := msg.(updateExecutionFinishedMsg)
	assert.True(t, ok)
}

func TestUpdateModelUpdateHandlesMessages(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	items := []updateItem{
		{
			ConfigIndex: 0,
			Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			DisplayName: "Alpha",
			Status:      updateItemStatusUpdating,
		},
	}
	model := newUpdateModel(updateModelInput{
		ctx:        ctx,
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: map[int]int{0: 0},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	assert.Nil(t, cmd)
	assert.Equal(t, 80, updated.(*updateModel).windowW)
	assert.Equal(t, 20, updated.(*updateModel).windowH)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.Nil(t, cmd)
	assert.ErrorIs(t, updated.(*updateModel).ctx.Err(), context.Canceled)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.IsType(t, &updateModel{}, updated)
	assert.Nil(t, cmd)

	updated, cmd = model.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	assert.IsType(t, viewport.Model{}, updated.(*updateModel).viewport)
	assert.Nil(t, cmd)

	_, _ = model.Update(spinner.TickMsg{})

	updated, cmd = model.Update(updateItemProgressMsg{
		index: 0,
		progress: updateProgress{
			ratio:      0.25,
			downloaded: 25,
			total:      100,
		},
	})
	assert.NotNil(t, cmd)
	assert.Equal(t, updateItemStatusDownloading, updated.(*updateModel).items[0].Status)
	assert.NotNil(t, updated.(*updateModel).items[0].Progress)

	updated, cmd = model.Update(updateItemStatusMsg{
		index:       0,
		status:      updateItemStatusFailed,
		failReason:  "boom",
		displayName: "New Name",
	})
	assert.NotNil(t, cmd)
	assert.Equal(t, updateItemStatusFailed, updated.(*updateModel).items[0].Status)
	assert.Equal(t, "boom", updated.(*updateModel).items[0].FailReason)
	assert.Equal(t, "New Name", updated.(*updateModel).items[0].DisplayName)
	assert.Nil(t, updated.(*updateModel).items[0].Progress)

	outcome := updateExecutionOutcome{
		items:   updated.(*updateModel).items,
		errType: updateExecutionErrorNone,
	}
	updated, cmd = model.Update(updateExecutionFinishedMsg{outcome: outcome})
	assert.NotNil(t, cmd)
	assert.True(t, updated.(*updateModel).finalRender)

	_, cmd = model.Update(updateFinalizeMsg{})
	assert.NotNil(t, cmd)

	updated, cmd = model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.IsType(t, &updateModel{}, updated)
}

func TestUpdateModelUpdateSkipsZeroWindowSize(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
	})
	model.windowW = 80
	model.windowH = 20

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	assert.Nil(t, cmd)
	assert.Equal(t, 80, updated.(*updateModel).windowW)
	assert.Equal(t, 20, updated.(*updateModel).windowH)
}

func TestUpdateModelUpdateKeepsWindowHeight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{
		{
			ConfigIndex: 0,
			Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			DisplayName: "Alpha",
			Status:      updateItemStatusUpdating,
		},
		{
			ConfigIndex: 1,
			Mod:         models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH},
			DisplayName: "Beta",
			Status:      updateItemStatusUpdating,
		},
	}

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: map[int]int{0: 0, 1: 1},
	})

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 1})
	assert.Nil(t, cmd)
	assert.Equal(t, 1, updated.(*updateModel).windowH)
}

func TestUpdateModelRenderRunningViewReturnsHeaderWhenContentEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
	})
	model.windowH = 10

	output := model.renderRunningView()
	assert.Equal(t, "cmd.update.header", output)
}

func TestUpdateModelRenderRunningViewReturnsContentWhenWindowHeightZero(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{{
		ConfigIndex: 0,
		Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		DisplayName: "Alpha",
		Status:      updateItemStatusUpdating,
	}}

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: map[int]int{0: 0},
	})
	model.windowH = 0

	output := model.renderRunningView()
	assert.Contains(t, output, "cmd.update.header")
	assert.Contains(t, output, "Alpha")
}

func TestUpdateModelRenderRunningViewReturnsViewportWhenHeaderVisible(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{{
		ConfigIndex: 0,
		Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		DisplayName: "Alpha",
		Status:      updateItemStatusUpdating,
	}}

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: map[int]int{0: 0},
	})
	model.windowH = 5

	output := model.renderRunningView()
	assert.Contains(t, output, "cmd.update.header")
	assert.Contains(t, output, "Alpha")
}

func TestUpdateModelRenderRunningViewReturnsHeaderWhenHeaderMissingAndHeightTight(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{{
		ConfigIndex: 0,
		Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		DisplayName: "Alpha",
		Status:      updateItemStatusUpToDate,
	}}

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: map[int]int{0: 0},
	})
	model.windowH = 2

	output := model.renderRunningView()
	assert.Contains(t, output, "cmd.update.header")
	assert.Contains(t, output, "Alpha")
}

func TestUpdateModelRenderRunningViewAddsHeaderWhenHeaderMissingAndSpaceAvailable(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{{
		ConfigIndex: 0,
		Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		DisplayName: "Alpha",
		Status:      updateItemStatusUpToDate,
	}}

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: map[int]int{0: 0},
	})
	model.windowH = 4

	output := model.renderRunningView()
	assert.Contains(t, output, "cmd.update.header")
	assert.Contains(t, output, "cmd.update.section.up_to_date")
}

func TestUpdateModelApplyItemProgressSkipsUnknownIndex(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{{ConfigIndex: 0, Status: updateItemStatusPending}},
		indexByKey: map[int]int{0: 0},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.applyItemProgress(updateItemProgressMsg{
		index: 5,
		progress: updateProgress{
			ratio:      0.3,
			downloaded: 30,
			total:      100,
		},
	})

	assert.Equal(t, updateItemStatusPending, model.items[0].Status)
}

func TestUpdateModelViewBranches(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.finalRender = true
	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorWriteLock, lockPath: "/lock"}
	assert.Contains(t, model.View(), "cmd.update.error.write_lock")

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorWriteConfig, configPath: "/config"}
	assert.Contains(t, model.View(), "cmd.update.error.write_config")

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorCanceled}
	assert.Equal(t, "", model.View())

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorUnknown, err: errors.New("boom")}
	assert.Contains(t, model.View(), "boom")

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorNone, items: []updateItem{}}
	assert.Contains(t, model.View(), "cmd.update.summary.success")

	model.suppressFinal = true
	assert.Equal(t, "", model.View())

	model.finalRender = false
	model.windowH = 0
	model.items = []updateItem{{ConfigIndex: 0, DisplayName: "Alpha", Status: updateItemStatusUpdating}}
	assert.Contains(t, model.View(), "cmd.update.header")
}

func TestUpdateViewportClampsNegativeHeight(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.updateViewport("line1\nline2", -1)
	assert.Equal(t, 0, model.viewport.Height)
}

func TestUpdateModelViewUsesViewportWhenWindowHeightSet(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
		items: []updateItem{
			{
				ConfigIndex: 0,
				Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
				DisplayName: "Alpha",
				Status:      updateItemStatusUpdating,
			},
		},
		indexByKey: map[int]int{0: 0},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.windowH = 5
	model.windowW = 42

	output := model.View()
	assert.NotEmpty(t, output)
	assert.Greater(t, model.viewport.Height, 0)
	assert.Equal(t, model.windowW, model.viewport.Width)
}

func TestUpdateModelViewReturnsHeaderWhenNoContent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.windowH = 10
	output := model.View()
	assert.Contains(t, output, "cmd.update.header")
	assert.NotContains(t, output, "cmd.update.section.updating")
}

func TestUpdateModelViewReturnsHeaderWhenViewportTooShort(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
		items: []updateItem{
			{ConfigIndex: 0, DisplayName: "Alpha", Status: updateItemStatusUpdating},
		},
		indexByKey: map[int]int{0: 0},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.windowH = 1
	output := model.View()
	assert.Contains(t, output, "cmd.update.header")
}

func TestUpdateViewportAutoScrollsUntilUserScrolls(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.updateViewport("one\ntwo\nthree\nfour", 2)
	assert.Equal(t, 2, model.viewport.Height)
	assert.Equal(t, 2, model.viewport.YOffset)

	model.userScrolled = true
	model.viewport.YOffset = 0
	model.updateViewport("one\ntwo\nthree\nfour", 2)
	assert.Equal(t, 0, model.viewport.YOffset)
}

func TestUpdateViewportUsesWindowHeight(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	model.windowW = 10
	model.updateViewport("one", 5)
	assert.Equal(t, 5, model.viewport.Height)
	assert.Equal(t, 0, model.viewport.YOffset)
}

func TestUpdateRenderWithStickyHeaderSkipsWhenWindowHeightZero(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
	})
	model.windowH = 0

	content := "cmd.update.header\nAlpha"
	expected := view.RenderViewSections([]string{content, "cmd.update.header"}, view.SectionSeparatorParagraph)
	normalizedExpected := terminal.NormalizeOutput(expected, terminal.NormalizeOptions{TrimTrailingWhitespace: true})
	assert.Equal(t, normalizedExpected, model.renderWithStickyHeader(content))
}

func TestUpdateRenderWithStickyHeaderSkipsWhenContentEmpty(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
	})
	model.windowH = 10

	assert.Equal(t, "", model.renderWithStickyHeader(""))
}

func TestUpdateRenderWithStickyHeaderSkipsWhenBodyEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
	})
	model.windowH = 10

	content := "cmd.update.header"
	assert.Equal(t, content, model.renderWithStickyHeader(content))
}

func TestUpdateRenderWithStickyHeaderEchoesWhenContentExceedsWindow(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
	})
	model.windowH = 2

	content := "cmd.update.header\nAlpha\nBeta"
	expected := view.RenderViewSections([]string{content, "cmd.update.header"}, view.SectionSeparatorParagraph)
	normalizedExpected := terminal.NormalizeOutput(expected, terminal.NormalizeOptions{TrimTrailingWhitespace: true})
	normalizedActual := terminal.NormalizeOutput(model.renderWithStickyHeader(content), terminal.NormalizeOptions{TrimTrailingWhitespace: true})
	assert.Equal(t, normalizedExpected, normalizedActual)
}

func TestUpdateRenderWithStickyHeaderSkipsWhenHeaderMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateModel(updateModelInput{
		ctx:       context.Background(),
		colorMode: view.ColorDisabled,
	})
	model.windowH = 10

	content := "Other Header\nAlpha"
	assert.Equal(t, content, model.renderWithStickyHeader(content))
}

func TestSplitHeaderFromContent(t *testing.T) {
	header, body := splitHeaderFromContent("line-one\nline-two")
	assert.Equal(t, "line-one", header)
	assert.Equal(t, "line-two", body)

	header, body = splitHeaderFromContent("line-one")
	assert.Equal(t, "line-one", header)
	assert.Equal(t, "", body)
}

func TestUpdateModelMarksUserScrolledOnViewportChange(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	assert.False(t, model.userScrolled)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.True(t, updated.(*updateModel).userScrolled)
}

func TestUpdateModelMarksUserScrolledOnVimKeys(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.True(t, updated.(*updateModel).userScrolled)
}

func TestIsViewportScrollKey(t *testing.T) {
	assert.True(t, isViewportScrollKey(tea.KeyMsg{Type: tea.KeyDown}))
	assert.True(t, isViewportScrollKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}))
	assert.False(t, isViewportScrollKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}))
}

func TestIsViewportScrollMouse(t *testing.T) {
	assert.True(t, isViewportScrollMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}))
	assert.False(t, isViewportScrollMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}))
	assert.False(t, isViewportScrollMouse(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonWheelDown}))
}

func TestApplyItemUpdateIgnoresUnknownIndex(t *testing.T) {
	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []updateItem{},
		indexByKey: map[int]int{},
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome { return updateExecutionOutcome{} },
	})
	model.applyItemUpdate(updateItemStatusMsg{index: 99, status: updateItemStatusUpdated})
}
